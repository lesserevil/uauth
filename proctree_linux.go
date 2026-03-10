//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
)

// processTreePIDs returns all PIDs descended from rootPID (inclusive),
// walking the full parent-child tree so processes that create new process
// groups (e.g. MCP servers) are still included.
func processTreePIDs(rootPID int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	// Build ppid→children map
	children := make(map[int][]int)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		// Format: pid (comm) state ppid pgrp ...
		// Find closing paren to skip comm (which may contain spaces/parens)
		s := string(data)
		idx := strings.LastIndex(s, ")")
		if idx < 0 || idx+2 >= len(s) {
			continue
		}
		fields := strings.Fields(s[idx+2:])
		if len(fields) < 1 {
			continue
		}
		// fields[0] is state, fields[1] is ppid
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
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
