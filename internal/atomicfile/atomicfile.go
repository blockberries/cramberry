// Package atomicfile writes to a path via a temp file + rename so that a
// crash or signal mid-write cannot leave a partially-written file at the
// destination.
package atomicfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Write invokes write with an io.Writer that targets a temp file in the
// destination's directory; on success it atomically renames the temp file
// over path. On failure (or panic) the temp file is removed and path is
// untouched.
//
// On POSIX systems (Linux/macOS), the parent directory is fsync'd after
// the rename so the new entry is durable across power loss. This is the
// difference between "atomic for crashes mid-write" and "atomic for power
// loss" — both matter for a tool that writes consensus-relevant artifacts.
func Write(path string, perm os.FileMode, write func(io.Writer) error) (retErr error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".cramberry-*.tmp")
	if err != nil {
		return fmt.Errorf("atomicfile: create temp: %w", err)
	}
	tmp := f.Name()
	// Idempotent close + best-effort cleanup. The deferred close handles
	// the panic-in-write path that previously leaked the file descriptor:
	// the Remove cleanup ran but f.Close did not.
	closed := false
	closeFn := func() error {
		if closed {
			return nil
		}
		closed = true
		return f.Close()
	}
	committed := false
	defer func() {
		_ = closeFn()
		if !committed {
			_ = os.Remove(tmp)
		}
	}()

	if err := write(f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("atomicfile: sync: %w", err)
	}
	if err := closeFn(); err != nil {
		return fmt.Errorf("atomicfile: close: %w", err)
	}
	if err := os.Chmod(tmp, perm); err != nil {
		return fmt.Errorf("atomicfile: chmod: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("atomicfile: rename: %w", err)
	}
	committed = true

	// fsync the parent directory so the new entry survives a power loss.
	// Skip on Windows (no equivalent semantics) and treat any failure as
	// non-fatal: the rename succeeded; durability is best-effort.
	if runtime.GOOS != "windows" {
		if dirF, err := os.Open(dir); err == nil {
			_ = dirF.Sync()
			_ = dirF.Close()
		}
	}
	return nil
}
