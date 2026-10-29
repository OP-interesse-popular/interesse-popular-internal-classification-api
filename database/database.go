package database

import (
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(connectionString string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(connectionString), &gorm.Config{})
}

func Migrate(db *gorm.DB) error { return db.AutoMigrate(&models.PartyPoliticalDirection{}) }
