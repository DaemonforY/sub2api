package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// wechatDomainVerifyFileSource returns the content of a 公众号 domain verify file.
type wechatDomainVerifyFileSource interface {
	GetWeChatDomainVerifyFile(ctx context.Context, filename string) (string, bool)
}

// wechatDomainVerifyFiles serves the 公众号 domain verify files (/MP_verify_xxx.txt, set in 系统设置)
// at the site root, before the SPA fallback answers the path with index.html.
func wechatDomainVerifyFiles(source wechatDomainVerifyFileSource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if source == nil || (c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) {
			c.Next()
			return
		}
		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if !service.IsWeChatDomainVerifyFilename(name) {
			c.Next()
			return
		}
		content, ok := source.GetWeChatDomainVerifyFile(c.Request.Context(), name)
		if !ok {
			c.Next()
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
		c.Abort()
	}
}
