package config

import (
	"flag"
	"os"
)

// Config holds all service configuration.
type Config struct {
	Addr   string
	APIKey string
}

// Load reads configuration from defaults, environment, and flags.
// Priority: defaults < env vars < flags.
func Load() Config {
	var cfg Config
	flag.StringVar(&cfg.Addr, "addr", getEnv("DIFFKIT_ADDR", ":8472"), "listen address")
	flag.StringVar(&cfg.APIKey, "api-key", getEnv("DIFFKIT_API_KEY", ""), "API key for authentication (optional, no auth if empty)")
	flag.Parse()
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
