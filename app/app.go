package app

import (
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/config"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/handlers"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type App struct {
	Router                   *gin.Engine
	Config                   config.Config
	PartyPoliticalDirections *handlers.PartyPoliticalDirection
}

func New(router *gin.Engine, db *gorm.DB, cfg config.Config, refresher services.PartyClassificationCacheRefresher) *App {
	repository := repositories.NewPartyPoliticalDirection(db)
	service := services.NewPartyPoliticalDirection(repository, refresher)
	return &App{Router: router, Config: cfg, PartyPoliticalDirections: handlers.NewPartyPoliticalDirection(service)}
}
