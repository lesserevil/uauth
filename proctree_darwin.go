//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// processTreePIDs returns all PIDs in the process group (pgid).
func processTreePIDs(pgid int) ([]int, error) {
	out, err := exec.Command("ps", "-eo", "pid,pgid").Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		pg, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		if pg == pgid {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}
