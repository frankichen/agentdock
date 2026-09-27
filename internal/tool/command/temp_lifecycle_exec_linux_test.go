//go:build linux

package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/uvwt/agentdock/internal/config"
	"github.com/uvwt/agentdock/internal/envstore"
	"github.com/uvwt/agentdock/internal/tool/command/session"
	"github.com/uvwt/agentdock/internal/workspace"
)

func newManagedTempExecService(t *testing.T) (*Service, string) {
	t.Helper()
	return newManagedTempExecServiceWithContext(t, context.Background())
}

func newManagedTempExecServiceWithContext(t *testing.T, commandCtx context.Context) (*Service, string) {
	t.Helper()
	home := t.TempDir()
	workdir := t.TempDir()
	ws, err := workspace.New(workdir)
	if err != nil {
		t.Fatal(err)
	}
	envs, err := envstore.New(home)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AgentDockHome: home, AgentDockDefaultDir: workdir}
	svc := New(
		func() config.Config { return cfg },
		ws,
		envs,
		session.NewStore(),
		func(string) (string, error) { return "", nil },
		func() (context.Context, error) { return commandCtx, nil },
		nil,
	)
	return svc, filepath.Join(home, "tmp", managedTempDirectoryName)
}

func waitManagedTempEmpty(t *testing.T, root string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		entries, err := os.ReadDir(root)
		if err == nil && len(entries) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("managed temp root not empty: entries=%v err=%v", entries, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestExecManagedTempLifecycleSuccessFailureTimeoutAndKill(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, root := newManagedTempExecService(t)
		result, err := svc.Exec(context.Background(), map[string]any{
			"cmd":            "test -d \"$TMPDIR\" && printf ok",
			"execution_mode": "sync",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result["status"] != "exited" {
			t.Fatalf("status = %v, want exited", result["status"])
		}
		waitManagedTempEmpty(t, root)
	})

	t.Run("failure", func(t *testing.T) {
		svc, root := newManagedTempExecService(t)
		result, err := svc.Exec(context.Background(), map[string]any{
			"cmd":            "exit 7",
			"execution_mode": "sync",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result["exit_code"] == 0 {
			t.Fatalf("exit_code = %v, want non-zero", result["exit_code"])
		}
		waitManagedTempEmpty(t, root)
	})

	t.Run("timeout", func(t *testing.T) {
		svc, root := newManagedTempExecService(t)
		result, err := svc.Exec(context.Background(), map[string]any{
			"cmd":            "sleep 30",
			"execution_mode": "sync",
			"timeout_ms":     50,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result["status"] != "timeout" {
			t.Fatalf("status = %v, want timeout", result["status"])
		}
		waitManagedTempEmpty(t, root)
	})

	t.Run("cancel", func(t *testing.T) {
		commandCtx, cancel := context.WithCancel(context.Background())
		svc, root := newManagedTempExecServiceWithContext(t, commandCtx)
		result, err := svc.Exec(context.Background(), map[string]any{
			"cmd":            "sleep 30",
			"execution_mode": "async",
		})
		if err != nil {
			t.Fatal(err)
		}
		sessionID, _ := result["session_id"].(string)
		sess, ok := svc.sessions.Get(sessionID)
		if !ok {
			t.Fatalf("session %q was not retained", sessionID)
		}
		cancel()
		select {
		case <-sess.Done:
		case <-time.After(3 * time.Second):
			t.Fatal("canceled command did not stop")
		}
		waitManagedTempEmpty(t, root)
	})

	t.Run("kill", func(t *testing.T) {
		svc, root := newManagedTempExecService(t)
		result, err := svc.Exec(context.Background(), map[string]any{
			"cmd":            "sleep 30",
			"execution_mode": "async",
		})
		if err != nil {
			t.Fatal(err)
		}
		sessionID, _ := result["session_id"].(string)
		if sessionID == "" {
			t.Fatalf("session_id = %v", result["session_id"])
		}
		if _, err := svc.killSession(map[string]any{"session_id": sessionID}); err != nil {
			t.Fatal(err)
		}
		waitManagedTempEmpty(t, root)
	})
}
