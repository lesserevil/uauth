//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// processTreePIDs returns all PIDs descended from rootPID (inclusive),
// walking the full parent-child tree so processes that create new process
// groups (e.g. MCP servers) are still included.
func processTreePIDs(rootPID int) ([]int, error) {
	out, err := exec.Command("ps", "-eo", "pid,ppid").Output()
	if err != nil {
		return nil, err
	}

	// Build ppid→children map
	children := make(map[int][]int)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
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
