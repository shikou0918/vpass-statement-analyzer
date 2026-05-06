package config

import "os"

type Config struct {
	Addr          string
	DatabasePath  string
	AllowedOrigin string
}

func Load() Config {
	return Config{
		Addr:          env("APP_ADDR", ":8080"),
		DatabasePath:  env("DATABASE_PATH", "vpass.db"),
		AllowedOrigin: env("ALLOWED_ORIGIN", "http://localhost:5173"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
