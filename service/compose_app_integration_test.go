package service_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appdocker "github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/docker/compose/v2/pkg/api"
)

func TestComposeAppLifecycle(t *testing.T) {
	if !appdocker.IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	projectName := fmt.Sprintf("casaos-ubuntu26-%d", time.Now().UnixNano())
	workingDir := t.TempDir()
	bindDir := filepath.Join(workingDir, "config")
	if err := os.Mkdir(bindDir, 0o700); err != nil {
		t.Fatal(err)
	}

	port := availableTCPPort(t)
	composePath := filepath.Join(workingDir, "docker-compose.yml")
	initialYAML := lifecycleComposeYAML(projectName, bindDir, port, "v1")
	if err := os.WriteFile(composePath, initialYAML, 0o600); err != nil {
		t.Fatal(err)
	}

	composeApp, err := service.LoadComposeAppFromConfigFile(projectName, composePath)
	if err != nil {
		t.Fatal(err)
	}

	composeService, dockerClient, err := service.ApiService()
	if err != nil {
		t.Fatal(err)
	}
	defer dockerClient.Close()
	uninstalled := false
	t.Cleanup(func() {
		if uninstalled {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		_ = composeService.Down(cleanupCtx, projectName, api.DownOptions{
			Images:        "all",
			RemoveOrphans: true,
			Volumes:       true,
		})
	})

	if err := composeApp.PullAndInstall(ctx); err != nil {
		t.Fatal(err)
	}
	assertComposeServiceState(t, ctx, composeApp, "running")
	assertComposeContainerConfig(t, ctx, composeApp, "CASAOS_TEST=v1")

	logs, err := composeApp.Logs(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logs), "casaos-lifecycle-v1") {
		t.Fatalf("initial logs did not contain lifecycle marker: %s", logs)
	}

	if err := composeService.Stop(ctx, projectName, api.StopOptions{}); err != nil {
		t.Fatal(err)
	}
	assertComposeServiceState(t, ctx, composeApp, "exited")

	if err := composeService.Start(ctx, projectName, api.StartOptions{Wait: true}); err != nil {
		t.Fatal(err)
	}
	assertComposeServiceState(t, ctx, composeApp, "running")

	if err := composeService.Restart(ctx, projectName, api.RestartOptions{}); err != nil {
		t.Fatal(err)
	}
	assertComposeServiceState(t, ctx, composeApp, "running")

	updatedYAML := lifecycleComposeYAML(projectName, bindDir, port, "v2")
	if err := composeApp.PullAndApply(ctx, updatedYAML); err != nil {
		t.Fatal(err)
	}
	assertComposeContainerConfig(t, ctx, composeApp, "CASAOS_TEST=v2")

	if err := composeApp.Uninstall(ctx, false); err != nil {
		t.Fatal(err)
	}
	uninstalled = true
	containers, err := composeService.Ps(ctx, projectName, api.PsOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 0 {
		t.Fatalf("compose app still has %d containers after uninstall", len(containers))
	}
}

func lifecycleComposeYAML(projectName, bindDir string, port int, marker string) []byte {
	return []byte(fmt.Sprintf(`name: %s
services:
  app:
    image: busybox:1.36
    command: ["sh", "-c", "echo casaos-lifecycle-%s; httpd -f -p 8080"]
    environment:
      CASAOS_TEST: %s
    ports:
      - "127.0.0.1:%d:8080"
    restart: unless-stopped
    volumes:
      - data:/data
      - "%s:/config"
volumes:
  data:
`, projectName, marker, marker, port, bindDir))
}

func availableTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func assertComposeServiceState(t *testing.T, ctx context.Context, composeApp *service.ComposeApp, want string) {
	t.Helper()
	containers, err := composeApp.Containers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	serviceContainers := containers["app"]
	if len(serviceContainers) != 1 {
		t.Fatalf("service has %d containers, want 1", len(serviceContainers))
	}
	if serviceContainers[0].State != want {
		t.Fatalf("container state = %q, want %q", serviceContainers[0].State, want)
	}
}

func assertComposeContainerConfig(t *testing.T, ctx context.Context, composeApp *service.ComposeApp, wantEnv string) {
	t.Helper()
	containers, err := composeApp.Containers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	serviceContainers := containers["app"]
	if len(serviceContainers) != 1 {
		t.Fatalf("service has %d containers, want 1", len(serviceContainers))
	}
	containerInfo, err := appdocker.Container(ctx, serviceContainers[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(containerInfo.Config.Env, wantEnv) {
		t.Fatalf("container environment %v does not contain %q", containerInfo.Config.Env, wantEnv)
	}
	if got := string(containerInfo.HostConfig.RestartPolicy.Name); got != "unless-stopped" {
		t.Fatalf("restart policy = %q, want %q", got, "unless-stopped")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
