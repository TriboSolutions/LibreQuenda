package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// An OPTIONS without a session must not run the handler behind AuthMiddleware.
func TestOptionsNeverReachesHandler(t *testing.T) {
	var called bool
	var handler = AuthMiddleware(func(w http.ResponseWriter, r *http.Request) { called = true })
	var rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodOptions, "/me/bookings", nil))
	if called {
		t.Fatal("OPTIONS reached the handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d, want 204", rec.Code)
	}
}

func TestGetWithoutTokenIsUnauthorized(t *testing.T) {
	var called bool
	var handler = AuthMiddleware(func(w http.ResponseWriter, r *http.Request) { called = true })
	var rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/me/bookings", nil))
	if called || rec.Code != http.StatusUnauthorized {
		t.Fatalf("called=%v status=%d, want 401 without the handler", called, rec.Code)
	}
}
