package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	OpenAI struct {
		APIKey         string `yaml:"apiKey"`
		Model          string `yaml:"model"`
		EmbeddingModel string `yaml:"embeddingModel"`
	} `yaml:"openai"`
	Diagnostic struct {
		BaseURL string `yaml:"baseURL"`
	} `yaml:"diagnostic"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Addr = ":8080"
	cfg.Database.DSN = "postgres://triage:triage@localhost:5432/triage"
	cfg.OpenAI.Model = "gpt-4o-mini"
	cfg.OpenAI.EmbeddingModel = "text-embedding-3-small"
	cfg.Diagnostic.BaseURL = "http://localhost:9090"

	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	return cfg, yaml.Unmarshal(data, cfg)
}
