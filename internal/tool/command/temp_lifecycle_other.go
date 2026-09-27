//go:build !linux

package command

func managedTempLifecycleSupported() bool { return false }

func probeManagedTempPath(string, int) managedTempState { return managedTempUnknown }
