package registry

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorDetail is a single error entry returned by a registry API.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

// DetailString returns a readable representation of the registry error detail.
func (e ErrorDetail) DetailString() string {
	switch detail := e.Detail.(type) {
	case nil:
		return ""
	case string:
		return detail
	default:
		data, err := json.Marshal(detail)
		if err != nil {
			return fmt.Sprintf("%v", detail)
		}
		return string(data)
	}
}

// ErrorResponse represents a structured error returned by a registry API.
type ErrorResponse struct {
	StatusCode int           `json:"-"`
	Errors     []ErrorDetail `json:"errors"`
	Body       string        `json:"-"`
}

func (e *ErrorResponse) Error() string {
	if len(e.Errors) == 0 {
		if e.Body == "" {
			return fmt.Sprintf("registry returned status %d", e.StatusCode)
		}
		return fmt.Sprintf("registry returned status %d: %s", e.StatusCode, e.Body)
	}

	first := e.Errors[0]
	if detail := first.DetailString(); detail != "" {
		if first.Code != "" {
			return fmt.Sprintf("registry returned %s: %s (%s)", first.Code, first.Message, detail)
		}
		return fmt.Sprintf("registry returned status %d: %s (%s)", e.StatusCode, first.Message, detail)
	}

	if first.Code != "" {
		return fmt.Sprintf("registry returned %s: %s", first.Code, first.Message)
	}

	return fmt.Sprintf("registry returned status %d: %s", e.StatusCode, first.Message)
}

// HasCode reports whether the registry returned the given error code.
func (e *ErrorResponse) HasCode(code string) bool {
	for _, err := range e.Errors {
		if strings.EqualFold(err.Code, code) {
			return true
		}
	}

	return false
}

// UnknownTag extracts the missing tag from Docker-style "unknown tag=<tag>" details.
func (e *ErrorResponse) UnknownTag() string {
	for _, err := range e.Errors {
		detail := err.DetailString()
		if strings.HasPrefix(detail, "unknown tag=") {
			return strings.TrimPrefix(detail, "unknown tag=")
		}
	}

	return ""
}

func newRegistryError(statusCode int, body []byte) error {
	trimmed := strings.TrimSpace(string(body))

	var parsed ErrorResponse
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Errors) > 0 {
		parsed.StatusCode = statusCode
		parsed.Body = trimmed
		return &parsed
	}

	if trimmed == "" {
		return fmt.Errorf("unexpected status code %d", statusCode)
	}

	return fmt.Errorf("unexpected status code %d: %s", statusCode, trimmed)
}
