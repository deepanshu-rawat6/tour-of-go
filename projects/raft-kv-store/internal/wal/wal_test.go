package wal

import (
	"testing"
)

func TestWAL_AppendAndReadAll(t *testing.T) {
	w, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	entries := []Entry{
		{Term: 1, Index: 1, Command: []byte(`{"op":"put","key":"a","value":"1"}`)},
		{Term: 1, Index: 2, Command: []byte(`{"op":"put","key":"b","value":"2"}`)},
		{Term: 2, Index: 3, Command: []byte(`{"op":"delete","key":"a"}`)},
	}
	for _, e := range entries {
		if err := w.Append(e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[2].Index != 3 {
		t.Errorf("unexpected index: %d", got[2].Index)
	}
}

func TestWAL_Truncate(t *testing.T) {
	w, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	for i := uint64(1); i <= 5; i++ {
		w.Append(Entry{Term: 1, Index: i, Command: []byte("cmd")})
	}

	if err := w.Truncate(3); err != nil {
		t.Fatal(err)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries after truncate at 3, got %d", len(got))
	}
	if got[len(got)-1].Index != 2 {
		t.Errorf("expected last index 2, got %d", got[len(got)-1].Index)
	}
}

func TestWAL_Recovery(t *testing.T) {
	dir := t.TempDir()
	w, _ := Open(dir)
	w.Append(Entry{Term: 1, Index: 1, Command: []byte("cmd1")})
	w.Append(Entry{Term: 1, Index: 2, Command: []byte("cmd2")})
	w.Close()

	// Reopen — simulates process restart
	w2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	entries, err := w2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 recovered entries, got %d", len(entries))
	}
}
