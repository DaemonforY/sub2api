package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

var corsWarningOnce sync.Once

const canvasCredentialPathPrefix = "/api/v1/canvas/"

// CORS 跨域中间件
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := normalizeOrigins(cfg.AllowedOrigins)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}
	wildcardWithSpecific := allowAll && len(allowedOrigins) > 1
	if wildcardWithSpecific {
		allowedOrigins = []string{"*"}
	}
	allowCredentials := cfg.AllowCredentials

	corsWarningOnce.Do(func() {
		if len(allowedOrigins) == 0 {
			log.Println("Warning: CORS allowed_origins not configured; cross-origin requests will be rejected.")
		}
		if wildcardWithSpecific {
			log.Println("Warning: CORS allowed_origins includes '*'; wildcard will take precedence over explicit origins.")
		}
		if allowAll && allowCredentials {
			log.Println("Warning: CORS allowed_origins set to '*', disabling allow_credentials.")
		}
	})
	if allowAll && allowCredentials {
		allowCredentials = false
	}

	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "" || origin == "*" {
			continue
		}
		allowedSet[origin] = struct{}{}
	}
	allowHeaders := []string{
		"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization",
		"accept", "origin", "Cache-Control", "X-Requested-With", "X-API-Key", "X-Admin-UI-Request", "X-User-UI-Request", CanvasRequestHeader,
	}
	// OpenAI Node SDK 会发送 x-stainless-* 请求头，需在 CORS 中显式放行。
	openAIProperties := []string{
		"lang", "package-version", "os", "arch", "retry-count", "runtime",
		"runtime-version", "async", "helper-method", "poll-helper", "custom-poll-interval", "timeout",
	}
	for _, prop := range openAIProperties {
		allowHeaders = append(allowHeaders, "x-stainless-"+prop)
	}
	allowHeadersValue := strings.Join(allowHeaders, ", ")

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		originAllowed := allowAll
		if origin != "" && !allowAll {
			_, originAllowed = allowedSet[origin]
		}
		if origin != "" && !originAllowed && isPublicGatewayPath(c.Request.URL.Path) {
			gatewayCORS(c, allowHeadersValue)
			return
		}

		if originAllowed {
			if allowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Add("Vary", "Origin")
			}
			// The canvas session endpoints are cookie-authenticated: explicitly allowed origins may
			// send credentials there even when credentials are off for the rest of the API.
			if allowCredentials || (!allowAll && origin != "" && strings.HasPrefix(c.Request.URL.Path, canvasCredentialPathPrefix)) {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeadersValue)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "ETag, Server-Timing")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}
		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			if originAllowed {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}

		c.Next()
	}
}

func normalizeOrigins(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

// isPublicGatewayPath is the API-key-authenticated gateway (/v1, /v1beta, root aliases such as
// /chat/completions). Browser apps — Obsidian plugins, web chat front ends — call it directly.
func isPublicGatewayPath(path string) bool {
	return IsCompatErrorPath(path) || path == "/v1beta" || strings.HasPrefix(path, "/v1beta/")
}

// gatewayExposeHeaders are the gateway response headers browser clients read (request id for
// support, Retry-After and rate-limit headers for SDK backoff).
const gatewayExposeHeaders = "ETag, Server-Timing, X-Request-Id, Retry-After, " +
	"X-RateLimit-Limit-Requests, X-RateLimit-Remaining-Requests, X-RateLimit-Reset-Requests, " +
	"X-RateLimit-Limit-Tokens, X-RateLimit-Remaining-Tokens, X-RateLimit-Reset-Tokens"

// gatewayCORS answers any origin on gateway paths. The gateway authenticates with the API key in
// a header, never a cookie, so "*" without credentials exposes nothing a page could not already
// send itself. Requested headers are echoed (SDKs add their own: x-stainless-*, anthropic-*,
// openai-*) once they are checked to be header names.
func gatewayCORS(c *gin.Context, defaultHeaders string) {
	h := c.Writer.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	allowHeaders := defaultHeaders
	if requested := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers")); requested != "" && isHeaderNameList(requested) {
		allowHeaders = requested
	}
	h.Set("Access-Control-Allow-Headers", allowHeaders)
	h.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
	h.Set("Access-Control-Expose-Headers", gatewayExposeHeaders)
	h.Set("Access-Control-Max-Age", "86400")
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

func isHeaderNameList(v string) bool {
	if len(v) > 2048 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == ',', r == ' ':
		default:
			return false
		}
	}
	return true
}
