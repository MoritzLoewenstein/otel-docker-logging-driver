package integration

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEndToEnd_GRPC(t *testing.T) { runEndToEnd(t, "grpc") }
func TestEndToEnd_HTTP(t *testing.T) { runEndToEnd(t, "http") }

func runEndToEnd(t *testing.T, proto string) {
	if os.Getenv("DOCKER_INTEGRATION") != "1" {
		t.Skip("set DOCKER_INTEGRATION=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	wd, _ := os.Getwd()
	stackDir := filepath.Clean(filepath.Join(wd))

	switch proto {
	case "grpc":
		_ = runCmd(ctx, "", true, "docker", "plugin", "disable", "-f", "moritzloewenstein/otel-docker-logging-driver:dev")
		mustRunCmd(ctx, t, "", true, "docker", "plugin", "set", "moritzloewenstein/otel-docker-logging-driver:dev",
			"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317",
			"OTEL_EXPORTER_OTLP_LOGS_INSECURE=true",
		)
		mustRunCmd(ctx, t, "", true, "docker", "plugin", "enable", "moritzloewenstein/otel-docker-logging-driver:dev")
	case "http":
		_ = runCmd(ctx, "", true, "docker", "plugin", "disable", "-f", "moritzloewenstein/otel-docker-logging-driver:dev")
		mustRunCmd(ctx, t, "", true, "docker", "plugin", "set", "moritzloewenstein/otel-docker-logging-driver:dev",
			"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4318",
			"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=http/protobuf",
		)
		mustRunCmd(ctx, t, "", true, "docker", "plugin", "enable", "moritzloewenstein/otel-docker-logging-driver:dev")
	default:
		t.Fatalf("unknown protocol: %s", proto)
	}

	// The test lives in test/integration; ensure we're there for compose
	// Remove old data file to avoid stale assertions
	dataFile := filepath.Join(stackDir, "data", "otel-logs.json")
	_ = os.Remove(dataFile)

	// Bring up collector + demo in background
	mustRunCmd(ctx, t, stackDir, true, "docker", "compose", "up", "-d")
	defer func() { _ = runCmd(context.Background(), stackDir, true, "docker", "compose", "down", "-v") }()

	// Wait for output file to be populated and contain expected attributes
	deadline := time.Now().Add(45 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s to be populated", dataFile)
		}
		if st, err := os.Stat(dataFile); err == nil && st.Size() > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	f, err := os.Open(dataFile)
	if err != nil {
		t.Fatalf("open data: %v", err)
	}
	defer func() { _ = f.Close() }()
	foundBody := false
	foundLabel := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.Contains(line, "\"hello ") {
			foundBody = true
		}
		if strings.Contains(line, "\"docker.label.test.label\"") && strings.Contains(line, "\"demo\"") {
			foundLabel = true
		}
		if foundBody && foundLabel {
			break
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundBody {
		t.Fatalf("expected at least one hello log record in %s", dataFile)
	}
	if !foundLabel {
		t.Fatalf("expected label attribute in log record in %s", dataFile)
	}
}
