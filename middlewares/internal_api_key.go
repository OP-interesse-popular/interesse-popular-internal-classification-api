package middlewares

import (
	"crypto/subtle"
	"net/http"

	commonserrors "github.com/TiB-Software/go-commons/errors"
	commonsutils "github.com/TiB-Software/go-commons/utils"
	"github.com/gin-gonic/gin"
)

const InternalAPIKeyHeader = "X-Internal-API-Key"

func RequireInternalAPIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := c.GetHeader(InternalAPIKeyHeader)
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusUnauthorized, commonserrors.ApiErrorItem{UserMessage: "unauthorized", SystemMessage: "internal API key invalid"}))
			return
		}
		c.Next()
	}
}

// RequireInternalReadAPIKey accepts either the dedicated read key or the
// administrative key. The administrative key retains access for internal tools.
func RequireInternalReadAPIKey(readKey, adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := c.GetHeader(InternalAPIKeyHeader)
		validReadKey := subtle.ConstantTimeCompare([]byte(provided), []byte(readKey)) == 1
		validAdminKey := subtle.ConstantTimeCompare([]byte(provided), []byte(adminKey)) == 1
		if provided == "" || (!validReadKey && !validAdminKey) {
			commonsutils.SendErrorResponse(c, commonserrors.NewApiError(http.StatusUnauthorized, commonserrors.ApiErrorItem{UserMessage: "unauthorized", SystemMessage: "internal API key invalid"}))
			return
		}
		c.Next()
	}
}
