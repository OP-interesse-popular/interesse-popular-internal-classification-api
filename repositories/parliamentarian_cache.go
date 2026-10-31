package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	// ParliamentariansCacheKey contains only people currently returned by Câmara and Senado.
	ParliamentariansCacheKey = "opip:parliamentarians:v1"
	// PoliticiansCacheKey contains all known identities and their mandate history,
	// including elected_pending and former mandates.
	PoliticiansCacheKey = "opip:politicians:v1"
)

var ErrCacheEmpty = errors.New("parliamentarians cache is empty")

type ParliamentarianCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewParliamentarianCache(url string, ttl time.Duration) (*ParliamentarianCache, error) {
	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &ParliamentarianCache{client: redis.NewClient(options), ttl: ttl}, nil
}
func (r *ParliamentarianCache) Ping(ctx context.Context) error { return r.client.Ping(ctx).Err() }
func (r *ParliamentarianCache) Set(ctx context.Context, items []models.Parliamentarian) error {
	value, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, ParliamentariansCacheKey, value, r.ttl).Err()
}

func (r *ParliamentarianCache) SetAll(ctx context.Context, items []models.Politician) error {
	value, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, PoliticiansCacheKey, value, r.ttl).Err()
}
func (r *ParliamentarianCache) IsEmpty(ctx context.Context) (bool, error) {
	count, err := r.client.Exists(ctx, ParliamentariansCacheKey).Result()
	return count == 0, err
}

func (r *ParliamentarianCache) IsAllEmpty(ctx context.Context) (bool, error) {
	count, err := r.client.Exists(ctx, PoliticiansCacheKey).Result()
	return count == 0, err
}
