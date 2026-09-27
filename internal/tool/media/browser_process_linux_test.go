//go:build linux

package media

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestRunBrowserCommandCancellationTerminatesProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.Command("sh", "-c", "sleep 30 & wait")
	bound := 0
	err := runBrowserCommand(ctx, cmd, func(processGroup int) error {
		bound = processGroup
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runBrowserCommand() error = %v, want context canceled", err)
	}
	if bound <= 0 {
		t.Fatalf("managed temp process group = %d, want positive", bound)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := syscall.Kill(-bound, 0)
		if err == syscall.ESRCH {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("browser process group %d is still alive: %v", bound, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
