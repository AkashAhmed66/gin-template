package auth

import "strings"

// deviceNameFromUserAgent renders a short, human-friendly label like
// "Chrome on Windows" or "curl" by string-sniffing the User-Agent header.
// Returns "" when the header is empty so callers can store nothing rather
// than a misleading default.
//
// This is intentionally lightweight (no UA parsing library): the goal is just
// to give the session list a recognizable label, not perfect UA classification.
// Misses fall through to "Browser" or the OS alone, which is fine for the
// "log me out from device X" UX.
func deviceNameFromUserAgent(userAgent string) string {
	if userAgent == "" {
		return ""
	}
	ua := strings.ToLower(userAgent)

	// Non-browser clients — these usually identify themselves cleanly.
	switch {
	case strings.HasPrefix(ua, "curl/"):
		return "curl"
	case strings.Contains(ua, "postmanruntime"):
		return "Postman"
	case strings.Contains(ua, "insomnia"):
		return "Insomnia"
	case strings.Contains(ua, "httpie"):
		return "HTTPie"
	case strings.HasPrefix(ua, "wget/"):
		return "wget"
	case strings.HasPrefix(ua, "go-http-client/"):
		return "Go client"
	}

	// Browser — checked in order because UAs lie about each other
	// (Edge says "Chrome", Chrome says "Safari", etc.).
	browser := ""
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browser = "Opera"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "chrome/"):
		browser = "Chrome"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	}

	// OS family.
	os := ""
	switch {
	case strings.Contains(ua, "windows"):
		os = "Windows"
	case strings.Contains(ua, "iphone"):
		os = "iPhone"
	case strings.Contains(ua, "ipad"):
		os = "iPad"
	case strings.Contains(ua, "android"):
		os = "Android"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		os = "macOS"
	case strings.Contains(ua, "linux"):
		os = "Linux"
	}

	switch {
	case browser != "" && os != "":
		return browser + " on " + os
	case browser != "":
		return browser
	case os != "":
		return os
	}
	// Last-resort fallback: first 64 chars of the raw UA so the session row
	// still has *something* identifying. Truncated to avoid filling the column.
	if len(userAgent) > 64 {
		return userAgent[:64]
	}
	return userAgent
}
