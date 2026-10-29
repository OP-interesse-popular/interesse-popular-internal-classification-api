package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireInternalAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/internal", RequireInternalAPIKey("expected-key"), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, test := range []struct {
		name   string
		key    string
		status int
	}{
		{name: "missing key", status: http.StatusUnauthorized},
		{name: "invalid key", key: "other-key", status: http.StatusUnauthorized},
		{name: "valid key", key: "expected-key", status: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/internal", nil)
			if test.key != "" {
				request.Header.Set(InternalAPIKeyHeader, test.key)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestRequireInternalReadAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/internal", RequireInternalReadAPIKey("read-key", "admin-key"), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, key := range []string{"read-key", "admin-key"} {
		request := httptest.NewRequest(http.MethodGet, "/internal", nil)
		request.Header.Set(InternalAPIKeyHeader, key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("key %q returned %d, want %d", key, response.Code, http.StatusNoContent)
		}
	}
}
