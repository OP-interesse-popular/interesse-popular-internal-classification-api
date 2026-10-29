package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
)

type ElectedCandidate struct {
	TSECandidateID   string
	FullName         string
	BallotName       string
	PartyAcronym     string
	State            string
	LegislativeHouse string
	Office           string
	ElectionYear     int
}

type PoliticianRepository interface {
	SyncActive(context.Context, []models.Parliamentarian) error
	SyncElected(context.Context, []ElectedCandidate) error
	List() ([]models.Politician, error)
	FindByID(string) (*models.Politician, error)
}

type politicianRepository struct{ db *gorm.DB }

func NewPolitician(db *gorm.DB) PoliticianRepository { return &politicianRepository{db: db} }

func (r *politicianRepository) List() ([]models.Politician, error) {
	var items []models.Politician
	return items, r.db.Preload("ExternalIDs").Preload("Mandates", func(db *gorm.DB) *gorm.DB {
		return db.Order("starts_at DESC NULLS LAST, created_at DESC")
	}).Order("full_name ASC").Find(&items).Error
}

func (r *politicianRepository) FindByID(id string) (*models.Politician, error) {
	var item models.Politician
	if err := r.db.Preload("ExternalIDs").Preload("Mandates", func(db *gorm.DB) *gorm.DB {
		return db.Order("starts_at DESC NULLS LAST, created_at DESC")
	}).First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *politicianRepository) SyncActive(ctx context.Context, items []models.Parliamentarian) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			house := strings.ToLower(strings.TrimSpace(item.LegislativeHouse))
			if item.ID == "" || house == "" {
				continue
			}
			politician, err := resolveByExternalID(tx, house, models.ExternalIdentifierParliamentarian, item.ID)
			if err != nil {
				return err
			}
			if politician == nil {
				politician, err = findUniquePendingMandateMatch(tx, item)
				if err != nil {
					return err
				}
			}
			if politician == nil {
				politician, err = createPolitician(tx, item.Name)
				if err != nil {
					return err
				}
			}
			if err = ensureExternalID(tx, politician.ID, house, models.ExternalIdentifierParliamentarian, item.ID, item.SourceURL); err != nil {
				return err
			}
			if err = ensureAlias(tx, politician.ID, item.Name, house); err != nil {
				return err
			}
			now := time.Now().UTC()
			if err := upsertCurrentMandate(tx, politician.ID, item, &now); err != nil {
				return err
			}
		}
		if err := closeMissingActiveMandates(tx, items); err != nil {
			return err
		}
		return nil
	})
}

func (r *politicianRepository) SyncElected(ctx context.Context, candidates []ElectedCandidate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, candidate := range candidates {
			if candidate.TSECandidateID == "" || candidate.ElectionYear == 0 {
				continue
			}
			politician, err := resolveByExternalID(tx, models.ExternalSourceTSE, models.ExternalIdentifierCandidate, candidate.TSECandidateID)
			if err != nil {
				return err
			}
			if politician == nil {
				matches, err := findExactOfficeMatches(tx, candidate)
				if err != nil {
					return err
				}
				if len(matches) == 1 {
					politician = &matches[0]
					if err := tx.Create(&models.PoliticianIdentityMatch{PoliticianID: politician.ID, Source: models.ExternalSourceTSE, ExternalID: candidate.TSECandidateID, MatchMethod: "exact_name_state_office", Status: models.MatchStatusAutomatic}).Error; err != nil {
						return err
					}
				} else {
					politician, err = createPolitician(tx, firstNonEmpty(candidate.FullName, candidate.BallotName))
					if err != nil {
						return err
					}
				}
				if err := ensureExternalID(tx, politician.ID, models.ExternalSourceTSE, models.ExternalIdentifierCandidate, candidate.TSECandidateID, ""); err != nil {
					return err
				}
			}
			if err := ensureAlias(tx, politician.ID, candidate.FullName, models.ExternalSourceTSE); err != nil {
				return err
			}
			if err := ensureAlias(tx, politician.ID, candidate.BallotName, models.ExternalSourceTSE); err != nil {
				return err
			}
			if err := upsertElectedMandate(tx, politician.ID, candidate); err != nil {
				return err
			}
		}
		return nil
	})
}

