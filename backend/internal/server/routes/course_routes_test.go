package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// /settings sits beside /:id in the admin course routes; registering both must not panic and
// both must resolve.
func TestCourseAdminRoutesRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Course: admin.NewCourseHandler(nil)}}
	require.NotPanics(t, func() { registerCourseAdminRoutes(r.Group("/api/v1/admin"), h) })
	paths := map[string]bool{}
	for _, rt := range r.Routes() {
		paths[rt.Method+" "+rt.Path] = true
	}
	for _, want := range []string{"GET /api/v1/admin/courses/settings", "PUT /api/v1/admin/courses/settings", "GET /api/v1/admin/courses/:id", "PUT /api/v1/admin/courses/:id"} {
		require.True(t, paths[want], want)
	}
}
