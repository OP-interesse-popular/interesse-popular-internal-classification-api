package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/services"
)

// ParliamentarianRefresh owns the execution policy; the synchronizer owns the work.
type ParliamentarianRefresh struct {
	synchronizer services.ParliamentarianSynchronizer
	cacheCheck   time.Duration
	refresh      time.Duration
}

func NewParliamentarianRefresh(synchronizer services.ParliamentarianSynchronizer, cacheCheck, refresh time.Duration) *ParliamentarianRefresh {
	return &ParliamentarianRefresh{synchronizer: synchronizer, cacheCheck: cacheCheck, refresh: refresh}
}

func (r *ParliamentarianRefresh) Start(ctx context.Context) {
	go r.synchronizer.Sync(ctx)
	go func() {
		cacheCheckTicker := time.NewTicker(r.cacheCheck)
		refreshTicker := time.NewTicker(r.refresh)
		defer cacheCheckTicker.Stop()
		defer refreshTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-cacheCheckTicker.C:
				slog.Info("parliamentarians_cache_check_started")
				if err := r.ensureCache(ctx); err != nil {
					slog.Error("parliamentarians_cache_check_failed", "error", err)
				} else {
					slog.Info("parliamentarians_cache_check_completed")
				}
			case <-refreshTicker.C:
				slog.Info("parliamentarians_refresh_started")
				go r.synchronizer.Sync(ctx)
			}
		}
	}()
}

func (r *ParliamentarianRefresh) ensureCache(ctx context.Context) error {
	if synchronizer, ok := r.synchronizer.(interface{ EnsureCache(context.Context) error }); ok {
		return synchronizer.EnsureCache(ctx)
	}
	return nil
}
