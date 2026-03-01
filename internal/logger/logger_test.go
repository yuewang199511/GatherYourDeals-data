package logger_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gatheryourdeals/data/internal/logger"
)

func TestRotatingWriter_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	w, err := logger.NewRotatingWriter(dir, "test", 1024, 2)
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}
	defer func() { _ = w.Close() }()

	_, err = w.Write([]byte("hello\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	files := logFiles(t, dir, "test")
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}
}

func TestRotatingWriter_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	// Max 50 bytes per file.
	w, err := logger.NewRotatingWriter(dir, "test", 50, 2)
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Write enough to trigger a rotation.
	msg := "this is a line that is over fifty bytes long, so it should trigger rotation\n"
	for i := 0; i < 3; i++ {
		_, err := w.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	files := logFiles(t, dir, "test")
	if len(files) > 2 {
		t.Errorf("expected at most 2 log files, got %d: %v", len(files), files)
	}
}

func TestRotatingWriter_PrunesOldFiles(t *testing.T) {
	dir := t.TempDir()
	// Max 10 bytes, keep 2 files.
	w, err := logger.NewRotatingWriter(dir, "test", 10, 2)
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Force multiple rotations.
	for i := 0; i < 10; i++ {
		_, err := w.Write([]byte("0123456789ab"))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	files := logFiles(t, dir, "test")
	if len(files) > 2 {
		t.Errorf("expected at most 2 log files after pruning, got %d: %v", len(files), files)
	}
}

func TestLogger_WritesToBothStdoutAndFile(t *testing.T) {
	dir := t.TempDir()
	l, err := logger.New(logger.Config{
		Dir:      dir,
		Prefix:   "test",
		MaxBytes: 1024,
		MaxFiles: 2,
	})
	if err != nil {
		t.Fatalf("New logger failed: %v", err)
	}
	defer func() { _ = l.Close() }()

	l.Info("test message", "key", "value")

	files := logFiles(t, dir, "test")
	if len(files) == 0 {
		t.Fatal("expected at least 1 log file")
	}

	content, err := os.ReadFile(filepath.Join(dir, files[0]))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "test message") {
		t.Errorf("expected 'test message' in log file, got: %s", string(content))
	}
}

func logFiles(t *testing.T, dir, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix+"-") && strings.HasSuffix(e.Name(), ".log") {
			names = append(names, e.Name())
		}
	}
	return names
}
