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
			"cdn.jsdelivr.net",
			"fonts.gstatic.com",
		},
		"script-src": {
			"code.jquery.com",
			"cdn.jsdelivr.net",
			"cdnjs.cloudflare.com",
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
		"img-src": {
			"'self'",
			"data:",
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
