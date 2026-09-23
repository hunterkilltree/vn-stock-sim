// Package config reads process environment into a typed Config for main.go.
package config

import "os"

type Config struct {
	Port      string
	JWTSecret string
	// MarketDataSource selects the market.MarketDataProvider adapter:
	// "vci" (default) for the real Vietcap Securities live adapter
	// (backend/internal/market/vciprovider.go), "mock" for the
	// deterministic synthetic generator only -- see
	// phase-vci-market-data.md decision 6.
	MarketDataSource string
}

func Load() Config {
	return Config{
		Port:             getenv("PORT", "8080"),
		JWTSecret:        getenv("JWT_SECRET", "dev-secret-change-me"),
		MarketDataSource: getenv("MARKET_DATA_SOURCE", "vci"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
