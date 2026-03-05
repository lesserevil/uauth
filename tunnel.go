package main

import (
	"fmt"
	"os"
	"os/exec"
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
		"-o", "BatchMode=yes", // never prompt for password/passphrase
		"-o", "ExitOnForwardFailure=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-R", remote,
		target,
	)

	// Suppress tunnel output so it doesn't pollute the wrapped command's terminal.
	devnull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdout = devnull
		cmd.Stderr = devnull
		defer devnull.Close()
	}

	if err := cmd.Start(); err != nil {
		logVerbose("Failed to establish tunnel for port %d: %v", port, err)
		return
	}

	tm.tunnels[port] = cmd
	logVerbose("Tunnel established: localhost:%d -> %s:%d", port, tm.clientIP, port)

	// Monitor tunnel process in background
	go func() {
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
