package web

import (
	"net/http"
	"net/url"
	"strings"
)

func (h *handler) cspHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imgSources := []string{"'self'", "data:"}
		if logoOrigin := cspImageOrigin(h.config.AppLogoURL); logoOrigin != "" {
			imgSources = append(imgSources, logoOrigin)
		}
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self' 'wasm-unsafe-eval' blob: https://cdn.jsdelivr.net https://*.duckdb.org; style-src 'self' 'unsafe-inline' blob:; img-src "+strings.Join(imgSources, " ")+";",
		)
		next.ServeHTTP(w, r)
	})
}

func cspImageOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
