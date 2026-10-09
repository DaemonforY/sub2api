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

		// AI 写文章: background runs (outline → article + pictures) on the user's own key.
		a := g.Group("/articles")
		a.GET("/config", h.Editor.ArticleConfig)
		a.GET("", h.Editor.ArticleList)
		a.POST("", panelRateLimiter.Heavy(), h.Editor.ArticleCreate)
		a.GET("/:id", h.Editor.ArticleGet)
		a.POST("/:id/outline", panelRateLimiter.Heavy(), h.Editor.ArticleOutline)
		a.POST("/:id/retry", panelRateLimiter.Heavy(), h.Editor.ArticleRetry)
		a.POST("/:id/cancel", h.Editor.ArticleCancel)
		a.POST("/:id/pushed", h.Editor.ArticlePushed)
		a.DELETE("/:id", h.Editor.ArticleDelete)
		a.GET("/:id/images/:n", h.Editor.ArticleImage)
	}
}
