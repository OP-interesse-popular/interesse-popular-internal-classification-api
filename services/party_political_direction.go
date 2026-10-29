package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
)

var ErrInvalidInput = errors.New("invalid party political direction")

type SavePartyPoliticalDirectionInput struct {
	PartyTSENumber     int        `json:"partyTseNumber"`
	PartyAcronym       string     `json:"partyAcronym"`
	PartyName          string     `json:"partyName"`
	Direction          string     `json:"direction"`
	Ideology           string     `json:"ideology"`
	MethodologyVersion string     `json:"methodologyVersion"`
	SourceName         string     `json:"sourceName"`
	SourceURL          string     `json:"sourceUrl"`
	EvidenceNote       string     `json:"evidenceNote"`
	ValidFrom          time.Time  `json:"validFrom"`
	ValidTo            *time.Time `json:"validTo"`
	ReviewedAt         *time.Time `json:"reviewedAt"`
}

type UpdatePartyPoliticalDirectionInput struct {
	ID string `json:"id"`
	SavePartyPoliticalDirectionInput
}

type BulkResult struct {
	Index  int                             `json:"index"`
	Status string                          `json:"status"`
	Item   *models.PartyPoliticalDirection `json:"item,omitempty"`
	Error  *BulkError                      `json:"error,omitempty"`
}

type BulkError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PartyPoliticalDirection struct {
	repository repositories.PartyPoliticalDirection
	refresher  PartyClassificationCacheRefresher
}

// PartyClassificationCacheRefresher refreshes the derived parliamentarian cache after editorial changes.
type PartyClassificationCacheRefresher interface {
	Refresh(context.Context)
}

func NewPartyPoliticalDirection(repository repositories.PartyPoliticalDirection, refreshers ...PartyClassificationCacheRefresher) *PartyPoliticalDirection {
	service := &PartyPoliticalDirection{repository: repository}
	if len(refreshers) > 0 {
		service.refresher = refreshers[0]
	}
	return service
}

func (s *PartyPoliticalDirection) Create(input SavePartyPoliticalDirectionInput) (*models.PartyPoliticalDirection, error) {
	item, err := partyPoliticalDirectionFrom(input)
	if err != nil {
		return nil, err
	}
	if err := s.repository.Create(item); err != nil {
		return nil, err
	}
	s.refreshCache()
	return item, nil
}
func (s *PartyPoliticalDirection) CreateMany(inputs []SavePartyPoliticalDirectionInput) []BulkResult {
	results := make([]BulkResult, 0, len(inputs))
	for index, input := range inputs {
		item, err := s.Create(input)
		results = append(results, bulkResult(index, item, err))
	}
	return results
}
func (s *PartyPoliticalDirection) List() ([]models.PartyPoliticalDirection, error) {
	return s.repository.List()
}
func (s *PartyPoliticalDirection) ListActiveByDirection(direction string) ([]models.PartyPoliticalDirection, error) {
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction != models.DirectionLeft && direction != models.DirectionRight {
		return nil, fmt.Errorf("%w: direction must be esquerda or direita", ErrInvalidInput)
	}
	return s.repository.ListActiveByDirection(direction)
}
func (s *PartyPoliticalDirection) ListActiveByDirectionAndIdeologies(direction string, ideologies []string) ([]models.PartyPoliticalDirection, error) {
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction != models.DirectionLeft && direction != models.DirectionRight {
		return nil, fmt.Errorf("%w: direction must be esquerda or direita", ErrInvalidInput)
	}
	if len(ideologies) == 0 {
		return s.repository.ListActiveByDirection(direction)
	}
	validIdeologies := map[string]struct{}{
		models.IdeologyRadicalLeft: {}, models.IdeologyLiberalLeft: {}, models.IdeologyLiberalRight: {}, models.IdeologyFarRight: {},
	}
	normalized := make([]string, 0, len(ideologies))
	for _, ideology := range ideologies {
		ideology = strings.ToLower(strings.TrimSpace(ideology))
		if _, ok := validIdeologies[ideology]; !ok || !validClassification(direction, ideology) {
			return nil, fmt.Errorf("%w: ideology is incompatible with direction", ErrInvalidInput)
		}
		normalized = append(normalized, ideology)
	}
	return s.repository.ListActiveByDirectionAndIdeologies(direction, normalized)
}
func (s *PartyPoliticalDirection) FindByID(id string) (*models.PartyPoliticalDirection, error) {
	return s.repository.FindByID(id)
}
func (s *PartyPoliticalDirection) Update(id string, input SavePartyPoliticalDirectionInput) (*models.PartyPoliticalDirection, error) {
	item, err := s.repository.FindByID(id)
	if err != nil {
		return nil, err
	}
	updated, err := partyPoliticalDirectionFrom(input)
	if err != nil {
		return nil, err
	}
	updated.ID, updated.CreatedAt = item.ID, item.CreatedAt
	*item = *updated
	if err := s.repository.Update(item); err != nil {
		return nil, err
	}
	s.refreshCache()
	return item, nil
}
func (s *PartyPoliticalDirection) UpdateMany(inputs []UpdatePartyPoliticalDirectionInput) []BulkResult {
	results := make([]BulkResult, 0, len(inputs))
	for index, input := range inputs {
		if strings.TrimSpace(input.ID) == "" {
			results = append(results, bulkResult(index, nil, fmt.Errorf("%w: id is required", ErrInvalidInput)))
			continue
		}
		item, err := s.Update(input.ID, input.SavePartyPoliticalDirectionInput)
		results = append(results, bulkResult(index, item, err))
	}
	return results
}
func (s *PartyPoliticalDirection) Delete(id string) error {
	if err := s.repository.Delete(id); err != nil {
		return err
	}
	s.refreshCache()
	return nil
}

