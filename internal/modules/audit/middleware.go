package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/gin-gonic/gin"
)

const (
	ctxAction       = "audit.action"
	ctxResourceType = "audit.resource_type"
	ctxResourceID   = "audit.resource_id"
	ctxSkip         = "audit.skip"
)

// Action attaches metadata to the current request that the audit middleware
// will copy into the row. Call from handlers via web.Handler before returning.
//
//	web.Handler(func(c *gin.Context) (*dto.ApiResponse, error) {
//	    audit.Action(c, "ORDER_CREATE", "Order", "")
//	    ...
//	})
func Action(c *gin.Context, action, resourceType, resourceID string) {
	if action != "" {
		c.Set(ctxAction, action)
	}
	if resourceType != "" {
		c.Set(ctxResourceType, resourceType)
	}
	if resourceID != "" {
		c.Set(ctxResourceID, resourceID)
	}
}

// Skip opts the current request out of audit capture.
func Skip(c *gin.Context) { c.Set(ctxSkip, true) }

// Middleware captures one row per request and dispatches it to the async
// service. Skips paths matching ExcludePatterns or outside IncludePrefixes.
func Middleware(svc Service, cfg config.AuditConfig) gin.HandlerFunc {
	masker := newBodyMasker(cfg.MaskedFields, cfg.MaxBodyLength)
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if !pathIncluded(path, cfg.IncludePrefixes) || pathExcluded(path, cfg.ExcludePatterns) {
			c.Next()
			return
		}

		start := time.Now()

		var reqBody []byte
		if cfg.CaptureRequestBody && c.Request.Body != nil {
			b, err := io.ReadAll(c.Request.Body)
			if err == nil {
				reqBody = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		var respBuf *bytes.Buffer
		if cfg.CaptureResponseBody {
			respBuf = &bytes.Buffer{}
			c.Writer = &responseCapturer{ResponseWriter: c.Writer, buf: respBuf}
		}

		c.Next()

		if v, ok := c.Get(ctxSkip); ok {
			if b, _ := v.(bool); b {
				return
			}
		}

		entry := Log{
			RequestID:   web.GetRequestID(c),
			Timestamp:   start.UTC(),
			DurationMs:  time.Since(start).Milliseconds(),
			Method:      c.Request.Method,
			Path:        path,
			QueryString: c.Request.URL.RawQuery,
			StatusCode:  c.Writer.Status(),
			ClientIP:    c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
		}
		if v, ok := c.Get(security.CtxUserID); ok {
			if u, ok := v.(uint64); ok {
				entry.UserID = u
			}
		}
		if v, ok := c.Get(security.CtxUsername); ok {
			if s, ok := v.(string); ok {
				entry.Username = s
			}
		}
		if v, ok := c.Get(ctxAction); ok {
			if s, ok := v.(string); ok {
				entry.Action = s
			}
		}
		if v, ok := c.Get(ctxResourceType); ok {
			if s, ok := v.(string); ok {
				entry.ResourceType = s
			}
		}
		if v, ok := c.Get(ctxResourceID); ok {
			if s, ok := v.(string); ok {
				entry.ResourceID = s
			}
		}
		if reqBody != nil {
			entry.RequestBody = masker.MaskAndTruncate(reqBody)
		}
		if respBuf != nil {
			entry.ResponseBody = masker.MaskAndTruncate(respBuf.Bytes())
		}
		if len(c.Errors) > 0 {
			entry.ErrorMessage = c.Errors.String()
		}

		svc.Record(entry)
	}
}

type responseCapturer struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (r *responseCapturer) Write(p []byte) (int, error) {
	r.buf.Write(p)
	return r.ResponseWriter.Write(p)
}

func (r *responseCapturer) WriteString(s string) (int, error) {
	r.buf.WriteString(s)
	return r.ResponseWriter.WriteString(s)
}

func pathIncluded(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func pathExcluded(path string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// bodyMasker walks JSON and replaces sensitive field values with "***".
type bodyMasker struct {
	fields    map[string]struct{}
	maxLength int
}

func newBodyMasker(fields []string, maxLen int) *bodyMasker {
	m := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		m[strings.ToLower(f)] = struct{}{}
	}
	if maxLen <= 0 {
		maxLen = 10000
	}
	return &bodyMasker{fields: m, maxLength: maxLen}
}

func (b *bodyMasker) MaskAndTruncate(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	masked := b.maskJSON(body)
	if len(masked) > b.maxLength {
		return masked[:b.maxLength] + "...[truncated]"
	}
	return masked
}

func (b *bodyMasker) maskJSON(body []byte) string {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body) // not JSON — store raw
	}
	masked := b.walk(v)
	out, err := json.Marshal(masked)
	if err != nil {
		return string(body)
	}
	return string(out)
}

func (b *bodyMasker) walk(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if _, ok := b.fields[strings.ToLower(k)]; ok {
				t[k] = "***"
				continue
			}
			t[k] = b.walk(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = b.walk(t[i])
		}
		return t
	}
	return v
}
