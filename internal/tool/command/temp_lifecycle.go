package command

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	managedTempDirectoryName       = "managed"
	managedTempPrefix              = "run-"
	managedTempMetadataName        = ".agentdock-managed-temp.json"
	managedTempSchemaVersion       = 1
	managedTempStaleAfter          = 24 * time.Hour
	managedTempReconcileEvery      = 15 * time.Minute
	managedTempTerminalReleaseWait = 3 * time.Second
	managedTempReleaseRetryEvery   = 25 * time.Millisecond
)

type managedTempState string

const (
	managedTempActive   managedTempState = "active"
	managedTempInactive managedTempState = "inactive"
	managedTempUnknown  managedTempState = "unknown"
)

type managedTempMetadata struct {
	SchemaVersion int       `json:"schema_version"`
	ID            string    `json:"id"`
	Kind          string    `json:"kind"`
	CreatedAt     time.Time `json:"created_at"`
}

type managedTempManager struct {
	config            ConfigProvider
	mu                sync.Mutex
	now               func() time.Time
	probe             func(string) managedTempState
	staleAfter        time.Duration
	reconcileInterval time.Duration
	lastReconcile     time.Time
}

type managedTempLease struct {
	manager  *managedTempManager
	path     string
	id       string
	mu       sync.Mutex
	released bool
}

type managedTempReconcileReport struct {
	Removed int
	Active  int
	Unknown int
	Skipped int
}

func newManagedTempManager(configProvider ConfigProvider) *managedTempManager {
	return newManagedTempManagerWithProbe(configProvider, probeManagedTempPath)
}

func newManagedTempManagerWithProbe(configProvider ConfigProvider, probe func(string) managedTempState) *managedTempManager {
	manager := &managedTempManager{
		config:            configProvider,
		now:               time.Now,
		probe:             probe,
		staleAfter:        managedTempStaleAfter,
		reconcileInterval: managedTempReconcileEvery,
	}
	if managedTempLifecycleSupported() {
		_, _ = manager.reconcile(true)
	}
	return manager
}

