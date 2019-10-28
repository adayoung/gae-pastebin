package utils

import (
	"strings"
)

func getCSP(staticDomain string) string {
	var CSP = map[string][]string{
		"default-src": {
			staticDomain,
		},
		"font-src": {
			"stackpath.bootstrapcdn.com",
			"fonts.gstatic.com",
		},
		"script-src": {
			"code.jquery.com",
			"stackpath.bootstrapcdn.com",
			"www.google-analytics.com",
			"cdnjs.cloudflare.com",
			"https://www.google.com/recaptcha/",
			"https://www.gstatic.com/recaptcha/",
			staticDomain,
		},
		"style-src": {
			"'unsafe-inline'",
			"stackpath.bootstrapcdn.com",
			"cdnjs.cloudflare.com",
			staticDomain,
		},
		"img-src": {
			"'self'",
			"data:",
			"www.google-analytics.com",
			"stats.g.doubleclick.net",
			staticDomain,
		},
		"connect-src": {
			"'self'",
			"*.googleusercontent.com",
		},
		"frame-src": {
			"'self'",
			"blob:",
			"https://www.google.com/recaptcha/",
		},
		"frame-ancestors": {
			"'none'",
		},
	}

	var policy []string
	for k, v := range CSP {
		policy = append(policy, k+" "+strings.Join(v, " "))
	}

	return strings.Join(policy, "; ")
}
