package storage

import (
	"librequenda/shared"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const council = "0192f5a0-1c2d-7e3f-8a9b-0c1d2e3f4a5b"

func TestValidRef(t *testing.T) {
	var cases = map[string]bool{
		council + "/councils/01a1e2.webp":                       true,
		council + "/facilities/01a1e2.jpg":                      true,
		council + "/issues/01a1e2.jpg":                          true,
		"users/01a1e2.jpg":                                      true,
		"branding/01a1e2.png":                                   true,
		"councils/01a1e2.webp":                                  false, // a council's file says whose it is
		"facilities/01a1e2.jpg":                                 false,
		council + "/users/01a1e2.jpg":                           false, // profile photos are the installation's
		"not-a-uuid/facilities/01a1e2.jpg":                      false,
		council + "/facilities/":                                false,
		council + "/facilities/../x.jpg":                        false,
		council + "/facilities/a/b.png":                         false,
		"users/":                                                false,
		"other/01a1e2.png":                                      false,
		"/users/01a1e2.png":                                     false,
		"https://evil.example/x.png":                            false,
		"0192F5A0-1C2D-7E3F-8A9B-0C1D2E3F4A5B/facilities/a.jpg": false, // one spelling per council
		council + "/facilities/pista-de-padel-1.jpg":            true,  // a seed image
		"users/" + strings.Repeat("a", 81) + ".jpg":             false, // longer than any name the API gives
		"users/01a1e2.gif":                                      false, // not an image type it takes
		"users/01A1E2.jpg":                                      false,
		"users/01a1e2":                                          false,
	}
	for ref, want := range cases {
		if got := ValidRef(ref); got != want {
			t.Errorf("ValidRef(%q) = %v, want %v", ref, got, want)
		}
	}
	if URL("") != "" {
		t.Error("URL must keep empty empty")
	}
}

func TestOwned(t *testing.T) {
	var id = uuid.MustParse(council)
	var other = uuid.MustParse("0192f5a0-1c2d-7e3f-8a9b-000000000000")
	var ref = CouncilRef(id, "facilities", "a.jpg")
	switch {
	case !Owned(ref, id, "facilities"):
		t.Error("a council owns its facility images")
	case Owned(ref, other, "facilities"):
		t.Error("another council must not save this council's image")
	case Owned(ref, id, "councils"):
		t.Error("the folder must match")
	case Owned(ref, uuid.Nil, "facilities"):
		t.Error("a council's file is not the installation's")
	case !Owned("users/a.jpg", uuid.Nil, "users"):
		t.Error("the installation owns profile photos")
	case Owned("users/a.jpg", id, "users"):
		t.Error("profile photos are no council's")
	}
}

func TestLocation(t *testing.T) {
	var s = &service{bucket: "media"}
	if b, k := s.location(council + "/facilities/a.jpg"); b != "council-"+council || k != "facilities/a.jpg" {
		t.Errorf("council file at %s %s", b, k)
	}
	if b, k := s.location("users/a.jpg"); b != "media" || k != "users/a.jpg" {
		t.Errorf("installation file at %s %s", b, k)
	}
}

func TestURLIsPresigned(t *testing.T) {
	shared.PUBLIC_URL = "https://reservas.example.gal"
	shared.STORAGE_ENDPOINT = "storage:3900"
	shared.STORAGE_REGION = "garage"
	shared.STORAGE_BUCKET = "media"
	shared.STORAGE_ACCESS_KEY = "GKtest"
	shared.STORAGE_SECRET_KEY = "secret"
	var ref = council + "/facilities/a.jpg"
	var link, err = url.Parse(URL(ref))
	if err != nil {
		t.Fatal(err)
	}
	if link.Scheme != "https" || link.Host != "reservas.example.gal" || link.Path != "/storage/council-"+council+"/facilities/a.jpg" {
		t.Errorf("link to %s", link)
	}
	var query = link.Query()
	if query.Get("X-Amz-Signature") == "" || query.Get("X-Amz-Expires") != "604800" {
		t.Errorf("not presigned for a week: %s", link.RawQuery)
	}
	if URL(ref) != link.String() {
		t.Error("the same link until it is renewed, so browsers keep the image cached")
	}
	if URL("facilities/a.jpg") != "" {
		t.Error("an invalid reference gets no link")
	}
}
