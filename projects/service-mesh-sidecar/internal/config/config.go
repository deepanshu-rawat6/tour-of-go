package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy struct {
		ListenAddr   string `yaml:"listenAddr"`
		UpstreamAddr string `yaml:"upstreamAddr"`
		MetricsAddr  string `yaml:"metricsAddr"`
	} `yaml:"proxy"`
	RateLimit struct {
		Rate  float64 `yaml:"rate"`  // tokens per second
		Burst float64 `yaml:"burst"` // max burst
	} `yaml:"rateLimit"`
	CircuitBreaker struct {
		Threshold  int           `yaml:"threshold"`
		RetryAfter time.Duration `yaml:"retryAfter"`
	} `yaml:"circuitBreaker"`
	Health struct {
		Interval time.Duration `yaml:"interval"`
		Timeout  time.Duration `yaml:"timeout"`
	} `yaml:"health"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Proxy.ListenAddr = ":8080"
	cfg.Proxy.UpstreamAddr = "localhost:9090"
	cfg.Proxy.MetricsAddr = ":9091"
	cfg.RateLimit.Rate = 100
	cfg.RateLimit.Burst = 20
	cfg.CircuitBreaker.Threshold = 5
	cfg.CircuitBreaker.RetryAfter = 30 * time.Second
	cfg.Health.Interval = 10 * time.Second
	cfg.Health.Timeout = 2 * time.Second

	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
