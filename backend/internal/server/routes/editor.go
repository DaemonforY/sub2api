package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterEditorRoutes registers the 公众号 editor's API (/editor): WeChat draft box, article
// import, the WeChat image proxy and AI. Formatting itself needs no server; all of these need login.
func RegisterEditorRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Editor == nil {
		return
	}
	g := v1.Group("/editor")
	g.Use(gin.HandlerFunc(jwtAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.POST("/wechat/check", panelRateLimiter.Heavy(), h.Editor.WechatCheck)
		g.POST("/wechat/upload", h.Editor.WechatUpload)
		g.POST("/wechat/draft", panelRateLimiter.Heavy(), h.Editor.WechatDraft)
		g.POST("/article/import", panelRateLimiter.Heavy(), h.Editor.ImportArticle)
		g.GET("/image", h.Editor.Image)
		g.POST("/ai/text", panelRateLimiter.Heavy(), h.Editor.AIText)
		g.POST("/ai/image", panelRateLimiter.Heavy(), h.Editor.AIImage)
	}
}
