package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

// Error is the body of every error response: a stable code for programs and
// a message for people.
type Error struct {
	status  int
	Code    string `json:"code" doc:"Stable error code, such as not_found or tunnel_not_found" example:"tunnel_not_found"`
	Message string `json:"message" doc:"What went wrong, in plain words" example:"tunnel not found"`
}

func (e *Error) Error() string  { return e.Message }
func (e *Error) GetStatus() int { return e.status }

// apiError builds an error response.
func apiError(status int, code, message string) *Error {
	return &Error{status: status, Code: code, Message: message}
}

// codeFor names a status for errors that did not choose their own code.
func codeFor(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusConflict:
		return "conflict"
	case http.StatusNotImplemented:
		return "not_implemented"
	case http.StatusRequestEntityTooLarge:
		return "too_large"
	case http.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	}
	if status >= 500 {
		return "internal"
	}
	return strings.ReplaceAll(strings.ToLower(http.StatusText(status)), " ", "_")
}

var installErrorsOnce sync.Once

// installErrors makes huma's own errors (validation, bad JSON, unknown
// content type) use the same {code, message} body. A request that does not
// match the schema answers 400, as the hand-written handlers did.
func installErrors() {
	installErrorsOnce.Do(func() {
		huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
			if status == http.StatusUnprocessableEntity {
				status = http.StatusBadRequest
			}
			parts := []string{msg}
			for _, err := range errs {
				if err != nil {
					parts = append(parts, err.Error())
				}
			}
			return &Error{status: status, Code: codeFor(status), Message: strings.Join(parts, ": ")}
		}
	})
}

// writeError writes an error outside huma, for the guard and unknown paths.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError(status, code, message))
}
