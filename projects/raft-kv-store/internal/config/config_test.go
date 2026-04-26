package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`
node_id: node1
grpc_addr: ":9001"
http_addr: ":8001"
wal_dir: /tmp/raft-node1
peers:
  - id: node2
    grpc_addr: ":9002"
    http_addr: ":8002"
`)
	f.Close()
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NodeID != "node1" {
		t.Errorf("unexpected node_id: %s", cfg.NodeID)
	}
	if cfg.ElectionTimeoutMinMs != 150 {
		t.Errorf("expected default 150ms, got %d", cfg.ElectionTimeoutMinMs)
	}
}

func TestLoad_MissingNodeID(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`grpc_addr: ":9001"`)
	f.Close()
	if _, err := Load(f.Name()); err == nil {
		t.Fatal("expected error for missing node_id")
	}
}
