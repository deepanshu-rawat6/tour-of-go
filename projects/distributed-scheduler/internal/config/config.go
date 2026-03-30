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
		DSN             string `yaml:"dsn"`
		MaxOpenConns    int    `yaml:"maxOpenConns"`
		MaxIdleConns    int    `yaml:"maxIdleConns"`
	} `yaml:"database"`
	Redis struct {
		Addr string `yaml:"addr"`
	} `yaml:"redis"`
	Scheduler struct {
		PoolSize              int           `yaml:"poolSize"`
		ColdInterval          time.Duration `yaml:"coldInterval"`
		StaleAfter            time.Duration `yaml:"staleAfter"`
		RefreshInterval       time.Duration `yaml:"refreshInterval"`
		LongPublishedThreshold time.Duration `yaml:"longPublishedThreshold"`
		HeartbeatInterval     time.Duration `yaml:"heartbeatInterval"`
		ZombieCheckInterval   time.Duration `yaml:"zombieCheckInterval"`
	} `yaml:"scheduler"`
	Search struct {
		IndexPath string `yaml:"indexPath"`
	} `yaml:"search"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	// Defaults
	cfg.Server.Addr = ":8080"
	cfg.Database.MaxOpenConns = 20
	cfg.Database.MaxIdleConns = 5
	cfg.Redis.Addr = "localhost:6379"
	cfg.Scheduler.PoolSize = 10
	cfg.Scheduler.ColdInterval = 2 * time.Minute
	cfg.Scheduler.StaleAfter = 2 * time.Minute
	cfg.Scheduler.RefreshInterval = 5 * time.Minute
	cfg.Scheduler.LongPublishedThreshold = 4 * time.Hour
	cfg.Scheduler.HeartbeatInterval = 1 * time.Minute
	cfg.Scheduler.ZombieCheckInterval = 5 * time.Minute
	cfg.Search.IndexPath = ":memory:"

	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil // use defaults if file not found
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
