package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

type Config struct {
	AppEnv              string
	AutoMigrate         bool
	CamaraBaseURL       string
	CacheCheckInterval  time.Duration
	CacheTTL            time.Duration
	CORSAllowedOrigins  []string
	DBConnectionString  string
	InternalAPIKey      string
	InternalReadAPIKey  string
	Port                string
	RedisURL            string
	RefreshInterval     time.Duration
	SenadoBaseURL       string
	ExternalHTTPTimeout time.Duration
	TrustedProxies      []string
}

func Load() Config {
	return Config{
		AppEnv:              envOrDefault("APP_ENV", EnvironmentDevelopment),
		AutoMigrate:         boolOrDefault("AUTO_MIGRATE", false),
		CamaraBaseURL:       envOrDefault("CAMARA_BASE_URL", "https://dadosabertos.camara.leg.br/api/v2"),
		CacheCheckInterval:  durationOrDefault("PARLIAMENTARIANS_CACHE_CHECK_INTERVAL", time.Minute),
		CacheTTL:            durationOrDefault("PARLIAMENTARIANS_CACHE_TTL", 8*time.Hour),
		CORSAllowedOrigins:  splitList(envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:8080")),
		DBConnectionString:  required("DB_CONNECTION_STRING"),
		InternalAPIKey:      required("INTERNAL_API_KEY"),
		InternalReadAPIKey:  required("INTERNAL_READ_API_KEY"),
		Port:                envOrDefault("PORT", "3001"),
		RedisURL:            required("REDIS_URL"),
		RefreshInterval:     durationOrDefault("PARLIAMENTARIANS_REFRESH_INTERVAL", time.Hour),
		SenadoBaseURL:       envOrDefault("SENADO_BASE_URL", "https://legis.senado.leg.br/dadosabertos"),
		ExternalHTTPTimeout: durationOrDefault("EXTERNAL_HTTP_TIMEOUT", 30*time.Second),
		TrustedProxies:      splitList(envOrDefault("TRUSTED_PROXIES", "127.0.0.1,::1")),
	}
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (c Config) IsProduction() bool { return strings.EqualFold(c.AppEnv, EnvironmentProduction) }

func required(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		panic(key + "_ENV_NOT_SET")
	}
	return value
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(key + "_ENV_INVALID")
	}
	return parsed
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}
