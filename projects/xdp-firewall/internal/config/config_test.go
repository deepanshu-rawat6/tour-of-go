package config

import (
	"os"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`
interface: eth0
http_addr: ":9090"
rules_file: /tmp/rules.json
metrics_poll_interval: "5s"
`)
	f.Close()
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interface != "eth0" {
		t.Errorf("unexpected interface: %s", cfg.Interface)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("unexpected http_addr: %s", cfg.HTTPAddr)
	}
}

func TestLoad_Defaults(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`interface: lo`)
	f.Close()
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected default :8080, got %s", cfg.HTTPAddr)
	}
	if cfg.MetricsPollInterval != "2s" {
		t.Errorf("expected default 2s, got %s", cfg.MetricsPollInterval)
	}
}

func TestLoad_MissingInterface(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`http_addr: ":8080"`)
	f.Close()
	if _, err := Load(f.Name()); err == nil {
		t.Fatal("expected error for missing interface")
	}
}
