package utils

import (
	// Go Builtin Packages
	"net/http"
	"strings"
)

// http://andyrees.github.io/2015/your-code-a-mess-maybe-its-time-to-bring-in-the-decorators/
func ExtraSugar(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src netdna.bootstrapcdn.com fonts.gstatic.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com www.google-analytics.com; style-src 'self' netdna.bootstrapcdn.com 'unsafe-inline'; img-src 'self' *; object-src 'none'; media-src 'none'")
		// w.Header().Set("Strict-Transport-Security", "max-age=15552000")

		if strings.Contains(strings.ToLower(r.UserAgent()), "msie") {
			w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		f(w, r)
	}
}
