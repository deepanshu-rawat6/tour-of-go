package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type StorageConfig struct {
	SQLitePath string `yaml:"sqlite_path"`
	BlevePath  string `yaml:"bleve_path"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

type AlertsConfig struct {
	Slack  SlackConfig `yaml:"slack"`
	Stdout bool        `yaml:"stdout"`
}

type Config struct {
	Namespaces         []string          `yaml:"namespaces"`           // ["*"] for cluster-wide
	DedupWindowSeconds int               `yaml:"dedup_window_seconds"` // default 300
	Severity           map[string]string `yaml:"severity"`             // reason → "critical"|"warning"|"ignore"
	IgnoreReasons      []string          `yaml:"ignore_reasons"`
	Storage            StorageConfig     `yaml:"storage"`
	Alerts             AlertsConfig      `yaml:"alerts"`
	MetricsAddr        string            `yaml:"metrics_addr"`
	KubeConfig         string            `yaml:"kubeconfig"` // empty = in-cluster
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
	if len(cfg.Namespaces) == 0 {
		cfg.Namespaces = []string{"*"}
	}
	if cfg.DedupWindowSeconds <= 0 {
		cfg.DedupWindowSeconds = 300
	}
	if cfg.Storage.SQLitePath == "" {
		cfg.Storage.SQLitePath = "/data/events.db"
	}
	if cfg.Storage.BlevePath == "" {
		cfg.Storage.BlevePath = "/data/events.bleve"
	}
	if cfg.MetricsAddr == "" {
		cfg.MetricsAddr = ":9090"
	}
	return &cfg, nil
}

func (c *Config) DedupWindow() time.Duration {
	return time.Duration(c.DedupWindowSeconds) * time.Second
}
