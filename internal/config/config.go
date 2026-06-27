package config

import "os"

type Config struct {
	DatabaseURL  string
	Port         string
	KafkaBrokers string
}

func Load() Config {
	return Config{
		DatabaseURL:  env("DATABASE_URL_GENERAL", "postgres://osship:osship_secret@postgres:5432/osship?sslmode=disable&search_path=general"),
		Port:         env("PORT", "8082"),
		KafkaBrokers: env("KAFKA_BROKERS", "kafka:9092"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
