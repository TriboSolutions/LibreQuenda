package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The OpenAPI description (docs/openapi/openapi.yaml) has every route of the code,
// and only those: a new route without its description, or a description left behind
// by a removed route, fails here. (/mcp is the MCP server, described in docs/mcp.md.)

var (
	routeA      = regexp.MustCompile(`Path:\s*"([^"]+)",\s*Method:\s*http\.Method(\w+)`)
	routeB      = regexp.MustCompile(`Method:\s*http\.Method(\w+),\s*Path:\s*"([^"]+)"`)
	healthRoute = regexp.MustCompile(`(?s)Path:\s*"(/health/\w+)"`)
	specPath    = regexp.MustCompile(`^  (/\S*):\s*$`)
	specVerb    = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
)

func TestOpenAPIHasEveryRoute(t *testing.T) {
	var code = map[string]bool{}
	var err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		var raw, readErr = os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var src = string(raw)
		for _, m := range routeA.FindAllStringSubmatch(src, -1) {
			code[strings.ToUpper(m[2])+" "+m[1]] = true
		}
		for _, m := range routeB.FindAllStringSubmatch(src, -1) {
			code[strings.ToUpper(m[1])+" "+m[2]] = true
		}
		for _, m := range healthRoute.FindAllStringSubmatch(src, -1) {
			code["GET "+m[1]] = true // health routes are declared apart (GET and HEAD)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(code) < 100 {
		t.Fatalf("found only %d routes in the code: the patterns no longer match how routes are declared", len(code))
	}

	var raw, readErr = os.ReadFile("../docs/openapi/openapi.yaml")
	if readErr != nil {
		t.Fatal(readErr)
	}
	var spec = map[string]bool{}
	var path string
	for _, line := range strings.Split(string(raw), "\n") {
		if m := specPath.FindStringSubmatch(line); m != nil {
			path = m[1]
		} else if m := specVerb.FindStringSubmatch(line); m != nil && path != "" {
			spec[strings.ToUpper(m[1])+" "+path] = true
		} else if line != "" && !strings.HasPrefix(line, " ") {
			path = "" // left paths: (components…)
		}
	}

	var missing, extra []string
	for route := range code {
		if !spec[route] {
			missing = append(missing, route)
		}
	}
	for route := range spec {
		if !code[route] {
			extra = append(extra, route)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("routes without their description in docs/openapi/openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("described in docs/openapi/openapi.yaml but not in the code:\n  %s", strings.Join(extra, "\n  "))
	}
}
