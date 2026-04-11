package config

import "os"

type APIConfig struct {
	Addr string
}

type RelayConfig struct {
	Addr string
}

type Config struct {
	API   APIConfig
	Relay RelayConfig
}

func Load() Config {
	return Config{
		API: APIConfig{
			Addr: envOrDefault("CIRCLE_LINK_API_ADDR", ":8080"),
		},
		Relay: RelayConfig{
			Addr: envOrDefault("CIRCLE_LINK_RELAY_ADDR", ":8081"),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
