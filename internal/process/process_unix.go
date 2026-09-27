//go:build darwin || linux

package process

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Controller owns the operating-system process group created for one command.
type Controller struct {
	pid int
}

// Configure makes the child the leader of a dedicated process group before it starts.
func Configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// Attach records the process group after the command has started.
func Attach(cmd *exec.Cmd) (*Controller, error) {
	if cmd == nil || cmd.Process == nil {
		return nil, fmt.Errorf("attach process controller: command has not started")
	}
	return AttachPID(cmd.Process.Pid)
}

func AttachPID(pid int) (*Controller, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("attach process controller: invalid pid %d", pid)
	}
	return &Controller{pid: pid}, nil
}

// Terminate stops the whole process tree represented by the process group.
func (c *Controller) Terminate() error {
	if c == nil || c.pid <= 0 {
		return nil
	}
	err := syscall.Kill(-c.pid, syscall.SIGKILL)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}

// TerminateGracefully gives cooperative children a bounded SIGTERM grace period
// before falling back to the existing process-group SIGKILL behavior.
func (c *Controller) TerminateGracefully(grace time.Duration) error {
	if c == nil || c.pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-c.pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return err
	}
	if grace > 0 {
		deadline := time.Now().Add(grace)
		for time.Now().Before(deadline) {
			if !c.running() {
				return nil
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	return c.Terminate()
}

func (c *Controller) running() bool {
	if c == nil || c.pid <= 0 {
		return false
	}
	err := syscall.Kill(-c.pid, 0)
	return err == nil || err == syscall.EPERM
}

// Close releases platform resources. Unix process groups do not own handles.
func (c *Controller) Close() error { return nil }
