// Package config reads process environment into a typed Config for main.go.
package config

import "os"

type Config struct {
	Port      string
	JWTSecret string
}

func Load() Config {
	return Config{
		Port:      getenv("PORT", "8080"),
		JWTSecret: getenv("JWT_SECRET", "dev-secret-change-me"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
