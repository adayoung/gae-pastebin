package utils

import (
	"strings"
)

func getCSP(staticDomain string) string {
	var CSP = map[string][]string{
		"connect-src": {
			"'self'",
			"cloudflareinsights.com",
			"*.googleusercontent.com",
		},
		"default-src": {
			staticDomain,
		},
		"img-src": {
			"'self'",
			"data:",
			staticDomain,
		},
		"frame-ancestors": {
			"'none'",
		},
		"frame-src": {
			"'self'",
			"blob:",
			"https://www.google.com/recaptcha/",
		},
		"font-src": {
			"cdn.jsdelivr.net",
			"fonts.gstatic.com",
		},
		"script-src": {
			"code.jquery.com",
			"cdn.jsdelivr.net",
			"cdnjs.cloudflare.com",
			"static.cloudflareinsights.com",
			"https://www.google.com/recaptcha/",
			"https://www.gstatic.com/recaptcha/",
			staticDomain,
		},
		"style-src": {
			"'unsafe-inline'",
			"cdn.jsdelivr.net",
			"cdnjs.cloudflare.com",
			staticDomain,
		},
	}

	var policy []string
	for k, v := range CSP {
		policy = append(policy, k+" "+strings.Join(v, " "))
	}

	return strings.Join(policy, "; ")
}
