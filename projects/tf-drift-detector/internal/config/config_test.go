package config

import (
	"os"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	f := tmpFile(t, `
backend:
  type: local
  file_path: /tmp/terraform.tfstate
alerts:
  stdout: true
concurrency: 5
interval: "10m"
drift_state_file: /tmp/drift.json
aws_region: us-east-1
`)
	cfg, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend.Type != "local" {
		t.Errorf("expected local, got %s", cfg.Backend.Type)
	}
	if cfg.Concurrency != 5 {
		t.Errorf("expected 5, got %d", cfg.Concurrency)
	}
}

func TestLoad_Defaults(t *testing.T) {
	f := tmpFile(t, `backend: {type: s3, bucket: my-bucket, key: terraform.tfstate}`)
	cfg, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Concurrency != 10 {
		t.Errorf("expected default concurrency 10, got %d", cfg.Concurrency)
	}
	if cfg.Interval != "5m" {
		t.Errorf("expected default interval 5m, got %s", cfg.Interval)
	}
}

func TestLoad_MissingBackendType(t *testing.T) {
	f := tmpFile(t, `alerts: {stdout: true}`)
	if _, err := Load(f); err == nil {
		t.Fatal("expected error for missing backend type")
	}
}

func TestParseInterval(t *testing.T) {
	cfg := &Config{Interval: "10m"}
	d, err := cfg.ParseInterval()
	if err != nil || d.Minutes() != 10 {
		t.Errorf("unexpected interval: %v %v", d, err)
	}
}

func tmpFile(t *testing.T, content string) string {
	t.Helper()
	f, _ := os.CreateTemp(t.TempDir(), "*.yaml")
	f.WriteString(content)
	f.Close()
	return f.Name()
}
