package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// childProcess wraps an os/exec.Cmd with process-group tracking.
type childProcess struct {
	cmd *exec.Cmd
}

// startChild launches the command in its own process group.
// If stdin is a terminal, the child becomes the foreground process group
// so it has full terminal control (raw mode, signals, window resize, etc.).
func startChild(args []string) (*childProcess, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "BROWSER=false")

	attr := &syscall.SysProcAttr{Setpgid: true}
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		attr.Foreground = true
		attr.Ctty = int(os.Stdin.Fd())
	}
	cmd.SysProcAttr = attr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &childProcess{cmd: cmd}, nil
}

// PGID returns the process group ID of the child.
func (c *childProcess) PGID() int {
	if c.cmd.Process == nil {
		return 0
	}
	// When Setpgid is true, the pgid equals the child's pid.
	return c.cmd.Process.Pid
}

// Wait waits for the child to exit and returns its exit code.
func (c *childProcess) Wait() int {
	err := c.cmd.Wait()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// Signal sends a signal to the child's process group.
func (c *childProcess) Signal(sig os.Signal) {
	if c.cmd.Process != nil {
		// Send to the whole process group
		syscall.Kill(-c.cmd.Process.Pid, sig.(syscall.Signal))
	}
}

// execPassthrough replaces the current process with the given command (no SSH tunneling needed).
func execPassthrough(args []string) {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		log_fatal("command not found: %s", args[0])
	}
	err = syscall.Exec(binary, args, os.Environ())
	if err != nil {
		log_fatal("exec failed: %v", err)
	}
}

func log_fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
