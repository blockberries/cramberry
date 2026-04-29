// Package atomicfile writes to a path via a temp file + rename so that a
// crash or signal mid-write cannot leave a partially-written file at the
// destination.
package atomicfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Write invokes write with an io.Writer that targets a temp file in the
// destination's directory; on success it atomically renames the temp file
// over path. On failure (or panic) the temp file is removed and path is
// untouched.
func Write(path string, perm os.FileMode, write func(io.Writer) error) (retErr error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".cramberry-*.tmp")
	if err != nil {
		return fmt.Errorf("atomicfile: create temp: %w", err)
	}
	tmp := f.Name()
	// Best-effort cleanup: removed on any return path that didn't rename.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmp)
		}
	}()

	if err := write(f); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("atomicfile: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("atomicfile: close: %w", err)
	}
	if err := os.Chmod(tmp, perm); err != nil {
		return fmt.Errorf("atomicfile: chmod: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("atomicfile: rename: %w", err)
	}
	committed = true
	return nil
}
