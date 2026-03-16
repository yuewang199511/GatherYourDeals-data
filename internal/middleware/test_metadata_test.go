package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatheryourdeals/data/internal/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestTestMetadata_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.TestMetadata())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "no test headers",
			headers: map[string]string{},
		},
		{
			name: "all four test headers",
			headers: map[string]string{
				"X-Test-Run-Id":     "run-001",
				"X-Test-Group":      "cpu_bound",
				"X-Test-Phase":      "moderate",
				"X-Test-Target-Rps": "100",
			},
		},
		{
			name: "partial headers",
			headers: map[string]string{
				"X-Test-Run-Id": "run-002",
				"X-Test-Group":  "read_heavy",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
		})
	}
}
