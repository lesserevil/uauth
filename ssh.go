package main

import (
	"os"
	"strings"
)

// detectSSHClient returns the client IP from SSH_CONNECTION or SSH_CLIENT.
// Returns empty string if not in an SSH session.
func detectSSHClient() string {
	// SSH_CONNECTION format: client_ip client_port server_ip server_port
	if conn := os.Getenv("SSH_CONNECTION"); conn != "" {
		fields := strings.Fields(conn)
		if len(fields) >= 1 {
			return fields[0]
		}
	}

	// SSH_CLIENT format: client_ip client_port server_port
	if client := os.Getenv("SSH_CLIENT"); client != "" {
		fields := strings.Fields(client)
		if len(fields) >= 1 {
			return fields[0]
		}
	}

	return ""
}
