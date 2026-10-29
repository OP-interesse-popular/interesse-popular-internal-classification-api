package repositories

import (
	"errors"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"gorm.io/gorm"
)

type PoliticianEditorialMark interface {
	Create(*models.PoliticianEditorialMark) error
	List() ([]models.PoliticianEditorialMark, error)
	FindByID(string) (*models.PoliticianEditorialMark, error)
	Update(*models.PoliticianEditorialMark) error
	Delete(string) error
}

type politicianEditorialMark struct{ db *gorm.DB }

func NewPoliticianEditorialMark(db *gorm.DB) PoliticianEditorialMark {
	return &politicianEditorialMark{db: db}
}
func (r *politicianEditorialMark) Create(item *models.PoliticianEditorialMark) error {
	return translateError(r.db.Create(item).Error)
}
func (r *politicianEditorialMark) List() ([]models.PoliticianEditorialMark, error) {
	var items []models.PoliticianEditorialMark
	return items, r.db.Order("created_at ASC").Find(&items).Error
}
func (r *politicianEditorialMark) FindByID(id string) (*models.PoliticianEditorialMark, error) {
	var item models.PoliticianEditorialMark
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}
func (r *politicianEditorialMark) Update(item *models.PoliticianEditorialMark) error {
	return translateError(r.db.Save(item).Error)
}
func (r *politicianEditorialMark) Delete(id string) error {
	result := r.db.Delete(&models.PoliticianEditorialMark{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