func resolveByExternalID(tx *gorm.DB, source, identifierType, externalID string) (*models.Politician, error) {
	var identifier models.PoliticianExternalID
	err := tx.Where("source = ? AND identifier_type = ? AND external_id = ?", source, identifierType, externalID).First(&identifier).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var politician models.Politician
	if err = tx.First(&politician, "id = ?", identifier.PoliticianID).Error; err != nil {
		return nil, err
	}
	return &politician, nil
}

func createPolitician(tx *gorm.DB, name string) (*models.Politician, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("politician name is required")
	}
	item := &models.Politician{FullName: name, NormalizedName: normalizeName(name)}
	if err := tx.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func findExactOfficeMatches(tx *gorm.DB, candidate ElectedCandidate) ([]models.Politician, error) {
	var items []models.Politician
	query := tx.Model(&models.Politician{}).
		Joins("JOIN parliamentary_mandates ON parliamentary_mandates.politician_id = politicians.id").
		Where("politicians.normalized_name IN ? AND parliamentary_mandates.state = ? AND parliamentary_mandates.office = ?", []string{normalizeName(candidate.FullName), normalizeName(candidate.BallotName)}, candidate.State, candidate.Office).
		Distinct()
	return items, query.Find(&items).Error
}

func upsertCurrentMandate(tx *gorm.DB, politicianID string, item models.Parliamentarian, seenAt *time.Time) error {
	var pending models.ParliamentaryMandate
	pendingErr := tx.Where("politician_id = ? AND legislative_house = ? AND office = ? AND state = ? AND status = ? AND starts_at <= ?", politicianID, item.LegislativeHouse, officeForHouse(item.LegislativeHouse), item.State, models.PoliticianStatusElectedPending, *seenAt).First(&pending).Error
	if pendingErr == nil {
		if err := tx.Model(&models.ParliamentaryMandate{}).Where("politician_id = ? AND legislative_house = ? AND office = ? AND state = ? AND status = ? AND id <> ?", politicianID, item.LegislativeHouse, officeForHouse(item.LegislativeHouse), item.State, models.PoliticianStatusActive, pending.ID).Updates(map[string]any{"status": models.PoliticianStatusFormer, "ends_at": pending.StartsAt.Add(-time.Second)}).Error; err != nil {
			return err
		}
		return tx.Model(&pending).Updates(map[string]any{"party_acronym": item.Party, "status": models.PoliticianStatusActive, "source_last_seen_at": seenAt}).Error
	}
	if pendingErr != nil && pendingErr != gorm.ErrRecordNotFound {
		return pendingErr
	}
	var mandate models.ParliamentaryMandate
	err := tx.Where("politician_id = ? AND legislative_house = ? AND office = ? AND state = ? AND election_year IS NULL", politicianID, item.LegislativeHouse, officeForHouse(item.LegislativeHouse), item.State).First(&mandate).Error
	if err == gorm.ErrRecordNotFound {
		startsAt, endsAt := inferredActiveTerm(item.LegislativeHouse, *seenAt)
		return tx.Create(&models.ParliamentaryMandate{PoliticianID: politicianID, LegislativeHouse: item.LegislativeHouse, Office: officeForHouse(item.LegislativeHouse), State: item.State, PartyAcronym: item.Party, Status: models.PoliticianStatusActive, StartsAt: startsAt, EndsAt: endsAt, SourceLastSeenAt: seenAt}).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{"party_acronym": item.Party, "status": models.PoliticianStatusActive, "source_last_seen_at": seenAt}
	if mandate.StartsAt == nil || mandate.EndsAt == nil {
		startsAt, endsAt := inferredActiveTerm(item.LegislativeHouse, *seenAt)
		if mandate.StartsAt == nil && startsAt != nil {
			updates["starts_at"] = startsAt
		}
		if mandate.EndsAt == nil && endsAt != nil {
			updates["ends_at"] = endsAt
		}
	}
	return tx.Model(&mandate).Updates(updates).Error
}

func ensureExternalID(tx *gorm.DB, politicianID, source, identifierType, externalID, sourceURL string) error {
	var existing models.PoliticianExternalID
	err := tx.Where("source = ? AND identifier_type = ? AND external_id = ?", source, identifierType, externalID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&models.PoliticianExternalID{PoliticianID: politicianID, Source: source, IdentifierType: identifierType, ExternalID: externalID, SourceURL: sourceURL}).Error
	}
	return err
}

func ensureAlias(tx *gorm.DB, politicianID, value, source string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var alias models.PoliticianAlias
	err := tx.Where("politician_id = ? AND normalized_value = ? AND source = ?", politicianID, normalizeName(value), source).First(&alias).Error
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&models.PoliticianAlias{PoliticianID: politicianID, Value: value, NormalizedValue: normalizeName(value), Source: source}).Error
	}
	return err
}

