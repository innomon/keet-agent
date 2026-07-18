package logger

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/innomon/keet-adk-gateway/pkg/config"
)

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)

	mh := NewMultiHandler(h1, h2)

	// Test Enabled
	if !mh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected MultiHandler to be enabled for LevelInfo")
	}

	// Test Handle
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test log message", 0)
	err := mh.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("failed to handle record: %v", err)
	}

	if !bytes.Contains(buf1.Bytes(), []byte("test log message")) {
		t.Errorf("buf1 expected to contain log message, got: %s", buf1.String())
	}
	if !bytes.Contains(buf2.Bytes(), []byte("test log message")) {
		t.Errorf("buf2 expected to contain log message, got: %s", buf2.String())
	}
}

func TestLoggerInit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_init_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	// We import "os" inside logger_test, so we need to add "os" to imports
	defer os.RemoveAll(tempDir)

	cfg := config.Config{
		LogLevel:       "DEBUG",
		ConsoleEnabled: false,
		FileEnabled:    true,
		LogDir:         tempDir,
		LogFileName:    "test_init.log",
		LogMaxSizeMB:   1,
		LogMaxBackups:  1,
	}

	logger, err := Init(cfg)
	if err != nil {
		t.Fatalf("logger init failed: %v", err)
	}

	logger.Info("init test message")

	// Read log file
	logFilePath := filepath.Join(tempDir, "test_init.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	if !strings.Contains(string(content), "init test message") {
		t.Errorf("log file expected to contain 'init test message', got: %s", string(content))
	}
}

// Custom handler to capture program counter (PC) for verification
type captureHandler struct {
	lastRecord slog.Record
}

func (c *captureHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (c *captureHandler) Handle(ctx context.Context, r slog.Record) error {
	c.lastRecord = r
	return nil
}

func (c *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return c }
func (c *captureHandler) WithGroup(name string) slog.Handler      { return c }

func TestCustomLogger_CallerPreservation(t *testing.T) {
	ch := &captureHandler{}
	testLog := slog.New(ch)

	cl := &CustomLogger{
		logger: testLog,
		module: "TEST_MOD",
	}

	// This is line 97 in this file
	cl.Infof("hello %s", "world")

	if cl.module != "TEST_MOD" {
		t.Errorf("expected module TEST_MOD, got: %s", cl.module)
	}

	if !strings.Contains(ch.lastRecord.Message, "[TEST_MOD] hello world") {
		t.Errorf("expected message to contain '[TEST_MOD] hello world', got: %s", ch.lastRecord.Message)
	}

	// Verify the PC records the correct calling function and file
	var pcs [1]uintptr
	pcs[0] = ch.lastRecord.PC
	frames := runtime.CallersFrames(pcs[:])
	frame, _ := frames.Next()

	if !strings.Contains(frame.Function, "TestCustomLogger_CallerPreservation") {
		t.Errorf("expected calling function to contain 'TestCustomLogger_CallerPreservation', got: %s", frame.Function)
	}
	if !strings.Contains(frame.File, "logger_test.go") {
		t.Errorf("expected calling file to contain 'logger_test.go', got: %s", frame.File)
	}
}
