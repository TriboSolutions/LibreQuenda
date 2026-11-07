package shared

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesTheFileWithTheCaller(t *testing.T) {
	LOG_DIR = t.TempDir()
	t.Cleanup(func() { Logger.Flush(); LOG_DIR = "" })
	Logger.Error("disk %s", "full")
	log.Printf("from the standard library")
	Logger.Flush()
	var data, err = os.ReadFile(filepath.Join(LOG_DIR, "api.log"))
	if err != nil {
		t.Fatal(err)
	}
	var text = string(data)
	if !strings.Contains(text, "[ERROR] [shared/logger_test.go:") || !strings.Contains(text, "disk full") {
		t.Fatalf("level, caller and text: %q", text)
	}
	if !strings.Contains(text, "[INFO] [shared/logger_test.go:") || !strings.Contains(text, "from the standard library") {
		t.Fatalf("the standard library's lines, with their caller: %q", text)
	}
}
