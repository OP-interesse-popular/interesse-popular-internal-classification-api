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
	Politicians              *handlers.Politicians
	PoliticianEditorialMarks *handlers.PoliticianEditorialMarks
}

func New(router *gin.Engine, db *gorm.DB, cfg config.Config, refresher services.PartyClassificationCacheRefresher) *App {
	partyRepository := repositories.NewPartyPoliticalDirection(db)
	politicianRepository := repositories.NewPolitician(db)
	partyService := services.NewPartyPoliticalDirection(partyRepository, refresher)
	markRepository := repositories.NewPoliticianEditorialMark(db)
	markService := services.NewPoliticianEditorialMarks(markRepository, politicianRepository)
	return &App{Router: router, Config: cfg, PartyPoliticalDirections: handlers.NewPartyPoliticalDirection(partyService), Politicians: handlers.NewPoliticians(politicianRepository), PoliticianEditorialMarks: handlers.NewPoliticianEditorialMarks(markService)}
}
