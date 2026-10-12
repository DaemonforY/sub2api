package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// imageFileTypes: the only files local image storage writes (see extensionForContentType).
var imageFileTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// ServeImageFile serves async image results kept on this server's disk (local image storage):
// GET /api/v1/image-files/*key. Public like an object-store link — the key holds the random task
// ID — and readable from any origin (the canvas fetches the result to keep a copy).
func ServeImageFile(dir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimPrefix(c.Param("key"), "/")
		ext := strings.ToLower(filepath.Ext(key))
		contentType, ok := imageFileTypes[ext]
		path := service.LocalImagePath(dir, key)
		if !ok || path == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			c.Header("Cache-Control", "no-store")
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Cache-Control", "public, max-age=86400, immutable")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Type", contentType)
		c.File(path)
	}
}
