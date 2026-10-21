//go:build embed || unit

package web

import (
	"net/http"
	"path"
	"strings"
)

// Vite emits content-hashed filenames under assets/, so the backend can apply
// immutable caching without relying on a reverse proxy to classify paths.
const staticAssetsCacheControl = "public, max-age=31536000, immutable"

// isFingerprintedEmbeddedAssetPath reports whether a cleaned URL path refers to
// a Vite asset whose filename contains the default eight-character build hash.
func isFingerprintedEmbeddedAssetPath(cleanPath string) bool {
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	if !strings.HasPrefix(cleanPath, "assets/") {
		return false
	}

	filename := path.Base(cleanPath)
	extension := path.Ext(filename)
	stem := strings.TrimSuffix(filename, extension)
	const fingerprintLength = 8
	delimiterIndex := len(stem) - fingerprintLength - 1
	if extension == "" || delimiterIndex < 1 || stem[delimiterIndex] != '-' {
		return false
	}

	// Vite hashes use URL-safe characters and are stable for immutable caching.
	fingerprint := stem[delimiterIndex+1:]
	for _, char := range fingerprint {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// applyStaticAssetCacheHeaders sets Cache-Control for long-cacheable static paths.
// index.html / SPA routes must keep no-cache and are not handled here.
func applyStaticAssetCacheHeaders(header http.Header, cleanPath string) {
	if header == nil || !isFingerprintedEmbeddedAssetPath(cleanPath) {
		return
	}
	header.Set("Cache-Control", staticAssetsCacheControl)
}

// installScripts are the one-click setup scripts in frontend/public, fetched with
// `curl | bash` and `irm | iex`. Windows PowerShell 5.1 decodes the body with the
// charset in Content-Type (ISO-8859-1 without one), so they are always served as
// UTF-8 text, and never cached so a fix reaches the next install right away.
var installScripts = map[string]bool{
	"install.sh":        true,
	"install.ps1":       true,
	"hivegpt-setup.mjs": true,
}

func applyInstallScriptHeaders(header http.Header, cleanPath string) {
	if header == nil || !installScripts[strings.TrimPrefix(cleanPath, "/")] {
		return
	}
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
}
