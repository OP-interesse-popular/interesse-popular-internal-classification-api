package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/clients"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
)

// ParliamentarianStore persists the read model consumed by the public API.
type ParliamentarianStore interface {
	Set(context.Context, []models.Parliamentarian) error
	SetAll(context.Context, []models.Politician) error
	IsEmpty(context.Context) (bool, error)
	IsAllEmpty(context.Context) (bool, error)
}

type ParliamentarianSynchronizer interface {
	Sync(context.Context)
}

// PartyClassificationReader supplies the active classifications embedded in the cache.
type PartyClassificationReader interface {
	ListActive() ([]models.PartyPoliticalDirection, error)
}

// ParliamentarianSync coordinates source clients and the cache. HTTP and scheduling stay outside this service.
type ParliamentarianSync struct {
	cache       ParliamentarianStore
	camara      clients.CamaraClient
	senado      clients.SenadoClient
	parties     PartyClassificationReader
	politicians repositories.PoliticianRepository
	tse         clients.TSEClient
	running     atomic.Bool
	pending     atomic.Bool
}

func NewParliamentarianSync(cache ParliamentarianStore, camara clients.CamaraClient, senado clients.SenadoClient, parties PartyClassificationReader, politicians repositories.PoliticianRepository, tse clients.TSEClient) *ParliamentarianSync {
	return &ParliamentarianSync{cache: cache, camara: camara, senado: senado, parties: parties, politicians: politicians, tse: tse}
}

func (s *ParliamentarianSync) Sync(ctx context.Context) {
	if !s.running.CompareAndSwap(false, true) {
		s.pending.Store(true)
		slog.Info("parliamentarians_sync_queued", "reason", "already_running")
		return
	}
	defer func() {
		s.running.Store(false)
		if s.pending.Swap(false) {
			slog.Info("parliamentarians_sync_started", "reason", "queued_refresh")
			go s.Sync(ctx)
		}
	}()

	slog.Info("parliamentarians_sync_started", "step", "camara")
	camara, err := s.camara.ListActiveDeputies(ctx)
	if err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "camara", "error", err)
		return
	}
	slog.Info("parliamentarians_sync_step_completed", "step", "camara", "count", len(camara))

	slog.Info("parliamentarians_sync_started", "step", "senado")
	senado, err := s.senado.ListActiveSenators(ctx)
	if err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "senado", "error", err)
		return
	}
	slog.Info("parliamentarians_sync_step_completed", "step", "senado", "count", len(senado))

	items := append(camara, senado...)
	if err := s.politicians.SyncActive(ctx, items); err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "politicians", "error", err)
		return
	}
	slog.Info("parliamentarians_sync_step_completed", "step", "politicians", "count", len(items))
	if s.tse != nil {
		slog.Info("parliamentarians_sync_started", "step", "tse_elected_candidates")
		candidates, err := s.tse.ListElectedFederalParliamentarians(ctx)
		if err != nil {
			slog.Error("parliamentarians_sync_failed", "step", "tse_elected_candidates", "error", err)
			return
		}
		elected := make([]repositories.ElectedCandidate, 0, len(candidates))
		for _, candidate := range candidates {
			elected = append(elected, repositories.ElectedCandidate{TSECandidateID: candidate.CandidateID, FullName: candidate.FullName, BallotName: candidate.BallotName, PartyAcronym: candidate.Party, State: candidate.State, LegislativeHouse: candidate.House, Office: candidate.Office, ElectionYear: candidate.ElectionYear})
		}
		if err := s.politicians.SyncElected(ctx, elected); err != nil {
			slog.Error("parliamentarians_sync_failed", "step", "tse_merge", "error", err)
			return
		}
		slog.Info("parliamentarians_sync_step_completed", "step", "tse_merge", "count", len(elected))
	} else {
		slog.Info("parliamentarians_sync_skipped", "step", "tse_merge", "reason", "TSE_ELECTED_CANDIDATES_URL_not_configured")
	}
	classifications, err := s.parties.ListActive()
	if err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "party_classifications", "error", err)
		return
	}
	items = applyPartyClassifications(items, classifications)
	slog.Info("parliamentarians_sync_step_completed", "step", "party_classifications", "count", len(classifications))
	allPoliticians, err := s.politicians.List()
	if err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "all_politicians_read", "error", err)
		return
	}
	allPoliticians = applyMandatePartyClassifications(allPoliticians, classifications)
	slog.Info("parliamentarians_sync_started", "step", "redis_all_politicians_write", "count", len(allPoliticians))
	if err := s.cache.SetAll(ctx, allPoliticians); err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "redis_all_politicians_write", "error", err)
		return
	}
	slog.Info("parliamentarians_sync_step_completed", "step", "redis_all_politicians_write", "count", len(allPoliticians))
	slog.Info("parliamentarians_sync_started", "step", "redis_write", "count", len(items))
	if err := s.cache.Set(ctx, items); err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "redis_write", "error", err)
		return
	}
	slog.Info("parliamentarians_sync_completed", "total", len(items))
}

// Refresh schedules a cache rebuild after a classification change without blocking the admin request.
func (s *ParliamentarianSync) Refresh(ctx context.Context) {
	slog.Info("parliamentarians_cache_refresh_requested", "reason", "party_classification_changed")
	go s.Sync(ctx)
}

// EnsureCache is used by the one-minute scheduler check.
func (s *ParliamentarianSync) EnsureCache(ctx context.Context) error {
	activeEmpty, err := s.cache.IsEmpty(ctx)
	if err != nil {
		return fmt.Errorf("check active parliamentarians cache: %w", err)
	}
	allEmpty, err := s.cache.IsAllEmpty(ctx)
	if err != nil {
		return fmt.Errorf("check all politicians cache: %w", err)
	}
	if activeEmpty || allEmpty {
		slog.Info("parliamentarians_cache_empty", "active_empty", activeEmpty, "all_politicians_empty", allEmpty, "action", "sync")
		s.Sync(ctx)
	}
	return nil
}

func applyPartyClassifications(items []models.Parliamentarian, classifications []models.PartyPoliticalDirection) []models.Parliamentarian {
	byAcronym := make(map[string]models.PartyPoliticalDirection, len(classifications))
	for _, classification := range classifications {
		byAcronym[strings.ToUpper(strings.TrimSpace(classification.PartyAcronym))] = classification
	}
	for index := range items {
		classification, ok := byAcronym[strings.ToUpper(strings.TrimSpace(items[index].Party))]
		if !ok {
			continue
		}
		items[index].PartyDirection = classification.Direction
		items[index].PartyIdeology = classification.Ideology
	}
	return items
}

func applyMandatePartyClassifications(items []models.Politician, classifications []models.PartyPoliticalDirection) []models.Politician {
	byAcronym := make(map[string]models.PartyPoliticalDirection, len(classifications))
	for _, classification := range classifications {
		byAcronym[strings.ToUpper(strings.TrimSpace(classification.PartyAcronym))] = classification
	}
	for politicianIndex := range items {
		for mandateIndex := range items[politicianIndex].Mandates {
			mandate := &items[politicianIndex].Mandates[mandateIndex]
			classification, ok := byAcronym[strings.ToUpper(strings.TrimSpace(mandate.PartyAcronym))]
			if !ok {
				continue
			}
			mandate.PartyDirection = classification.Direction
			mandate.PartyIdeology = classification.Ideology
		}
	}
	return items
}
