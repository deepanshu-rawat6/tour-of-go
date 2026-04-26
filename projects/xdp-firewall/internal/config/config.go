package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interface           string `yaml:"interface"`
	HTTPAddr            string `yaml:"http_addr"`
	RulesFile           string `yaml:"rules_file"`
	MetricsPollInterval string `yaml:"metrics_poll_interval"`
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
	if cfg.Interface == "" {
		return nil, fmt.Errorf("interface is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.RulesFile == "" {
		cfg.RulesFile = "/etc/xdp-firewall/rules.json"
	}
	if cfg.MetricsPollInterval == "" {
		cfg.MetricsPollInterval = "2s"
	}
	return &cfg, nil
}

func (c *Config) PollInterval() (time.Duration, error) {
	return time.ParseDuration(c.MetricsPollInterval)
}
