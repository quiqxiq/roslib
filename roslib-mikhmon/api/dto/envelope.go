// Package dto berisi tipe request/response HTTP untuk roslib-mikhmon.
// DTO sengaja dipisah dari domain.* supaya:
//   - Field naming pakai snake_case (JSON convention).
//   - Validator tag binding:"required,..." hanya di DTO.
//   - Bisa hide field sensitif (mis. password tidak di-expose di GET).
package dto

// Envelope adalah response wrapper yang konsisten untuk semua endpoint
// REST. Field Data dan Error mutual exclusive — success satu, error satu.
type Envelope struct {
	Data  any            `json:"data,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
	Error *ErrorPayload  `json:"error,omitempty"`
}

// ErrorPayload struktur konsisten untuk error response. Code stabil
// (NOT_FOUND, INVALID_ARGUMENT, AMBIGUOUS, TIMEOUT, INTERNAL, VALIDATION,
// CANCELED). Message human-readable. Details optional payload tambahan.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Details any    `json:"details,omitempty"`
}

// OK membungkus single-item response.
func OK(data any) Envelope { return Envelope{Data: data} }

// List membungkus list response dengan meta opsional (mis. {count: N}).
func List(items any, meta map[string]any) Envelope {
	return Envelope{Data: items, Meta: meta}
}

// Err membungkus error response. Path biasanya c.Request.URL.Path.
func Err(code, msg, path string) Envelope {
	return Envelope{Error: &ErrorPayload{Code: code, Message: msg, Path: path}}
}

// ErrDetails varian dengan field details (validator error, dst).
func ErrDetails(code, msg, path string, details any) Envelope {
	return Envelope{Error: &ErrorPayload{Code: code, Message: msg, Path: path, Details: details}}
}
