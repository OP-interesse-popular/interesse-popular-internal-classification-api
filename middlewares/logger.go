package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	commonserrors "github.com/TiB-Software/go-commons/errors"
	commonsutils "github.com/TiB-Software/go-commons/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

func ConfigureLogger() { slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil))) }

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}

func SanitizedLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		requestID, _ := c.Get("request_id")
		slog.InfoContext(c.Request.Context(), "http_request",
			"request_id", requestID, "method", c.Request.Method, "route", route(c),
			"status", c.Writer.Status(), "duration_ms", time.Since(startedAt).Milliseconds(), "source_ip", c.ClientIP(),
		)
	}
}

func SanitizedRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get("request_id")
		slog.ErrorContext(c.Request.Context(), "panic_recovered", "request_id", requestID, "method", c.Request.Method, "route", route(c), "panic_type", fmt.Sprintf("%T", recovered))
		commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusInternalServerError, commonserrors.InternalServerError("panic recovered")))
	})
}

func route(c *gin.Context) string {
	if value := c.FullPath(); value != "" {
		return value
	}
	return "<unmatched>"
}
