package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("DOCKER_INTEGRATION") != "1" {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	wd, _ := os.Getwd()
	stackDir := filepath.Clean(filepath.Join(wd))
	repoRoot := filepath.Dir(filepath.Dir(stackDir))
	if err := runCmd(ctx, repoRoot, true, "make", "plugin-up"); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}