func findUniquePendingMandateMatch(tx *gorm.DB, item models.Parliamentarian) (*models.Politician, error) {
	var items []models.Politician
	err := tx.Model(&models.Politician{}).
		Joins("JOIN parliamentary_mandates ON parliamentary_mandates.politician_id = politicians.id").
		Joins("LEFT JOIN politician_aliases ON politician_aliases.politician_id = politicians.id").
		Where("parliamentary_mandates.legislative_house = ? AND parliamentary_mandates.office = ? AND parliamentary_mandates.state = ? AND parliamentary_mandates.status = ?", item.LegislativeHouse, officeForHouse(item.LegislativeHouse), item.State, models.PoliticianStatusElectedPending).
		Where("politicians.normalized_name = ? OR politician_aliases.normalized_value = ?", normalizeName(item.Name), normalizeName(item.Name)).
		Distinct().Find(&items).Error
	if err != nil || len(items) != 1 {
		return nil, err
	}
	return &items[0], nil
}

func closeMissingActiveMandates(tx *gorm.DB, items []models.Parliamentarian) error {
	known := make(map[string]map[string]struct{}, 2)
	for _, item := range items {
		house := strings.ToLower(strings.TrimSpace(item.LegislativeHouse))
		if known[house] == nil {
			known[house] = make(map[string]struct{})
		}
		known[house][item.ID] = struct{}{}
	}
	for house, ids := range known {
		var externalIDs []models.PoliticianExternalID
		if err := tx.Where("source = ? AND identifier_type = ?", house, models.ExternalIdentifierParliamentarian).Find(&externalIDs).Error; err != nil {
			return err
		}
		for _, externalID := range externalIDs {
			if _, present := ids[externalID.ExternalID]; present {
				continue
			}
			var mandates []models.ParliamentaryMandate
			if err := tx.Where("politician_id = ? AND legislative_house = ? AND status = ?", externalID.PoliticianID, house, models.PoliticianStatusActive).Find(&mandates).Error; err != nil {
				return err
			}
			now := time.Now().UTC()
			for _, mandate := range mandates {
				endsAt := now
				if mandate.EndsAt != nil && mandate.EndsAt.Before(endsAt) {
					endsAt = *mandate.EndsAt
				}
				if err := tx.Model(&mandate).Updates(map[string]any{"status": models.PoliticianStatusFormer, "ends_at": endsAt}).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func inferredActiveTerm(house string, now time.Time) (*time.Time, *time.Time) {
	if house != "camara" {
		return nil, nil
	}
	nextElectionYear := now.Year()
	for nextElectionYear%4 != 2 {
		nextElectionYear++
	}
	start := time.Date(nextElectionYear-3, time.February, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(nextElectionYear+1, time.January, 31, 23, 59, 59, 0, time.UTC)
	return &start, &end
}

func upsertElectedMandate(tx *gorm.DB, politicianID string, candidate ElectedCandidate) error {
	var mandate models.ParliamentaryMandate
	err := tx.Where("politician_id = ? AND legislative_house = ? AND office = ? AND state = ? AND election_year = ?", politicianID, candidate.LegislativeHouse, candidate.Office, candidate.State, candidate.ElectionYear).First(&mandate).Error
	start := time.Date(candidate.ElectionYear+1, time.February, 1, 0, 0, 0, 0, time.UTC)
	years := 4
	if candidate.LegislativeHouse == "senado" {
		years = 8
	}
	end := time.Date(candidate.ElectionYear+1+years, time.January, 31, 23, 59, 59, 0, time.UTC)
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&models.ParliamentaryMandate{PoliticianID: politicianID, LegislativeHouse: candidate.LegislativeHouse, Office: candidate.Office, State: candidate.State, PartyAcronym: candidate.PartyAcronym, Status: models.PoliticianStatusElectedPending, ElectionYear: &candidate.ElectionYear, StartsAt: &start, EndsAt: &end}).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&mandate).Updates(map[string]any{"party_acronym": candidate.PartyAcronym, "status": models.PoliticianStatusElectedPending, "starts_at": start, "ends_at": end}).Error
}

func officeForHouse(house string) string {
	if strings.EqualFold(house, "senado") {
		return "senador"
	}
	return "deputado_federal"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeName(value string) string {
	value = norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value)
}
