package shared

import (
	"io"
	"net/http"
)

// MaxJSONBody: the largest JSON body any endpoint takes (a facility with every
// text at its limit is about 14 KB). Media uploads set their own limit.
const MaxJSONBody = 64 * 1024

// JSONBody: the request's body, cut off at MaxJSONBody, so a huge one is refused
// while it is read instead of being decoded whole (decoding it then fails: a 400).
func JSONBody(w http.ResponseWriter, r *http.Request) io.Reader {
	return http.MaxBytesReader(w, r.Body, MaxJSONBody)
}
