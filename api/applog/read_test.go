package applog

import (
	"librequenda/shared"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestReadNewestFirstFilteredAndPaged(t *testing.T) {
	var dir = t.TempDir()
	shared.LOG_DIR = dir
	t.Cleanup(func() { shared.LOG_DIR = "" })
	// the rotated file holds the older lines, api.log the newer ones
	write(t, filepath.Join(dir, "api-20261101-000000.000.log"),
		"[2026-11-01 10:00:00.000] [INFO] [cronJobs/a.go:1] old info\n"+
			"[2026-11-01 10:00:01.000] [ERROR] [mailer/m.go:2] old smtp failure\n")
	write(t, filepath.Join(dir, "api.log"),
		"[2026-11-07 09:00:00.000] [INFO] [main.go:3] started\n"+
			"not a log line\n"+
			"[2026-11-07 09:00:01.000] [WARN] [mailer/m.go:4] email reminder to x@example.com not sent: SMTP refused\n")

	var all = Read(ParseFilters(url.Values{}))
	if len(all.Lines) != 4 || all.Lines[0].Text != "email reminder to x@example.com not sent: SMTP refused" || all.Lines[3].Text != "old info" {
		t.Fatalf("newest first across files, got %+v", all.Lines)
	}
	var errors = Read(ParseFilters(url.Values{"level": {"warn,error"}, "q": {"SMTP"}}))
	if len(errors.Lines) != 2 || errors.Lines[0].Level != "WARN" || errors.Lines[1].Level != "ERROR" {
		t.Fatalf("level and text, got %+v", errors.Lines)
	}
	var older = Read(ParseFilters(url.Values{"before": {"2026-11-07 09:00:00.000"}}))
	if len(older.Lines) != 2 || older.Lines[0].Text != "old smtp failure" {
		t.Fatalf("before, got %+v", older.Lines)
	}
	if f := ParseFilters(url.Values{"level": {"nope"}, "before": {"yesterday"}}); len(f.Levels) != 0 || f.Before != "" {
		t.Fatalf("invalid values are no filter, got %+v", f)
	}
}

func TestReadWithoutLogDir(t *testing.T) {
	shared.LOG_DIR = ""
	if page := Read(Filters{}); page.Enabled || len(page.Lines) != 0 {
		t.Fatalf("no LOG_DIR, no lines: %+v", page)
	}
}

func write(t *testing.T, name, text string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
