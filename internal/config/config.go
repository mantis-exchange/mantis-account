package config

import "os"

type Config struct {
	Port      string
	DBURL     string
	JWTSecret string
	JWTExpiry string
}

func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "50053"),
		DBURL:     getEnv("DB_URL", "postgres://mantis:mantis@localhost:5432/mantis_account?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "changeme"),
		JWTExpiry: getEnv("JWT_EXPIRY", "24h"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
