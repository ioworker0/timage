package cmd

import (
	"testing"

	"github.com/ioworker0/timage/pkg/registry"
)

func TestFormatPullManifestErrorLatestTag(t *testing.T) {
	err := &registry.ErrorResponse{
		StatusCode: 404,
		Errors: []registry.ErrorDetail{
			{
				Code:    "MANIFEST_UNKNOWN",
				Message: "manifest unknown",
				Detail:  "unknown tag=latest",
			},
		},
	}

	got := formatPullManifestError(err, "bitnami/zookeeper", "latest", "docker.io")
	want := `tag "latest" was not found for bitnami/zookeeper on docker.io. This repository does not publish a latest tag; use an explicit version tag.`

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatPullManifestErrorFallback(t *testing.T) {
	err := formatPullManifestError(assertiveError("boom"), "busybox", "latest", "docker.io")
	want := "Failed to get manifest: boom"

	if err != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

type assertiveError string

func (e assertiveError) Error() string {
	return string(e)
}
