package routes

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// apiPathsWithoutV1 are OpenAI / Anthropic endpoints that clients reach when the base URL lacks /v1.
// /chat/completions, /embeddings, /responses, /models and /images/* have aliases in gateway.go and
// never get here (with the right method).
var apiPathsWithoutV1 = []string{"/chat/completions", "/completions", "/embeddings", "/messages", "/models", "/responses"}

// RegisterAPINotFound answers unknown API paths with a JSON error that says how to fix the address,
// instead of gin's plain "404 page not found". Site pages never get here: the embedded frontend
// serves them before routing.
func RegisterAPINotFound(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		code, message := apiNotFoundMessage(c.Request.Method, c.Request.URL.Path, requestOrigin(c.Request))
		c.JSON(http.StatusNotFound, gin.H{"code": code, "message": message})
	})
}

func apiNotFoundMessage(method, path, origin string) (string, string) {
	base := origin + "/v1"
	switch {
	case path == "/v1" || path == "/v1/":
		return "API_BASE_URL", "这是接口的 Base URL，不能直接打开：把 " + base + " 填进工具的 API 地址，对话请求发到 POST " + base + "/chat/completions" +
			"（This is the API base URL; send chat requests to POST " + base + "/chat/completions）"
	case strings.HasPrefix(path, "/v1/v1/"):
		return "API_PATH_DUPLICATED_V1", "接口地址多了一个 /v1（" + path + "）：工具会自己在 Base URL 后面加 /v1，请把 API 地址改成 " + origin +
			"（Duplicated /v1 in the API path; set the base URL to " + origin + "）"
	case !strings.HasPrefix(path, "/v1/") && isAPIPathWithoutV1(path):
		return "API_PATH_MISSING_V1", "接口地址少了 /v1：请把 Base URL 填成 " + base + "，完整地址是 " + method + " " + base + path +
			"（Missing /v1 in the API path; set the base URL to " + base + "）"
	default:
		return "API_NOT_FOUND", "接口不存在：" + method + " " + path + "。对话接口是 POST " + base + "/chat/completions，可用模型用 GET " + base + "/models 查看" +
			"（Unknown API endpoint " + method + " " + path + "）"
	}
}

func isAPIPathWithoutV1(path string) bool {
	if strings.HasPrefix(path, "/audio/") || strings.HasPrefix(path, "/chat/") || strings.HasPrefix(path, "/messages/") {
		return true
	}
	for _, p := range apiPathsWithoutV1 {
		if path == p {
			return true
		}
	}
	return false
}

// requestOrigin is the browser-visible origin (TLS ends at the proxy).
func requestOrigin(r *http.Request) string {
	scheme := "http"
	if proto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])); proto == "https" || proto == "http" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
