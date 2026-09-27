//go:build linux

package command

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/uvwt/agentdock/internal/config"
	processcontrol "github.com/uvwt/agentdock/internal/process"
)

func testManagedTempManager(t *testing.T) *managedTempManager {
	t.Helper()
	home := t.TempDir()
	return newManagedTempManager(func() config.Config {
		return config.Config{AgentDockHome: home}
	})
}

func TestManagedTempAcquireReleaseUsesDedicatedNamespaceAndPreservesLegacy(t *testing.T) {
	manager := testManagedTempManager(t)
	home := manager.config().AgentDockHome
	legacy := filepath.Join(home, "tmp", "legacy.keep")
	if err := os.WriteFile(legacy, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	lease, err := manager.acquire("test")
	if err != nil {
		t.Fatal(err)
	}
	if lease == nil {
		t.Fatal("expected managed temp lease")
	}
	wantRoot := filepath.Join(home, "tmp", managedTempDirectoryName)
	if filepath.Dir(lease.Path()) != wantRoot {
		t.Fatalf("managed temp path = %q, want direct child of %q", lease.Path(), wantRoot)
	}
	if err := os.WriteFile(filepath.Join(lease.Path(), "payload"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	lease.Release()
	if _, err := os.Stat(lease.Path()); !os.IsNotExist(err) {
		t.Fatalf("managed temp resource still exists after release: %v", err)
	}
	if data, err := os.ReadFile(legacy); err != nil || string(data) != "keep" {
		t.Fatalf("legacy temp sibling changed: data=%q err=%v", string(data), err)
	}
}

func TestManagedTempReleaseProtectsActiveProcessGroup(t *testing.T) {
	manager := testManagedTempManager(t)
	lease, err := manager.acquire("active")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", "sleep 30")
	cmd.Env = append(os.Environ(), "TMPDIR="+lease.Path())
	processcontrol.Configure(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	controller, err := processcontrol.Attach(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	if err := lease.BindProcessGroup(cmd.Process.Pid); err != nil {
		_ = controller.Terminate()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = controller.Terminate()
		_ = cmd.Wait()
		_ = controller.Close()
	})

	lease.Release()
	if _, err := os.Stat(lease.Path()); err != nil {
		t.Fatalf("active managed temp resource was removed: %v", err)
	}

	if err := controller.Terminate(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	_ = controller.Close()
	lease.ReleaseEventually(2 * time.Second)
	if _, err := os.Stat(lease.Path()); !os.IsNotExist(err) {
		t.Fatalf("inactive managed temp resource still exists: %v", err)
	}
}

func TestManagedTempReconcileKeepsUnknownAndRemovesStaleInactive(t *testing.T) {
	manager := testManagedTempManager(t)
	base := time.Now().UTC()
	manager.now = func() time.Time { return base }
	unknownLease, err := manager.acquire("unknown")
	if err != nil {
		t.Fatal(err)
	}
	inactiveLease, err := manager.acquire("inactive")
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return base.Add(25 * time.Hour) }
	manager.staleAfter = 24 * time.Hour
	manager.probe = func(path string, _ int) managedTempState {
		if path == unknownLease.Path() {
			return managedTempUnknown
		}
		return managedTempInactive
	}
	report, err := manager.reconcile(true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Removed != 1 || report.Unknown != 1 {
		t.Fatalf("reconcile report = %#v, want one removed and one unknown", report)
	}
	if _, err := os.Stat(unknownLease.Path()); err != nil {
		t.Fatalf("unknown resource was removed: %v", err)
	}
	if _, err := os.Stat(inactiveLease.Path()); !os.IsNotExist(err) {
		t.Fatalf("stale inactive resource still exists: %v", err)
	}
}

func TestManagedTempReconcileFailsClosedOnSymlinkChild(t *testing.T) {
	manager := testManagedTempManager(t)
	root, err := manager.ensureRoot()
	if err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	sentinel := filepath.Join(external, "sentinel")
	if err := os.WriteFile(sentinel, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, managedTempPrefix+"symlink")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	manager.staleAfter = 0
	report, err := manager.reconcile(true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Skipped == 0 {
		t.Fatalf("expected symlink child to be skipped: %#v", report)
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "safe" {
		t.Fatalf("symlink target changed: data=%q err=%v", string(data), err)
	}
}

func TestManagedTempStartupReconcilesStaleInactive(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "tmp", managedTempDirectoryName)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	id := "startup-stale"
	path := filepath.Join(root, managedTempPrefix+id)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := managedTempMetadata{
		SchemaVersion: managedTempSchemaVersion,
		ID:            id,
		Kind:          "startup",
		CreatedAt:     time.Now().UTC().Add(-managedTempStaleAfter - time.Hour),
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, managedTempMetadataName), append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	_ = newManagedTempManagerWithProbe(func() config.Config {
		return config.Config{AgentDockHome: home}
	}, func(string, int) managedTempState {
		return managedTempInactive
	})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("startup reconciliation left stale inactive resource: %v", err)
	}
}

func TestManagedTempValidationRejectsTraversalOutsideNamespace(t *testing.T) {
	manager := testManagedTempManager(t)
	root, err := manager.ensureRoot()
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	traversal := filepath.Join(root, "..", "outside")
	if _, err := validateManagedTempChild(root, traversal, ""); err == nil {
		t.Fatal("expected traversal outside managed namespace to be rejected")
	}
	if err := removeManagedTempChild(root, traversal, ""); err == nil {
		t.Fatal("expected traversal removal to fail closed")
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "safe" {
		t.Fatalf("outside traversal target changed: data=%q err=%v", string(data), err)
	}
}

func TestManagedTempConcurrentAcquireProducesUniqueResources(t *testing.T) {
	manager := testManagedTempManager(t)
	const count = 16
	var wg sync.WaitGroup
	paths := make(chan string, count)
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := manager.acquire("concurrent")
			if err != nil {
				errs <- err
				return
			}
			paths <- lease.Path()
			lease.Release()
		}()
	}
	wg.Wait()
	close(paths)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for path := range paths {
		if seen[path] {
			t.Fatalf("duplicate managed temp path %q", path)
		}
		seen[path] = true
	}
	if len(seen) != count {
		t.Fatalf("unique managed temp paths = %d, want %d", len(seen), count)
	}
}
