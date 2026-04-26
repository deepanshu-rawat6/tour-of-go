package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type PeerConfig struct {
	ID       string `yaml:"id"`
	GRPCAddr string `yaml:"grpc_addr"`
	HTTPAddr string `yaml:"http_addr"`
}

type Config struct {
	NodeID               string       `yaml:"node_id"`
	GRPCAddr             string       `yaml:"grpc_addr"`
	HTTPAddr             string       `yaml:"http_addr"`
	WALDir               string       `yaml:"wal_dir"`
	Peers                []PeerConfig `yaml:"peers"`
	ElectionTimeoutMinMs int          `yaml:"election_timeout_min_ms"`
	ElectionTimeoutMaxMs int          `yaml:"election_timeout_max_ms"`
	HeartbeatMs          int          `yaml:"heartbeat_ms"`
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
	if cfg.NodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if cfg.ElectionTimeoutMinMs == 0 {
		cfg.ElectionTimeoutMinMs = 150
	}
	if cfg.ElectionTimeoutMaxMs == 0 {
		cfg.ElectionTimeoutMaxMs = 300
	}
	if cfg.HeartbeatMs == 0 {
		cfg.HeartbeatMs = 50
	}
	if cfg.WALDir == "" {
		cfg.WALDir = "/tmp/raft-" + cfg.NodeID
	}
	return &cfg, nil
}

// PeerHTTPAddr returns the HTTP address of a peer by ID.
func (c *Config) PeerHTTPAddr(id string) string {
	for _, p := range c.Peers {
		if p.ID == id {
			return p.HTTPAddr
		}
	}
	return ""
}
