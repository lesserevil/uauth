//go:build linux

package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// findListeningPorts returns localhost-only listening ports owned by processes in the given process group.
func findListeningPorts(pgid int) ([]int, error) {
	pids, err := processTreePIDs(pgid)
	if err != nil {
		return nil, err
	}

	pidSet := make(map[int]bool, len(pids))
	for _, p := range pids {
		pidSet[p] = true
	}

	// Build inode→port map from /proc/net/tcp and tcp6
	inodePorts := make(map[uint64]int)
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if err := parseProcNetTCP(path, inodePorts); err != nil {
			logVerbose("Failed to parse %s: %v", path, err)
		}
	}

	if len(inodePorts) == 0 {
		return nil, nil
	}

	// Find which inodes belong to our process tree
	portSet := make(map[int]bool)
	for pid := range pidSet {
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		entries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			link, err := os.Readlink(filepath.Join(fdDir, e.Name()))
			if err != nil {
				continue
			}
			if strings.HasPrefix(link, "socket:[") {
				inodeStr := link[8 : len(link)-1]
				inode, err := strconv.ParseUint(inodeStr, 10, 64)
				if err != nil {
					continue
				}
				if port, ok := inodePorts[inode]; ok {
					portSet[port] = true
				}
			}
		}
	}

	ports := make([]int, 0, len(portSet))
	for p := range portSet {
		ports = append(ports, p)
	}
	return ports, nil
}

// parseProcNetTCP parses /proc/net/tcp or tcp6 for localhost LISTEN entries.
// State 0A = LISTEN. Adds inode→port mappings.
func parseProcNetTCP(path string, inodePorts map[uint64]int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		// fields[3] is state, 0A = LISTEN
		if fields[3] != "0A" {
			continue
		}
		// fields[1] is local_address (hex_ip:hex_port)
		addr := fields[1]
		parts := strings.SplitN(addr, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if !isLocalhost(parts[0]) {
			continue
		}
		port, err := strconv.ParseInt(parts[1], 16, 32)
		if err != nil {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		inodePorts[inode] = int(port)
	}
	return scanner.Err()
}

// isLocalhost checks if a hex-encoded IP is 127.0.0.1 or ::1.
func isLocalhost(hexIP string) bool {
	switch len(hexIP) {
	case 8: // IPv4
		b, err := hex.DecodeString(hexIP)
		if err != nil || len(b) != 4 {
			return false
		}
		// /proc/net/tcp uses little-endian
		ip := net.IPv4(b[3], b[2], b[1], b[0])
		return ip.IsLoopback()
	case 32: // IPv6
		b, err := hex.DecodeString(hexIP)
		if err != nil || len(b) != 16 {
			return false
		}
		// /proc/net/tcp6 stores in groups of 4 bytes, each group little-endian
		var ip [16]byte
		for i := 0; i < 4; i++ {
			ip[i*4+0] = b[i*4+3]
			ip[i*4+1] = b[i*4+2]
			ip[i*4+2] = b[i*4+1]
			ip[i*4+3] = b[i*4+0]
		}
		return net.IP(ip[:]).IsLoopback()
	}
	return false
}
