package diagnostic_test

import (
	"context"
	"strings"
	"testing"

	"tour_of_go/projects/triage-engine/internal/adapters/diagnostic"
)

func TestCheckBuildStatus(t *testing.T) {
	srv := diagnostic.NewMockServer()
	defer srv.Close()

	client := diagnostic.NewClient(srv.URL)
	result, err := client.CheckBuildStatus(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "SUCCESS") {
		t.Fatalf("expected SUCCESS in result, got: %s", result)
	}
	if !strings.Contains(result, "42") {
		t.Fatalf("expected build number 42 in result, got: %s", result)
	}
}
