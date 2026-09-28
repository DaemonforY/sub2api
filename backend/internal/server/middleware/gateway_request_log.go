package middleware

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GatewayRequestLogRecorder is the sink used by GatewayRequestLog.
type GatewayRequestLogRecorder interface {
	Enabled() bool
	Record(entry *service.GatewayRequestLog)
}

var gatewayRequestLogRecorder atomic.Pointer[gatewayRequestLogRecorderHolder]

type gatewayRequestLogRecorderHolder struct{ r GatewayRequestLogRecorder }

// SetGatewayRequestLogRecorder installs the recorder (called once when the router is built).
func SetGatewayRequestLogRecorder(r GatewayRequestLogRecorder) {
	if r == nil {
		gatewayRequestLogRecorder.Store(nil)
		return
	}
	gatewayRequestLogRecorder.Store(&gatewayRequestLogRecorderHolder{r: r})
}

// Gateway path prefixes (route groups) and top-level gateway aliases.
var (
	gatewayRequestLogPrefixes = []string{"/v1", "/v1beta", "/backend-api", "/antigravity"}
	gatewayRequestLogAliases  = []string{"/responses", "/models", "/chat", "/embeddings", "/images", "/messages", "/alpha"}
)

func pathHasSegmentPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// isGatewayRequestLogPath decides whether a request belongs to the API gateway.
// Web-app API (/api/...) and static assets are excluded; any other request that
// carries an API-key credential is included so mistyped endpoints are visible too.
func isGatewayRequestLogPath(c *gin.Context) bool {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api/") || path == "/api" || path == "/health" {
		return false
	}
	for _, p := range gatewayRequestLogPrefixes {
		if pathHasSegmentPrefix(path, p) {
			return true
		}
	}
	for _, p := range gatewayRequestLogAliases {
		if pathHasSegmentPrefix(path, p) {
			return true
		}
	}
	return hasAPIKeyCredentialInput(c) || strings.TrimSpace(c.Query("key")) != ""
}

// presentedAPIKeyCredential returns the raw credential the caller sent, using the same
// sources the auth middlewares accept (Bearer, x-api-key, x-goog-api-key, ?key=).
func presentedAPIKeyCredential(c *gin.Context) string {
	if auth := strings.TrimSpace(c.GetHeader("Authorization")); auth != "" {
		if parts := strings.SplitN(auth, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if k := strings.TrimSpace(parts[1]); k != "" {
				return k
			}
		}
	}
	for _, h := range []string{"x-api-key", "x-goog-api-key"} {
		if k := strings.TrimSpace(c.GetHeader(h)); k != "" {
			return k
		}
	}
	return strings.TrimSpace(c.Query("key"))
}

func requestFullURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); proto == "http" || proto == "https" {
		scheme = proto
	}
	return scheme + "://" + c.Request.Host + c.Request.URL.RequestURI()
}

func contextInt64(c *gin.Context, key string) *int64 {
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case int64:
		if n > 0 {
			return &n
		}
	case int:
		if n > 0 {
			out := int64(n)
			return &out
		}
	}
	return nil
}

// GatewayRequestLog records every API gateway request (success or failure) with its
// full URL and the API key presented. Registered globally so rejected requests and
// unmatched routes are captured as well; writes are asynchronous and never block.
func GatewayRequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		holder := gatewayRequestLogRecorder.Load()
		if holder == nil || holder.r == nil || !holder.r.Enabled() ||
			c.Request.Method == "OPTIONS" || !isGatewayRequestLogPath(c) {
			c.Next()
			return
		}

		start := time.Now()
		// Capture before handlers run: some handlers rewrite the path/query.
		method := c.Request.Method
		fullURL := requestFullURL(c)
		path := c.Request.URL.Path
		presentedKey := presentedAPIKeyCredential(c)

		c.Next()

		status := c.Writer.Status()
		entry := &service.GatewayRequestLog{
			CreatedAt:  start.UTC(),
			Method:     method,
			URL:        fullURL,
			Path:       path,
			StatusCode: status,
			Success:    status > 0 && status < 400,
			DurationMs: time.Since(start).Milliseconds(),
			APIKey:     presentedKey,
			Model:      strings.TrimSpace(c.GetString(service.OpsModelContextKey)),
			AccountID:  contextInt64(c, service.OpsAccountIDContextKey),
			ClientIP:   strings.TrimSpace(ip.GetClientIP(c)),
			UserAgent:  c.GetHeader("User-Agent"),
		}
		if reqID, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
			entry.RequestID = reqID
		}
		if reason, ok := GetIngressRejectReason(c); ok {
			entry.ErrorCode = string(reason)
		}

		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey == nil {
			apiKey, ok = GetOpsFallbackAPIKey(c)
		}
		if ok && apiKey != nil {
			if apiKey.Key != "" {
				entry.APIKey = apiKey.Key
			}
			if apiKey.ID > 0 {
				id := apiKey.ID
				entry.APIKeyID = &id
			}
			if apiKey.UserID > 0 {
				uid := apiKey.UserID
				entry.UserID = &uid
			}
			entry.GroupID = apiKey.GroupID
		}

		holder.r.Record(entry)
	}
}
