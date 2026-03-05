//go:build windows

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

	out, err := exec.Command("netstat", "-ano", "-p", "TCP").Output()
	if err != nil {
		return nil, err
	}

	var ports []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "LISTENING") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// fields[1] = local address, fields[4] = PID
		addr := fields[1]
		if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "[::1]:") {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil || !pidSet[pid] {
			continue
		}
		idx := strings.LastIndex(addr, ":")
		if idx < 0 {
			continue
		}
		port, err := strconv.Atoi(addr[idx+1:])
		if err != nil {
			continue
		}
		ports = append(ports, port)
	}

	return ports, nil
}
