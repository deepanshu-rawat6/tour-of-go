package image

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorePathHelpers(t *testing.T) {
	s := &Store{Base: "/tmp/gocker-test"}
	if got := s.ImageDir("alpine", "3.19"); got != "/tmp/gocker-test/images/alpine:3.19" {
		t.Errorf("unexpected ImageDir: %s", got)
	}
	if got := s.RootfsDir("alpine", "3.19"); got != "/tmp/gocker-test/images/alpine:3.19/rootfs" {
		t.Errorf("unexpected RootfsDir: %s", got)
	}
}

func TestParseRef(t *testing.T) {
	tests := []struct{ ref, name, tag string }{
		{"alpine:3.19", "alpine", "3.19"},
		{"alpine", "alpine", "latest"},
		{"ubuntu:22.04", "ubuntu", "22.04"},
	}
	for _, tc := range tests {
		n, tg := ParseRef(tc.ref)
		if n != tc.name || tg != tc.tag {
			t.Errorf("ParseRef(%q) = (%q,%q), want (%q,%q)", tc.ref, n, tg, tc.name, tc.tag)
		}
	}
}

func TestStoreExists(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Base: dir}
	rootfs := s.RootfsDir("alpine", "3.19")
	if s.Exists("alpine", "3.19") {
		t.Error("should not exist before creation")
	}
	os.MkdirAll(rootfs, 0755)
	if !s.Exists("alpine", "3.19") {
		t.Error("should exist after creation")
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Base: dir}
	os.MkdirAll(filepath.Join(dir, "images", "alpine:3.19"), 0755)
	os.MkdirAll(filepath.Join(dir, "images", "ubuntu:22.04"), 0755)
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 images, got %d", len(list))
	}
}
