package atomicfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite_SuccessReplacesTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Write(path, 0o644, func(w io.Writer) error {
		_, err := io.WriteString(w, "new content")
		return err
	})
	if err != nil {
		t.Fatalf("Write returned %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new content" {
		t.Errorf("file = %q, want %q", got, "new content")
	}
}

func TestWrite_FailureLeavesTargetUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("boom")
	err := Write(path, 0o644, func(w io.Writer) error {
		// Write some bytes, then fail. The partial bytes must not
		// reach the destination file.
		_, _ = io.WriteString(w, "garbage")
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Write error = %v, want %v", err, wantErr)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Errorf("destination clobbered: got %q, want %q", got, "original")
	}

	// And no .tmp file should be left behind in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
