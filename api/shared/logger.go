package shared

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// LOG_DIR: where the API writes its log files; "" writes to the console only (tests,
// a bare go run). In the containers it is a volume, so the files outlive them.
var LOG_DIR = os.Getenv("LOG_DIR")

// LOG_KEEP_DAYS: how long rotated api-*.log files are kept (30 days unless set).
var LOG_KEEP_DAYS = envInt("LOG_KEEP_DAYS", 30)

// Logger writes each line to the console and, with LOG_DIR, to LOG_DIR/api.log,
// which rotates at logMaxBytes into api-<time>.log. A background goroutine does
// the writing, so a slow disk or stdout never stalls a request; Flush waits for it.
var Logger = newAppLogger()

const logMaxBytes = 20 << 20
const logBuffer = 4096

type appLogger struct {
	lines   chan logLine
	dropped atomic.Uint64
	file    *os.File
	size    int64
}

type logLine struct {
	text    string
	flushed chan struct{}
}

func newAppLogger() *appLogger {
	var l = &appLogger{lines: make(chan logLine, logBuffer)}
	go l.writeLoop()
	// what the standard library's log prints (some jobs use it) goes the same way
	log.SetFlags(0)
	log.SetOutput(stdlibWriter{l})
	return l
}

func (l *appLogger) Debug(msg string, args ...any) { l.log("DEBUG", 2, msg, args...) }
func (l *appLogger) Info(msg string, args ...any)  { l.log("INFO", 2, msg, args...) }
func (l *appLogger) Warn(msg string, args ...any)  { l.log("WARN", 2, msg, args...) }
func (l *appLogger) Error(msg string, args ...any) { l.log("ERROR", 2, msg, args...) }
func (l *appLogger) Fatal(msg string, args ...any) {
	l.log("FATAL", 2, msg, args...)
	l.Flush()
	os.Exit(1)
}

// Flush blocks until every line logged so far is written.
func (l *appLogger) Flush() {
	var done = make(chan struct{})
	l.lines <- logLine{flushed: done}
	<-done
}

func (l *appLogger) log(level string, skip int, msg string, args ...any) {
	var caller = "?"
	if _, file, line, ok := runtime.Caller(skip); ok {
		caller = fmt.Sprintf("%s:%d", shortPath(file), line)
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.enqueue(fmt.Sprintf("[%s] [%s] [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), level, caller, msg))
}

func (l *appLogger) enqueue(text string) {
	select {
	case l.lines <- logLine{text: text}:
	default:
		l.dropped.Add(1)
	}
}

func (l *appLogger) writeLoop() {
	for line := range l.lines {
		if dropped := l.dropped.Swap(0); dropped > 0 {
			l.write(fmt.Sprintf("[%s] [WARN] [logger] %d lines dropped\n", time.Now().Format("2006-01-02 15:04:05.000"), dropped))
		}
		if line.flushed != nil {
			if l.file != nil {
				l.file.Sync()
			}
			close(line.flushed)
			continue
		}
		l.write(line.text)
	}
}

func (l *appLogger) write(text string) {
	io.WriteString(os.Stdout, text)
	if LOG_DIR == "" {
		return
	}
	if l.file == nil && !l.open() {
		return
	}
	var n, err = l.file.WriteString(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log file: %v\n", err)
		l.file.Close()
		l.file = nil
		return
	}
	l.size += int64(n)
	if l.size >= logMaxBytes {
		l.rotate()
	}
}

func (l *appLogger) open() bool {
	if err := os.MkdirAll(LOG_DIR, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "log directory: %v\n", err)
		return false
	}
	var file, err = os.OpenFile(filepath.Join(LOG_DIR, "api.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log file: %v\n", err)
		return false
	}
	l.file = file
	l.size = 0
	if info, err := file.Stat(); err == nil {
		l.size = info.Size()
	}
	return true
}

// rotate: api.log becomes api-<time>.log, and the rotated ones past LOG_KEEP_DAYS go.
func (l *appLogger) rotate() {
	l.file.Close()
	l.file = nil
	var current = filepath.Join(LOG_DIR, "api.log")
	var rotated = filepath.Join(LOG_DIR, "api-"+time.Now().Format("20060102-150405.000")+".log")
	if err := os.Rename(current, rotated); err != nil {
		fmt.Fprintf(os.Stderr, "log rotation: %v\n", err)
	}
	pruneLogs("api-", time.Duration(LOG_KEEP_DAYS)*24*time.Hour)
}

// pruneLogs removes LOG_DIR/<prefix>*.log files last written before maxAge ago.
func pruneLogs(prefix string, maxAge time.Duration) {
	var entries, err = os.ReadDir(LOG_DIR)
	if err != nil {
		return
	}
	var cutoff = time.Now().Add(-maxAge)
	for _, entry := range entries {
		var name = entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".log") {
			continue
		}
		if info, err := entry.Info(); err == nil && info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(LOG_DIR, name))
		}
	}
}

// stdlibWriter: lines from the standard library's log package, as INFO.
type stdlibWriter struct{ l *appLogger }

func (w stdlibWriter) Write(p []byte) (int, error) {
	var caller = "log"
	// Write <- log.(*Logger).output <- log.Printf <- the caller: 3 frames up
	if _, file, line, ok := runtime.Caller(3); ok {
		caller = fmt.Sprintf("%s:%d", shortPath(file), line)
	}
	var msg = strings.TrimRight(string(p), "\n")
	w.l.enqueue(fmt.Sprintf("[%s] [INFO] [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), caller, msg))
	return len(p), nil
}

// shortPath: the last two elements of a source path (package/file.go).
func shortPath(file string) string {
	var dir, name = filepath.Split(file)
	return filepath.Join(filepath.Base(dir), name)
}

func envInt(key string, fallback int) int {
	var n int
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}
