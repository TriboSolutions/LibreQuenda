package shared

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Args    []any  `json:"-"` // values for the message's {{0}}, {{1}}…
}
