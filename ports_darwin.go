//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// findListeningPorts returns localhost-only listening ports owned by processes in the given process group.
func findListeningPorts(pgid int) ([]int, error) {
	pids, err := processTreePIDs(pgid)
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, nil
	}

	pidSet := make(map[int]bool, len(pids))
	for _, p := range pids {
		pidSet[p] = true
	}

	// Use lsof to find TCP listeners. -n = no DNS, -P = no port names.
	out, err := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P", "-F", "pn").Output()
	if err != nil {
		return nil, err
	}

	var ports []int
	var currentPID int

	for _, line := range strings.Split(string(out), "\n") {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, err := strconv.Atoi(line[1:])
			if err != nil {
				currentPID = 0
				continue
			}
			currentPID = pid
		case 'n':
			if !pidSet[currentPID] {
				continue
			}
			// Format: n127.0.0.1:PORT or n[::1]:PORT or n*:PORT
			addr := line[1:]
			if !isLocalhostAddr(addr) {
				continue
			}
			port := extractPort(addr)
			if port > 0 {
				ports = append(ports, port)
			}
		}
	}

	return ports, nil
}

func isLocalhostAddr(addr string) bool {
	return strings.HasPrefix(addr, "127.0.0.1:") ||
		strings.HasPrefix(addr, "[::1]:") ||
		strings.HasPrefix(addr, "localhost:")
}

func extractPort(addr string) int {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0
	}
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil {
		return 0
	}
	return port
}
