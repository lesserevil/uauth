//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// processTreePIDs returns all PIDs descended from the given root PID.
// Windows has no process groups, so we walk the parent-child tree.
func processTreePIDs(rootPID int) ([]int, error) {
	out, err := exec.Command("wmic", "process", "get", "ProcessId,ParentProcessId", "/format:csv").Output()
	if err != nil {
		return nil, err
	}

	// Build parent→children map
	children := make(map[int][]int)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		// CSV format: Node,ParentProcessId,ProcessId
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}
		ppid, err1 := strconv.Atoi(strings.TrimSpace(fields[1]))
		pid, err2 := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err1 != nil || err2 != nil {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}

	// BFS from root
	var pids []int
	queue := []int{rootPID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		pids = append(pids, current)
		queue = append(queue, children[current]...)
	}

	return pids, nil
}
