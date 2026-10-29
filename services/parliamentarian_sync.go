package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/clients"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
)

// ParliamentarianStore persists the read model consumed by the public API.
type ParliamentarianStore interface {
	Set(context.Context, []models.Parliamentarian) error
	IsEmpty(context.Context) (bool, error)
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
	cache   ParliamentarianStore
	camara  clients.CamaraClient
	senado  clients.SenadoClient
	parties PartyClassificationReader
	running atomic.Bool
	pending atomic.Bool
}

func NewParliamentarianSync(cache ParliamentarianStore, camara clients.CamaraClient, senado clients.SenadoClient, parties PartyClassificationReader) *ParliamentarianSync {
	return &ParliamentarianSync{cache: cache, camara: camara, senado: senado, parties: parties}
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
	classifications, err := s.parties.ListActive()
	if err != nil {
		slog.Error("parliamentarians_sync_failed", "step", "party_classifications", "error", err)
		return
	}
	items = applyPartyClassifications(items, classifications)
	slog.Info("parliamentarians_sync_step_completed", "step", "party_classifications", "count", len(classifications))
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
	empty, err := s.cache.IsEmpty(ctx)
	if err != nil {
		return fmt.Errorf("check parliamentarians cache: %w", err)
	}
	if empty {
		slog.Info("parliamentarians_cache_empty", "action", "sync")
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