func (s *PartyPoliticalDirection) refreshCache() {
	if s.refresher != nil {
		s.refresher.Refresh(context.Background())
	}
}

func bulkResult(index int, item *models.PartyPoliticalDirection, err error) BulkResult {
	if err == nil {
		return BulkResult{Index: index, Status: "success", Item: item}
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
	return BulkResult{Index: index, Status: "error", Error: &BulkError{Code: code, Message: err.Error()}}
}

func partyPoliticalDirectionFrom(input SavePartyPoliticalDirectionInput) (*models.PartyPoliticalDirection, error) {
	input.PartyAcronym = strings.ToUpper(strings.TrimSpace(input.PartyAcronym))
	input.PartyName = strings.TrimSpace(input.PartyName)
	input.Direction = strings.ToLower(strings.TrimSpace(input.Direction))
	input.Ideology = strings.ToLower(strings.TrimSpace(input.Ideology))
	input.MethodologyVersion = strings.TrimSpace(input.MethodologyVersion)
	input.SourceName = strings.TrimSpace(input.SourceName)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	input.EvidenceNote = strings.TrimSpace(input.EvidenceNote)
	if input.PartyTSENumber < 10 || input.PartyTSENumber > 99 || input.PartyAcronym == "" || input.PartyName == "" || input.MethodologyVersion == "" || input.SourceName == "" || input.ValidFrom.IsZero() {
		return nil, fmt.Errorf("%w: required field missing or partyTseNumber outside 10-99", ErrInvalidInput)
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
	if !validClassification(input.Direction, input.Ideology) {
		return nil, fmt.Errorf("%w: direction and ideology are incompatible", ErrInvalidInput)
	}
	return &models.PartyPoliticalDirection{PartyTSENumber: input.PartyTSENumber, PartyAcronym: input.PartyAcronym, PartyName: input.PartyName, Direction: input.Direction, Ideology: input.Ideology, MethodologyVersion: input.MethodologyVersion, SourceName: input.SourceName, SourceURL: input.SourceURL, EvidenceNote: input.EvidenceNote, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo, ReviewedAt: input.ReviewedAt}, nil
}

func validClassification(direction, ideology string) bool {
	return (direction == models.DirectionLeft && (ideology == models.IdeologyRadicalLeft || ideology == models.IdeologyLiberalLeft)) || (direction == models.DirectionRight && (ideology == models.IdeologyLiberalRight || ideology == models.IdeologyFarRight))
}
