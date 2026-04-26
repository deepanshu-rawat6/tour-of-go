package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type BackendConfig struct {
	Type     string `yaml:"type"`      // "s3" or "local"
	Bucket   string `yaml:"bucket"`
	Key      string `yaml:"key"`
	Region   string `yaml:"region"`
	FilePath string `yaml:"file_path"` // for local backend
}

type SlackConfig   struct{ WebhookURL string `yaml:"webhook_url"` }
type DiscordConfig struct{ WebhookURL string `yaml:"webhook_url"` }

type AlertsConfig struct {
	Slack   SlackConfig   `yaml:"slack"`
	Discord DiscordConfig `yaml:"discord"`
	Stdout  bool          `yaml:"stdout"`
}

type Config struct {
	Backend        BackendConfig              `yaml:"backend"`
	Alerts         AlertsConfig               `yaml:"alerts"`
	IgnoreFields   map[string][]string        `yaml:"ignore_fields"` // resource_type → []field_path
	Interval       string                     `yaml:"interval"`      // e.g. "5m"
	Concurrency    int                        `yaml:"concurrency"`
	DriftStateFile string                     `yaml:"drift_state_file"`
	AWSRegion      string                     `yaml:"aws_region"`
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
	if cfg.Backend.Type == "" {
		return nil, fmt.Errorf("backend.type is required (s3 or local)")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.DriftStateFile == "" {
		cfg.DriftStateFile = "/tmp/tf-drift-state.json"
	}
	if cfg.Interval == "" {
		cfg.Interval = "5m"
	}
	return &cfg, nil
}

func (c *Config) ParseInterval() (time.Duration, error) {
	return time.ParseDuration(c.Interval)
}
