package storage

import "testing"

func TestValidRef(t *testing.T) {
	var cases = map[string]bool{
		"councils/01a1e2.webp":       true,
		"facilities/01a1e2.jpg":      true,
		"councils/":                  false,
		"other/01a1e2.png":           false,
		"councils/../secret":         false,
		"councils/a/b.png":           false,
		"/councils/01a1e2.png":       false,
		"https://evil.example/x.png": false,
	}
	for ref, want := range cases {
		if got := ValidRef(ref); got != want {
			t.Errorf("ValidRef(%q) = %v, want %v", ref, got, want)
		}
	}
	if URL("councils/a.png") != "/api/media/councils/a.png" || URL("") != "" {
		t.Error("URL must map a reference to /api/media/<ref> and keep empty empty")
	}
}
