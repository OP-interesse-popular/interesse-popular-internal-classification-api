package services

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
)

type SavePoliticianEditorialMarkInput struct {
	PoliticianID       string     `json:"politicianId"`
	Direction          string     `json:"direction"`
	Ideology           string     `json:"ideology"`
	CardEnabled        bool       `json:"cardEnabled"`
	MethodologyVersion string     `json:"methodologyVersion"`
	SourceName         string     `json:"sourceName"`
	SourceURL          string     `json:"sourceUrl"`
	EvidenceNote       string     `json:"evidenceNote"`
	ValidFrom          time.Time  `json:"validFrom"`
	ValidTo            *time.Time `json:"validTo"`
	ReviewedAt         *time.Time `json:"reviewedAt"`
}

type UpdatePoliticianEditorialMarkInput struct {
	ID string `json:"id"`
	SavePoliticianEditorialMarkInput
}
type PoliticianEditorialMarkBulkResult struct {
	Index  int                             `json:"index"`
	Status string                          `json:"status"`
	Item   *models.PoliticianEditorialMark `json:"item,omitempty"`
	Error  *BulkError                      `json:"error,omitempty"`
}

type PoliticianEditorialMarks struct {
	repository  repositories.PoliticianEditorialMark
	politicians repositories.PoliticianRepository
}

func NewPoliticianEditorialMarks(repository repositories.PoliticianEditorialMark, politicians repositories.PoliticianRepository) *PoliticianEditorialMarks {
	return &PoliticianEditorialMarks{repository: repository, politicians: politicians}
}

func (s *PoliticianEditorialMarks) Create(input SavePoliticianEditorialMarkInput) (*models.PoliticianEditorialMark, error) {
	item, err := politicianEditorialMarkFrom(input)
	if err != nil {
		return nil, err
	}
	if _, err := s.politicians.FindByID(item.PoliticianID); err != nil {
		return nil, err
	}
	if err := s.repository.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *PoliticianEditorialMarks) CreateMany(inputs []SavePoliticianEditorialMarkInput) []PoliticianEditorialMarkBulkResult {
	results := make([]PoliticianEditorialMarkBulkResult, 0, len(inputs))
	for index, input := range inputs {
		item, err := s.Create(input)
		results = append(results, politicianMarkBulkResult(index, item, err))
	}
	return results
}
func (s *PoliticianEditorialMarks) List() ([]models.PoliticianEditorialMark, error) {
	return s.repository.List()
}
func (s *PoliticianEditorialMarks) Update(id string, input SavePoliticianEditorialMarkInput) (*models.PoliticianEditorialMark, error) {
	item, err := s.repository.FindByID(id)
	if err != nil {
		return nil, err
	}
	updated, err := politicianEditorialMarkFrom(input)
	if err != nil {
		return nil, err
	}
	if _, err = s.politicians.FindByID(updated.PoliticianID); err != nil {
		return nil, err
	}
	updated.ID, updated.CreatedAt = item.ID, item.CreatedAt
	*item = *updated
	if err = s.repository.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *PoliticianEditorialMarks) UpdateMany(inputs []UpdatePoliticianEditorialMarkInput) []PoliticianEditorialMarkBulkResult {
	results := make([]PoliticianEditorialMarkBulkResult, 0, len(inputs))
	for index, input := range inputs {
		if strings.TrimSpace(input.ID) == "" {
			results = append(results, politicianMarkBulkResult(index, nil, fmt.Errorf("%w: id is required", ErrInvalidInput)))
			continue
		}
		item, err := s.Update(input.ID, input.SavePoliticianEditorialMarkInput)
		results = append(results, politicianMarkBulkResult(index, item, err))
	}
	return results
}
func (s *PoliticianEditorialMarks) Delete(id string) error { return s.repository.Delete(id) }

func politicianEditorialMarkFrom(input SavePoliticianEditorialMarkInput) (*models.PoliticianEditorialMark, error) {
	input.PoliticianID, input.Direction, input.Ideology = strings.TrimSpace(input.PoliticianID), strings.ToLower(strings.TrimSpace(input.Direction)), strings.ToLower(strings.TrimSpace(input.Ideology))
	input.MethodologyVersion, input.SourceName, input.SourceURL, input.EvidenceNote = strings.TrimSpace(input.MethodologyVersion), strings.TrimSpace(input.SourceName), strings.TrimSpace(input.SourceURL), strings.TrimSpace(input.EvidenceNote)
	if input.PoliticianID == "" || input.MethodologyVersion == "" || input.SourceName == "" || input.EvidenceNote == "" || input.ValidFrom.IsZero() || !validClassification(input.Direction, input.Ideology) {
		return nil, fmt.Errorf("%w: required field missing or invalid classification", ErrInvalidInput)
	}
	if input.SourceURL != "" {
		parsed, err := url.ParseRequestURI(input.SourceURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("%w: sourceUrl must be an absolute URL", ErrInvalidInput)
		}
	}
	if input.ValidTo != nil && input.ValidTo.Before(input.ValidFrom) {
		return nil, fmt.Errorf("%w: validTo must be on or after validFrom", ErrInvalidInput)
	}
	return &models.PoliticianEditorialMark{PoliticianID: input.PoliticianID, Direction: input.Direction, Ideology: input.Ideology, CardEnabled: input.CardEnabled, MethodologyVersion: input.MethodologyVersion, SourceName: input.SourceName, SourceURL: input.SourceURL, EvidenceNote: input.EvidenceNote, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo, ReviewedAt: input.ReviewedAt}, nil
}
func politicianMarkBulkResult(index int, item *models.PoliticianEditorialMark, err error) PoliticianEditorialMarkBulkResult {
	if err == nil {
		return PoliticianEditorialMarkBulkResult{Index: index, Status: "success", Item: item}
	}
	code := "internal_error"
	if errors.Is(err, ErrInvalidInput) {
		code = "validation_error"
	}
	if errors.Is(err, repositories.ErrConflict) {
		code = "conflict"
	}
	if errors.Is(err, repositories.ErrNotFound) {
		code = "not_found"
	}
	return PoliticianEditorialMarkBulkResult{Index: index, Status: "error", Error: &BulkError{Code: code, Message: err.Error()}}
}
