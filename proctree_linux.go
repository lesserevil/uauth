//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
)

// processTreePIDs returns all PIDs in the process group (pgid).
func processTreePIDs(pgid int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var pids []int
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
		if len(fields) < 3 {
			continue
		}
		// fields[2] is pgrp (0-indexed from after state)
		pgrp, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		if pgrp == pgid {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}
