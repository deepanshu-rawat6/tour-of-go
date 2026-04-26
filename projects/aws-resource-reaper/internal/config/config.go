package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AccountConfig struct {
	ID      string `yaml:"id"`
	RoleARN string `yaml:"role_arn"`
}

type Config struct {
	Accounts    []AccountConfig `yaml:"accounts"`
	Regions     []string        `yaml:"regions"`
	Concurrency int             `yaml:"concurrency"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if len(cfg.Accounts) == 0 {
		return nil, fmt.Errorf("config must specify at least one account")
	}
	if len(cfg.Regions) == 0 {
		return nil, fmt.Errorf("config must specify at least one region")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	return &cfg, nil
}
