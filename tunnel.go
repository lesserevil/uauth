package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type tunnelManager struct {
	user     string
	clientIP string
	mu       sync.Mutex
	tunnels  map[int]*exec.Cmd
}

func newTunnelManager(user, clientIP string) *tunnelManager {
	return &tunnelManager{
		user:     user,
		clientIP: clientIP,
		tunnels:  make(map[int]*exec.Cmd),
	}
}

// establish starts a reverse SSH tunnel for the given port.
func (tm *tunnelManager) establish(port int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tunnels[port]; exists {
		return
	}

	remote := fmt.Sprintf("%d:localhost:%d", port, port)
	target := fmt.Sprintf("%s@%s", tm.user, tm.clientIP)

	cmd := exec.Command("ssh",
		"-N",                  // no remote command
		"-v",                  // debug output so we can detect when forward is ready
		"-o", "BatchMode=yes", // never prompt for password/passphrase
		"-o", "ExitOnForwardFailure=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-R", remote,
		target,
	)

	// Suppress stdout; pipe stderr so we can detect tunnel readiness.
	devnull, _ := os.Open(os.DevNull)
	if devnull != nil {
		cmd.Stdout = devnull
		defer devnull.Close()
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		// Fall back to suppressing stderr entirely.
		cmd.Stderr = devnull
		stderrPipe = nil
	}

	if err := cmd.Start(); err != nil {
		logVerbose("Failed to start tunnel for port %d: %v", port, err)
		return
	}

	tm.tunnels[port] = cmd
	logVerbose("Tunnel connecting: localhost:%d -> %s:%d", port, tm.clientIP, port)

	// Monitor tunnel process in background; log when the forward is confirmed ready.
	go func() {
		if stderrPipe != nil {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "remote forward success") {
					logVerbose("Tunnel ready: localhost:%d -> %s:%d", port, tm.clientIP, port)
				}
			}
		}
		err := cmd.Wait()
		tm.mu.Lock()
		delete(tm.tunnels, port)
		tm.mu.Unlock()
		if err != nil {
			logVerbose("Tunnel for port %d exited: %v", port, err)
		}
	}()
}

// teardown stops the tunnel for the given port.
func (tm *tunnelManager) teardown(port int) {
	tm.mu.Lock()
	cmd, exists := tm.tunnels[port]
	if exists {
		delete(tm.tunnels, port)
	}
	tm.mu.Unlock()

	if exists && cmd.Process != nil {
		cmd.Process.Kill()
		logVerbose("Tunnel torn down for port %d", port)
	}
}

// teardownAll stops all active tunnels.
func (tm *tunnelManager) teardownAll() {
	tm.mu.Lock()
	tunnels := make(map[int]*exec.Cmd, len(tm.tunnels))
	for k, v := range tm.tunnels {
		tunnels[k] = v
	}
	tm.tunnels = make(map[int]*exec.Cmd)
	tm.mu.Unlock()

	for port, cmd := range tunnels {
		if cmd.Process != nil {
			cmd.Process.Kill()
			logVerbose("Tunnel torn down for port %d", port)
		}
	}
}
