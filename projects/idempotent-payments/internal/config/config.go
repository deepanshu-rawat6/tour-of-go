package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Idempotency struct {
		TTL             time.Duration `yaml:"ttl"`
		CleanupInterval time.Duration `yaml:"cleanupInterval"`
	} `yaml:"idempotency"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Addr = ":8080"
	cfg.Database.DSN = "postgres://payments:payments@localhost:5432/payments"
	cfg.Idempotency.TTL = 24 * time.Hour
	cfg.Idempotency.CleanupInterval = time.Hour

	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
