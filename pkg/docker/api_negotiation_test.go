package docker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func TestAPIVersionNegotiationUsesServerVersion(t *testing.T) {
	originalVersion, versionWasSet := os.LookupEnv("DOCKER_API_VERSION")
	if err := os.Unsetenv("DOCKER_API_VERSION"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if versionWasSet {
			_ = os.Setenv("DOCKER_API_VERSION", originalVersion)
			return
		}
		_ = os.Unsetenv("DOCKER_API_VERSION")
	})

	var mu sync.Mutex
	versionedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_ping":
			w.Header().Set("API-Version", "1.44")
			_, _ = w.Write([]byte("OK"))
		case "/v1.44/containers/json":
			mu.Lock()
			versionedPath = r.URL.Path
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		default:
			http.Error(w, "unexpected Docker API path: "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithHost(server.URL),
		client.WithHTTPClient(server.Client()),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	if _, err := cli.ContainerList(context.Background(), container.ListOptions{}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	got := versionedPath
	mu.Unlock()
	if got != "/v1.44/containers/json" {
		t.Fatalf("negotiated Docker API path = %q, want %q", got, "/v1.44/containers/json")
	}
}
