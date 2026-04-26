package httpadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockEngine struct {
	rules []string
	err   error
}

func (m *mockEngine) BlockCIDR(cidr string) error   { m.rules = append(m.rules, cidr); return m.err }
func (m *mockEngine) UnblockCIDR(cidr string) error { return m.err }
func (m *mockEngine) BatchBlock(cidrs []string) error {
	m.rules = append(m.rules, cidrs...)
	return m.err
}
func (m *mockEngine) ImportList(cidrs []string) error {
	m.rules = append(m.rules, cidrs...)
	return m.err
}
func (m *mockEngine) ListRules() []string { return m.rules }

func setup(t *testing.T) (*mockEngine, *http.ServeMux) {
	t.Helper()
	eng := &mockEngine{}
	h := New(eng)
	mux := http.NewServeMux()
	h.Register(mux)
	return eng, mux
}

func TestPostBlacklist(t *testing.T) {
	eng, mux := setup(t)
	body := `{"cidr":"10.0.0.0/8"}`
	req := httptest.NewRequest(http.MethodPost, "/api/blacklist", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if len(eng.rules) != 1 || eng.rules[0] != "10.0.0.0/8" {
		t.Errorf("expected rule added, got %v", eng.rules)
	}
}

func TestGetBlacklist(t *testing.T) {
	eng, mux := setup(t)
	eng.rules = []string{"10.0.0.0/8", "192.168.0.0/16"}
	req := httptest.NewRequest(http.MethodGet, "/api/blacklist", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 2 {
		t.Errorf("expected count 2, got %v", resp["count"])
	}
}

func TestDeleteBlacklist(t *testing.T) {
	_, mux := setup(t)
	body := `{"cidr":"10.0.0.0/8"}`
	req := httptest.NewRequest(http.MethodDelete, "/api/blacklist", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestBatchBlacklist(t *testing.T) {
	eng, mux := setup(t)
	body := `{"cidrs":["10.0.0.0/8","172.16.0.0/12"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/blacklist/batch", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if len(eng.rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(eng.rules))
	}
}

func TestImportBlacklist_JSON(t *testing.T) {
	eng, mux := setup(t)
	body := `{"cidrs":["10.0.0.0/8","192.168.0.0/16","172.16.0.0/12"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/blacklist/import", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if len(eng.rules) != 3 {
		t.Errorf("expected 3 imported rules, got %d", len(eng.rules))
	}
}

func TestImportBlacklist_File(t *testing.T) {
	eng, mux := setup(t)
	// Build multipart form
	var buf bytes.Buffer
	boundary := "testboundary"
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"cidrs.txt\"\r\n\r\n")
	buf.WriteString("10.0.0.0/8\n# comment\n192.168.0.0/16\n")
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	req := httptest.NewRequest(http.MethodPost, "/api/blacklist/import", &buf)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(eng.rules) != 2 {
		t.Errorf("expected 2 imported rules (comment skipped), got %d", len(eng.rules))
	}
}

func TestPostBlacklist_InvalidCIDR(t *testing.T) {
	eng, mux := setup(t)
	eng.err = fmt.Errorf("invalid CIDR")
	body := `{"cidr":"not-a-cidr"}`
	req := httptest.NewRequest(http.MethodPost, "/api/blacklist", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
