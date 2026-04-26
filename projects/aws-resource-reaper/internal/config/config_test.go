package config

import (
	"os"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	f := writeTmp(t, `
accounts:
  - id: "123456789012"
    role_arn: "arn:aws:iam::123456789012:role/Reaper"
regions:
  - us-east-1
  - eu-west-1
concurrency: 5
`)
	cfg, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(cfg.Accounts))
	}
	if len(cfg.Regions) != 2 {
		t.Errorf("expected 2 regions, got %d", len(cfg.Regions))
	}
	if cfg.Concurrency != 5 {
		t.Errorf("expected concurrency 5, got %d", cfg.Concurrency)
	}
}

func TestLoad_DefaultConcurrency(t *testing.T) {
	f := writeTmp(t, `
accounts:
  - id: "123456789012"
    role_arn: "arn:aws:iam::123456789012:role/Reaper"
regions:
  - us-east-1
`)
	cfg, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Concurrency != 10 {
		t.Errorf("expected default concurrency 10, got %d", cfg.Concurrency)
	}
}

func TestLoad_MissingAccounts(t *testing.T) {
	f := writeTmp(t, `regions: [us-east-1]`)
	_, err := Load(f)
	if err == nil {
		t.Fatal("expected error for missing accounts")
	}
}

func TestLoad_MissingRegions(t *testing.T) {
	f := writeTmp(t, `
accounts:
  - id: "123456789012"
    role_arn: "arn:aws:iam::123456789012:role/Reaper"
`)
	_, err := Load(f)
	if err == nil {
		t.Fatal("expected error for missing regions")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	f := writeTmp(t, `:::invalid`)
	_, err := Load(f)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}
