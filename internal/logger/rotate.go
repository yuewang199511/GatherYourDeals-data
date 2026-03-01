package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Compile-time check: *RotatingWriter must satisfy io.Writer.
var _ io.Writer = (*RotatingWriter)(nil)

// RotatingWriter is an io.Writer that writes to a log file and rotates
// when the file exceeds a configured size. Old files are deleted so that
// at most maxFiles log files exist at any time. Each file is named with
// its creation timestamp in Y-M-D-H-M-S format.
type RotatingWriter struct {
	mu       sync.Mutex
	dir      string // directory for log files
	prefix   string // filename prefix, e.g. "gatheryourdeals"
	maxBytes int64  // max size per file in bytes
	maxFiles int    // max number of files to keep

	file    *os.File
	written int64
}

// NewRotatingWriter creates a new rotating file writer.
// It immediately opens the first log file.
func NewRotatingWriter(dir, prefix string, maxBytes int64, maxFiles int) (*RotatingWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	w := &RotatingWriter{
		dir:      dir,
		prefix:   prefix,
		maxBytes: maxBytes,
		maxFiles: maxFiles,
	}
	if err := w.rotate(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write implements io.Writer. It rotates the file if writing would exceed maxBytes.
func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.written+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.written += int64(n)
	return n, err
}

// Close closes the current log file.
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// rotate closes the current file, prunes old files, and opens a new one.
func (w *RotatingWriter) rotate() error {
	if w.file != nil {
		_ = w.file.Close()
	}

	if err := w.prune(); err != nil {
		return err
	}

	name := fmt.Sprintf("%s-%s.log", w.prefix, time.Now().Format("2006-01-02-15-04-05"))
	path := filepath.Join(w.dir, name)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	w.file = f
	w.written = 0

	// If the file already existed (same-second rotation), get the real size.
	info, err := f.Stat()
	if err == nil {
		w.written = info.Size()
	}

	return nil
}

// prune removes old log files so that after opening a new one the total
// count does not exceed maxFiles.
func (w *RotatingWriter) prune() error {
	files, err := w.logFiles()
	if err != nil {
		return err
	}

	// We are about to create a new file, so keep at most maxFiles-1 existing.
	excess := len(files) - (w.maxFiles - 1)
	if excess <= 0 {
		return nil
	}

	for i := 0; i < excess; i++ {
		path := filepath.Join(w.dir, files[i])
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove old log: %w", err)
		}
	}
	return nil
}

// logFiles returns existing log files sorted by name (oldest first).
func (w *RotatingWriter) logFiles() ([]string, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("read log directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, w.prefix+"-") && strings.HasSuffix(name, ".log") {
			names = append(names, name)
		}
	}
	sort.Strings(names) // timestamp in name → lexicographic = chronological
	return names, nil
}
