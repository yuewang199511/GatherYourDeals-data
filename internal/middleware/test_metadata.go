package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// testHeader maps a request header name to its OTel span attribute key.
var testHeaders = []struct {
	header string
	attr   string
}{
	{"X-Test-Run-Id", "test.run_id"},
	{"X-Test-Group", "test.group"},
	{"X-Test-Phase", "test.phase"},
	{"X-Test-Target-Rps", "test.target_rps"},
}

// TestMetadata reads X-Test-* request headers and sets the corresponding
// span attributes (test.run_id, test.group, test.phase, test.target_rps).
//
// A header that is absent from the request results in no attribute being set
// on the span (FR-007). This middleware must run after otelgin so that the
// span already exists in the request context (see contracts/span-attributes.md).
func TestMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			for _, h := range testHeaders {
				if v := c.GetHeader(h.header); v != "" {
					span.SetAttributes(attribute.String(h.attr, v))
				}
			}
		}
		c.Next()
	}
}
