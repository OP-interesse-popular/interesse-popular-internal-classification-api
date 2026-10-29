package repositories

import (
	"errors"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrConflict = errors.New("record conflicts with an existing party")
	ErrNotFound = errors.New("record not found")
)

type PartyPoliticalDirection interface {
	Create(*models.PartyPoliticalDirection) error
	List() ([]models.PartyPoliticalDirection, error)
	ListActive() ([]models.PartyPoliticalDirection, error)
	ListActiveByDirection(string) ([]models.PartyPoliticalDirection, error)
	ListActiveByDirectionAndIdeologies(string, []string) ([]models.PartyPoliticalDirection, error)
	FindByID(string) (*models.PartyPoliticalDirection, error)
	Update(*models.PartyPoliticalDirection) error
	Delete(string) error
}

type partyPoliticalDirection struct{ db *gorm.DB }

func NewPartyPoliticalDirection(db *gorm.DB) PartyPoliticalDirection {
	return &partyPoliticalDirection{db: db}
}
func (r *partyPoliticalDirection) Create(item *models.PartyPoliticalDirection) error {
	return translateError(r.db.Create(item).Error)
}
func (r *partyPoliticalDirection) List() ([]models.PartyPoliticalDirection, error) {
	var items []models.PartyPoliticalDirection
	return items, r.db.Order("party_acronym ASC").Find(&items).Error
}
func (r *partyPoliticalDirection) ListActive() ([]models.PartyPoliticalDirection, error) {
	var items []models.PartyPoliticalDirection
	return items, r.db.Where("valid_from <= CURRENT_DATE AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)").Order("party_acronym ASC").Find(&items).Error
}
func (r *partyPoliticalDirection) ListActiveByDirection(direction string) ([]models.PartyPoliticalDirection, error) {
	var items []models.PartyPoliticalDirection
	return items, r.db.Where("direction = ? AND valid_from <= CURRENT_DATE AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)", direction).Order("party_acronym ASC").Find(&items).Error
}
func (r *partyPoliticalDirection) ListActiveByDirectionAndIdeologies(direction string, ideologies []string) ([]models.PartyPoliticalDirection, error) {
	var items []models.PartyPoliticalDirection
	return items, r.db.Where("direction = ? AND ideology IN ? AND valid_from <= CURRENT_DATE AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)", direction, ideologies).Order("party_acronym ASC").Find(&items).Error
}
func (r *partyPoliticalDirection) FindByID(id string) (*models.PartyPoliticalDirection, error) {
	var item models.PartyPoliticalDirection
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}
func (r *partyPoliticalDirection) Update(item *models.PartyPoliticalDirection) error {
	return translateError(r.db.Save(item).Error)
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (r *partyPoliticalDirection) Delete(id string) error {
	result := r.db.Delete(&models.PartyPoliticalDirection{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
