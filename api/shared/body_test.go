package shared

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONBody(t *testing.T) {
	var small = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Ana"}`))
	var into struct{ Name string }
	if err := json.NewDecoder(JSONBody(httptest.NewRecorder(), small)).Decode(&into); err != nil || into.Name != "Ana" {
		t.Fatalf("small body: %v %+v", err, into)
	}
	var huge = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"`+strings.Repeat("a", MaxJSONBody)+`"}`))
	if err := json.NewDecoder(JSONBody(httptest.NewRecorder(), huge)).Decode(&into); err == nil {
		t.Fatal("a body over the limit was decoded")
	}
}
