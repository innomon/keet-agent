package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogRotator_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_rotator_basic_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := "test.log"
	rotator, err := NewLogRotator(tempDir, filename, 1, 3)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}
	defer rotator.Close()

	data := []byte("hello log rotator\n")
	n, err := rotator.Write(data)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	path := filepath.Join(tempDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("expected content %q, got %q", string(data), string(content))
	}
}

func TestLogRotator_RotationAndPruning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_rotator_rot_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := "rotate.log"
	maxBackups := 2

	rotator, err := NewLogRotator(tempDir, filename, 1, maxBackups)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}
	defer rotator.Close()

	// Set small maxSize in bytes directly for testing rotation
	rotator.maxSize = 10

	// Write first log line (12 bytes, should fit first file but trigger rotation on next write)
	line1 := []byte("1234567890\n") // 11 bytes
	_, err = rotator.Write(line1)
	if err != nil {
		t.Fatalf("write line 1: %v", err)
	}

	// Write second log line. Since size (11) + line2 (11) = 22 > 10, it should rotate
	time.Sleep(10 * time.Millisecond) // Ensure timestamp change
	line2 := []byte("abcdefghij\n") // 11 bytes
	_, err = rotator.Write(line2)
	if err != nil {
		t.Fatalf("write line 2: %v", err)
	}

	// Write third log line (will rotate again)
	time.Sleep(10 * time.Millisecond)
	line3 := []byte("xyzwrtyuio\n") // 11 bytes
	_, err = rotator.Write(line3)
	if err != nil {
		t.Fatalf("write line 3: %v", err)
	}

	// Write fourth log line (will rotate again, pruning the oldest)
	time.Sleep(10 * time.Millisecond)
	line4 := []byte("1112223334\n") // 11 bytes
	_, err = rotator.Write(line4)
	if err != nil {
		t.Fatalf("write line 4: %v", err)
	}

	// Verify files in directory
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	var logFiles []string
	for _, f := range files {
		if !f.IsDir() {
			logFiles = append(logFiles, f.Name())
		}
	}

	fmt.Printf("Log files found: %v\n", logFiles)

	// We expect the active file (rotate.log) plus at most maxBackups (2) rotated files.
	// Total files should be 3: rotate.log and two backups.
	if len(logFiles) != 3 {
		t.Errorf("expected 3 log files, got %d: %v", len(logFiles), logFiles)
	}

	// Active file should contain line4
	activePath := filepath.Join(tempDir, filename)
	activeContent, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read active log: %v", err)
	}
	if string(activeContent) != string(line4) {
		t.Errorf("expected active content %q, got %q", string(line4), string(activeContent))
	}

	// Ensure active file is rotate.log
	foundActive := false
	for _, lf := range logFiles {
		if lf == "rotate.log" {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("rotate.log not found in log files: %v", logFiles)
	}
}
