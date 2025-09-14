package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
)

// runCmd runs a command optionally silencing stdout/stderr.
// If dir is empty, runs in current directory. When quiet is true, output is discarded.
func runCmd(ctx context.Context, dir string, quiet bool, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if quiet {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("exit %d", ee.ExitCode())
		}
		return err
	}
	return nil
}

// mustRunCmd runs a command or fails the test.
func mustRunCmd(ctx context.Context, t *testing.T, dir string, quiet bool, name string, args ...string) {
	t.Helper()
	if err := runCmd(ctx, dir, quiet, name, args...); err != nil {
		if dir == "" {
			t.Fatalf("%s %v: %v", name, args, err)
		}
		t.Fatalf("[%s] %s %v: %v", dir, name, args, err)
	}
}
