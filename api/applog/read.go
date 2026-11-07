// Package applog reads the API's log files (LOG_DIR/api.log and the rotated
// api-*.log) for the superadministrators: newest first, filtered.
package applog

import (
	"bufio"
	"librequenda/shared"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	pageSize  = 200
	maxQuery  = 100
	maxLineKB = 64
)

type Line struct {
	At     string `json:"at"` // as written: 2006-01-02 15:04:05.000, the server's time
	Level  string `json:"level"`
	Caller string `json:"caller"`
	Text   string `json:"text"`
}

type Page struct {
	Lines   []Line `json:"lines"`
	Next    string `json:"next"`    // pass as ?before= for older lines; "" at the end
	Enabled bool   `json:"enabled"` // false: LOG_DIR is not set, there are no files
}

type Filters struct {
	Levels map[string]bool // empty: every level
	Query  string          // lowercase
	Before string          // only lines written before this time (as in At)
}

var levels = map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true, "FATAL": true}
var stampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}$`)

// ParseFilters reads ?level=WARN,ERROR&q=&before=; a value that is not valid is no filter.
func ParseFilters(q url.Values) Filters {
	var f = Filters{Levels: map[string]bool{}}
	for _, level := range strings.Split(strings.ToUpper(q.Get("level")), ",") {
		if levels[level] {
			f.Levels[level] = true
		}
	}
	if query := strings.TrimSpace(q.Get("q")); query != "" && utf8.RuneCountInString(query) <= maxQuery {
		f.Query = strings.ToLower(query)
	}
	if before := q.Get("before"); stampPattern.MatchString(before) {
		f.Before = before
	}
	return f
}

// linePattern: [time] [LEVEL] [caller] text, as shared.Logger writes it.
var linePattern = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\] \[([A-Z]+)\] \[([^\]]*)\] (.*)$`)

// Read goes through the files from the newest, each from its end, until a page is full.
func Read(f Filters) Page {
	var page = Page{Lines: []Line{}, Enabled: shared.LOG_DIR != ""}
	if !page.Enabled {
		return page
	}
	for _, file := range files() {
		var lines = readFile(file)
		for i := len(lines) - 1; i >= 0; i-- {
			var line, ok = parse(lines[i])
			if !ok || (f.Before != "" && line.At >= f.Before) {
				continue
			}
			if len(f.Levels) > 0 && !f.Levels[line.Level] {
				continue
			}
			if f.Query != "" && !strings.Contains(strings.ToLower(line.Caller+" "+line.Text), f.Query) {
				continue
			}
			page.Lines = append(page.Lines, line)
			if len(page.Lines) == pageSize {
				page.Next = line.At
				return page
			}
		}
	}
	return page
}

// files: api.log, then the rotated ones from the newest (their names sort by time).
func files() []string {
	var rotated, _ = filepath.Glob(filepath.Join(shared.LOG_DIR, "api-*.log"))
	sort.Sort(sort.Reverse(sort.StringSlice(rotated)))
	return append([]string{filepath.Join(shared.LOG_DIR, "api.log")}, rotated...)
}

func readFile(name string) []string {
	var file, err = os.Open(name)
	if err != nil {
		return nil
	}
	defer file.Close()
	var lines []string
	var scanner = bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), maxLineKB*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func parse(text string) (Line, bool) {
	var m = linePattern.FindStringSubmatch(text)
	if m == nil {
		return Line{}, false
	}
	return Line{At: m[1], Level: m[2], Caller: m[3], Text: m[4]}, true
}
