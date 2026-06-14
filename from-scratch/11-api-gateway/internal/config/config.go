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
	JWT struct {
		Secret string        `yaml:"secret"`
		Expiry time.Duration `yaml:"expiry"`
	} `yaml:"jwt"`
	RateLimit struct {
		Rate  float64 `yaml:"rate"`
		Burst float64 `yaml:"burst"`
	} `yaml:"rateLimit"`
	Upstreams map[string]string `yaml:"upstreams"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Addr = ":8080"
	cfg.JWT.Secret = "change-me-in-production"
	cfg.JWT.Expiry = time.Hour
	cfg.RateLimit.Rate = 100
	cfg.RateLimit.Burst = 100
	cfg.Upstreams = map[string]string{
		"users":   "http://localhost:8081",
		"billing": "http://localhost:8082",
	}

	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
