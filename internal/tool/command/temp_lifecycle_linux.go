//go:build linux

package command

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func managedTempLifecycleSupported() bool { return true }

func probeManagedTempPath(path string, processGroup int) managedTempState {
	path = filepath.Clean(path)
	if processGroup > 0 {
		switch state := managedTempProcessGroupState(processGroup); state {
		case managedTempActive, managedTempUnknown:
			return state
		}
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return managedTempUnknown
	}
	euid := uint32(os.Geteuid())
	unknown := processGroup == 0
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		procRoot := filepath.Join("/proc", entry.Name())
		info, err := os.Stat(procRoot)
		if err != nil {
			if processGroup == 0 && !errors.Is(err, os.ErrNotExist) {
				unknown = true
			}
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			if processGroup == 0 {
				unknown = true
			}
			continue
		}
		if stat.Uid != euid {
			continue
		}
		active, uncertain := sameUserProcessReferencesPath(procRoot, path)
		if active {
			return managedTempActive
		}
		if uncertain && processGroup == 0 {
			unknown = true
		}
	}
	if unknown {
		return managedTempUnknown
	}
	return managedTempInactive
}

func managedTempProcessGroupState(processGroup int) managedTempState {
	if processGroup <= 0 {
		return managedTempUnknown
	}
	err := syscall.Kill(-processGroup, 0)
	switch err {
	case nil, syscall.EPERM:
		return managedTempActive
	case syscall.ESRCH:
		return managedTempInactive
	default:
		return managedTempUnknown
	}
}

func sameUserProcessReferencesPath(procRoot, path string) (bool, bool) {
	uncertain := false

	if data, err := os.ReadFile(filepath.Join(procRoot, "environ")); err == nil {
		for _, entry := range bytes.Split(data, []byte{0}) {
			key, value, ok := bytes.Cut(entry, []byte("="))
			if !ok {
				continue
			}
			switch string(key) {
			case "TMPDIR", "TEMP", "TMP":
				if managedTempPathReference(string(value), path) {
					return true, uncertain
				}
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		uncertain = true
	}

	if data, err := os.ReadFile(filepath.Join(procRoot, "cmdline")); err == nil {
		for _, token := range bytes.Split(data, []byte{0}) {
			if managedTempPathReference(string(token), path) {
				return true, uncertain
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		uncertain = true
	}

	if cwd, err := os.Readlink(filepath.Join(procRoot, "cwd")); err == nil {
		if managedTempPathReference(cwd, path) {
			return true, uncertain
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		uncertain = true
	}

	fdRoot := filepath.Join(procRoot, "fd")
	fds, err := os.ReadDir(fdRoot)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			uncertain = true
		}
		return false, uncertain
	}
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(fdRoot, fd.Name()))
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				uncertain = true
			}
			continue
		}
		target = strings.TrimSuffix(target, " (deleted)")
		if managedTempPathReference(target, path) {
			return true, uncertain
		}
	}
	return false, uncertain
}

func managedTempPathReference(value, root string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	clean := filepath.Clean(value)
	root = filepath.Clean(root)
	if clean == root {
		return true
	}
	return strings.HasPrefix(clean, root+string(os.PathSeparator))
}