func (m *managedTempManager) acquire(kind string) (*managedTempLease, error) {
	if m == nil || !managedTempLifecycleSupported() {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	root, err := m.ensureRootLocked()
	if err != nil {
		return nil, err
	}
	if m.lastReconcile.IsZero() || m.now().Sub(m.lastReconcile) >= m.reconcileInterval {
		_, _ = m.reconcileLocked(root, false)
	}
	id, err := newManagedTempID()
	if err != nil {
		return nil, fmt.Errorf("generate managed temp id: %w", err)
	}
	path := filepath.Join(root, managedTempPrefix+id)
	if err := os.Mkdir(path, 0o700); err != nil {
		return nil, fmt.Errorf("create managed temp resource: %w", err)
	}
	metadata := managedTempMetadata{
		SchemaVersion: managedTempSchemaVersion,
		ID:            id,
		Kind:          strings.TrimSpace(kind),
		CreatedAt:     m.now().UTC(),
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("encode managed temp metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(path, managedTempMetadataName), append(raw, '\n'), 0o600); err != nil {
		_ = os.RemoveAll(path)
		return nil, fmt.Errorf("write managed temp metadata: %w", err)
	}
	return &managedTempLease{manager: m, path: path, id: id}, nil
}

func (l *managedTempLease) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *managedTempLease) Release() {
	_, _ = l.releaseOnce()
}

func (l *managedTempLease) ReleaseEventually(maxWait time.Duration) {
	if l == nil || l.manager == nil {
		return
	}
	deadline := time.Now().Add(maxWait)
	for {
		state, err := l.releaseOnce()
		if err != nil || state == managedTempInactive || state == managedTempUnknown {
			return
		}
		if state != managedTempActive || maxWait <= 0 {
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return
		}
		delay := managedTempReleaseRetryEvery
		if remaining < delay {
			delay = remaining
		}
		time.Sleep(delay)
	}
}

func (l *managedTempLease) releaseOnce() (managedTempState, error) {
	if l == nil || l.manager == nil {
		return managedTempUnknown, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return managedTempInactive, nil
	}
	state, err := l.manager.release(l.path, l.id)
	if err == nil && state == managedTempInactive {
		l.released = true
	}
	return state, err
}

func (m *managedTempManager) release(path, id string) (managedTempState, error) {
	if m == nil || !managedTempLifecycleSupported() {
		return managedTempUnknown, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.ensureRootLocked()
	if err != nil {
		return managedTempUnknown, err
	}
	metadata, err := validateManagedTempChild(root, path, id)
	if err != nil {
		return managedTempUnknown, err
	}
	state := m.probe(path)
	switch state {
	case managedTempInactive:
		if err := removeManagedTempChild(root, path, metadata.ID); err != nil {
			return managedTempUnknown, err
		}
	case managedTempActive, managedTempUnknown:
	default:
		return managedTempUnknown, fmt.Errorf("invalid managed temp state %q", state)
	}
	return state, nil
}

func (m *managedTempManager) reconcile(force bool) (managedTempReconcileReport, error) {
	if m == nil || !managedTempLifecycleSupported() {
		return managedTempReconcileReport{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.ensureRootLocked()
	if err != nil {
		return managedTempReconcileReport{}, err
	}
	return m.reconcileLocked(root, force)
}

func (m *managedTempManager) reconcileLocked(root string, force bool) (managedTempReconcileReport, error) {
	now := m.now()
	if !force && !m.lastReconcile.IsZero() && now.Sub(m.lastReconcile) < m.reconcileInterval {
		return managedTempReconcileReport{}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return managedTempReconcileReport{}, fmt.Errorf("read managed temp root: %w", err)
	}
	report := managedTempReconcileReport{}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), managedTempPrefix) {
			report.Skipped++
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			report.Skipped++
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			report.Skipped++
			continue
		}
		metadata, err := validateManagedTempChild(root, path, "")
		if err != nil {
			report.Skipped++
			continue
		}
		if now.Sub(metadata.CreatedAt) < m.staleAfter {
			report.Skipped++
			continue
		}
		switch state := m.probe(path); state {
		case managedTempActive:
			report.Active++
		case managedTempUnknown:
			report.Unknown++
		case managedTempInactive:
			if err := removeManagedTempChild(root, path, metadata.ID); err != nil {
				report.Unknown++
				continue
			}
			report.Removed++
		default:
			report.Unknown++
		}
	}
	m.lastReconcile = now
	return report, nil
}

func (m *managedTempManager) ensureRoot() (string, error) {
	if m == nil {
		return "", fmt.Errorf("managed temp manager is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureRootLocked()
}

func (m *managedTempManager) ensureRootLocked() (string, error) {
	if m.config == nil {
		return "", fmt.Errorf("managed temp config provider is nil")
	}
	home := filepath.Clean(strings.TrimSpace(m.config().AgentDockHome))
	if home == "" || home == "." {
		return "", fmt.Errorf("AgentDock home is not configured")
	}
	absoluteHome, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("resolve AgentDock home: %w", err)
	}
	canonicalHome, err := filepath.EvalSymlinks(absoluteHome)
	if err != nil {
		return "", fmt.Errorf("resolve AgentDock home symlinks: %w", err)
	}
	tempRoot := filepath.Join(canonicalHome, "tmp")
	if err := ensureDirectoryNoSymlink(tempRoot); err != nil {
		return "", fmt.Errorf("prepare AgentDock temp root: %w", err)
	}
	root := filepath.Join(tempRoot, managedTempDirectoryName)
	if err := ensureDirectoryNoSymlink(root); err != nil {
		return "", fmt.Errorf("prepare managed temp root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve managed temp root: %w", err)
	}
	if canonicalRoot != filepath.Clean(root) {
		return "", fmt.Errorf("managed temp root resolves through a symlink")
	}
	return canonicalRoot, nil
}

func ensureDirectoryNoSymlink(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(path, 0o700); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func validateManagedTempChild(root, path, expectedID string) (managedTempMetadata, error) {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.Dir(rel) != "." {
		return managedTempMetadata{}, fmt.Errorf("managed temp path is outside the direct managed namespace")
	}
	if !strings.HasPrefix(rel, managedTempPrefix) {
		return managedTempMetadata{}, fmt.Errorf("managed temp child has an invalid name")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return managedTempMetadata{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return managedTempMetadata{}, fmt.Errorf("managed temp child is not a direct directory")
	}
	metadataPath := filepath.Join(path, managedTempMetadataName)
	metadataInfo, err := os.Lstat(metadataPath)
	if err != nil {
		return managedTempMetadata{}, err
	}
	if metadataInfo.Mode()&os.ModeSymlink != 0 || !metadataInfo.Mode().IsRegular() {
		return managedTempMetadata{}, fmt.Errorf("managed temp metadata is not a regular file")
	}
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return managedTempMetadata{}, err
	}
	var metadata managedTempMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return managedTempMetadata{}, err
	}
	idFromName := strings.TrimPrefix(rel, managedTempPrefix)
	if metadata.SchemaVersion != managedTempSchemaVersion || metadata.ID == "" || metadata.ID != idFromName {
		return managedTempMetadata{}, fmt.Errorf("managed temp metadata identity mismatch")
	}
	if expectedID != "" && metadata.ID != expectedID {
		return managedTempMetadata{}, fmt.Errorf("managed temp lease identity mismatch")
	}
	if metadata.CreatedAt.IsZero() {
		return managedTempMetadata{}, fmt.Errorf("managed temp metadata has no creation time")
	}
	return metadata, nil
}

func removeManagedTempChild(root, path, expectedID string) error {
	if _, err := validateManagedTempChild(root, path, expectedID); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func newManagedTempID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func withManagedTempEnvironment(env []string, path string) []string {
	keys := map[string]bool{"TMPDIR": true, "TEMP": true, "TMP": true}
	result := make([]string, 0, len(env)+3)
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && keys[strings.ToUpper(key)] {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "TMPDIR="+path, "TEMP="+path, "TMP="+path)
}

func setManagedTempEnvironment(env map[string]string, path string) {
	env["TMPDIR"] = path
	env["TEMP"] = path
	env["TMP"] = path
}
