package aggregator_test

import (
	"testing"

	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/aggregator"
)

func TestAggregator_IngestAndSearch(t *testing.T) {
	a := aggregator.New()
	a.Ingest("app1", "INFO: server started")
	a.Ingest("app1", "ERROR: connection refused")
	a.Ingest("app2", "INFO: request received")

	results := a.Search("ERROR", "", 10)
	if len(results) != 1 || results[0].Source != "app1" {
		t.Fatalf("want 1 ERROR from app1, got %v", results)
	}

	results = a.Search("", "app2", 10)
	if len(results) != 1 {
		t.Fatalf("want 1 result for app2, got %d", len(results))
	}

	results = a.Search("INFO", "", 10)
	if len(results) != 2 {
		t.Fatalf("want 2 INFO results, got %d", len(results))
	}
}
