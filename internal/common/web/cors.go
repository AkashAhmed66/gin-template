package web

import (
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS builds a CORS middleware from the typed config. Uses pattern matching
// for AllowedOrigins (so http://localhost:* matches any port) — required when
// AllowCredentials is true.
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	c := cors.Config{
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		ExposeHeaders:    cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
		AllowOriginFunc: func(origin string) bool {
			return matchOriginPatterns(origin, cfg.AllowedOrigins)
		},
	}
	return cors.New(c)
}

// matchOriginPatterns supports wildcards in either the host or the port
// position, e.g. "http://localhost:*", "https://*.example.com".
func matchOriginPatterns(origin string, patterns []string) bool {
	for _, p := range patterns {
		if p == "*" || p == origin {
			return true
		}
		if matchWildcard(origin, p) {
			return true
		}
	}
	return false
}

func matchWildcard(s, pattern string) bool {
	if pattern == "" {
		return false
	}
	// Simple greedy match: split by "*" and check each chunk appears in order.
	parts := splitN(pattern, '*')
	if len(parts) == 1 {
		return s == pattern
	}
	if !startsWith(s, parts[0]) {
		return false
	}
	pos := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		idx := indexFrom(s, parts[i], pos)
		if idx < 0 {
			return false
		}
		pos = idx + len(parts[i])
	}
	last := parts[len(parts)-1]
	return endsWithFrom(s, last, pos)
}

func splitN(s string, sep rune) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == sep {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func endsWithFrom(s, suffix string, fromPos int) bool {
	if len(s)-fromPos < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}

func indexFrom(s, sub string, from int) int {
	if from < 0 || from > len(s) {
		return -1
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
