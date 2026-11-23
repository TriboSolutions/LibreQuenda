// Package identity answers Zitadel's call when someone signs in with a public
// identity provider (Cl@ve, Chave365): their name, email and NIF from its attributes.
package identity

import (
	"io"
	"net/http"
	"time"

	"librequenda/middlewares"
	"librequenda/shared"
	"librequenda/zitadel"

	"github.com/angelbarreiros/Penguin/router"
)

func RegisterRoutes(r *router.Router) {
	r.NewRoute(router.Route{
		Path:    "/zitadel/idp-intent",
		Method:  http.MethodPost,
		Handler: middlewares.VersionMiddleware(idpIntent),
	})
}

// An empty 200 leaves Zitadel's response unchanged.
func idpIntent(w http.ResponseWriter, r *http.Request) {
	var mapping, err = zitadel.LoadIDPMapping()
	if err != nil {
		shared.Logger.Error("identity provider mapping: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if mapping == nil {
		http.NotFound(w, r)
		return
	}
	var body []byte
	if body, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20)); err != nil {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err = zitadel.VerifySignature(r.Header.Get("ZITADEL-Signature"), body, mapping.SigningKey, time.Now()); err != nil {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}
	var mapped []byte
	if mapped, err = mapping.IdentityIntent(body); err != nil {
		shared.Logger.Error("identity provider mapping: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if mapped != nil {
		_, _ = w.Write(mapped)
	}
}
