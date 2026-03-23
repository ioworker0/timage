package registry

import (
	"errors"
	"testing"
)

func TestNewRegistryErrorParsesStructuredBody(t *testing.T) {
	err := newRegistryError(404, []byte(`{"errors":[{"code":"MANIFEST_UNKNOWN","message":"manifest unknown","detail":"unknown tag=latest"}]}`))

	var registryErr *ErrorResponse
	if !errors.As(err, &registryErr) {
		t.Fatalf("expected ErrorResponse, got %T", err)
	}

	if !registryErr.HasCode("MANIFEST_UNKNOWN") {
		t.Fatalf("expected MANIFEST_UNKNOWN, got %+v", registryErr.Errors)
	}

	if got := registryErr.UnknownTag(); got != "latest" {
		t.Fatalf("expected latest tag, got %q", got)
	}

	want := "registry returned MANIFEST_UNKNOWN: manifest unknown (unknown tag=latest)"
	if got := registryErr.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNewRegistryErrorFallsBackToStatus(t *testing.T) {
	err := newRegistryError(404, []byte("not found"))
	want := "unexpected status code 404: not found"

	if got := err.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
