package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`alerts: {stdout: true}`)
	f.Close()
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Namespaces) != 1 || cfg.Namespaces[0] != "*" {
		t.Errorf("expected default namespace [*], got %v", cfg.Namespaces)
	}
	if cfg.DedupWindowSeconds != 300 {
		t.Errorf("expected default 300s, got %d", cfg.DedupWindowSeconds)
	}
}

func TestLoad_Custom(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(`
namespaces: ["default", "kube-system"]
dedup_window_seconds: 60
ignore_reasons: ["Pulling", "Pulled"]
severity:
  OOMKilled: critical
  CrashLoopBackOff: critical
`)
	f.Close()
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Namespaces) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(cfg.Namespaces))
	}
	if cfg.Severity["OOMKilled"] != "critical" {
		t.Errorf("expected OOMKilled=critical, got %s", cfg.Severity["OOMKilled"])
	}
	if len(cfg.IgnoreReasons) != 2 {
		t.Errorf("expected 2 ignore reasons, got %d", len(cfg.IgnoreReasons))
	}
}
