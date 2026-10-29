package app

import (
	"net/http"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/middlewares"
	"github.com/TiB-Software/go-commons/commonsmodels"
	commonsutils "github.com/TiB-Software/go-commons/utils"
	"github.com/gin-gonic/gin"
)

func (a *App) RegisterRoutes() {
	a.Router.Use(
		commonsutils.CORS(commonsutils.CORSConfig{AllowedOrigins: a.Config.CORSAllowedOrigins, AllowCredentials: true}),
		commonsutils.SecurityHeaders(commonsutils.SecurityHeadersConfig{Production: a.Config.IsProduction()}),
		commonsutils.RequireHTTPS(commonsutils.HTTPSConfig{Production: a.Config.IsProduction(), TrustedProxies: a.Config.TrustedProxies}),
	)
	a.Router.GET("/api/v1/healthcheck", func(c *gin.Context) { commonsutils.SendResponse(c, commonsmodels.NewResponseSuccess(), http.StatusOK) })

	directions := a.Router.Group("/api/internal/v1/party-political-directions")
	directions.GET("", middlewares.RequireInternalReadAPIKey(a.Config.InternalReadAPIKey, a.Config.InternalAPIKey), a.PartyPoliticalDirections.List())
	directions.GET("/:id", middlewares.RequireInternalReadAPIKey(a.Config.InternalReadAPIKey, a.Config.InternalAPIKey), a.PartyPoliticalDirections.Get())
	directions.POST("", middlewares.RequireInternalAPIKey(a.Config.InternalAPIKey), a.PartyPoliticalDirections.CreateMany())
	directions.PUT("", middlewares.RequireInternalAPIKey(a.Config.InternalAPIKey), a.PartyPoliticalDirections.UpdateMany())
	directions.DELETE("/:id", middlewares.RequireInternalAPIKey(a.Config.InternalAPIKey), a.PartyPoliticalDirections.Delete())
}
