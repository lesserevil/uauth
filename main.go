package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	sshUser      string
	pollInterval time.Duration
	verbose      bool
)

func main() {
	flag.StringVar(&sshUser, "ssh-user", os.Getenv("USER"), "Username for reverse SSH connection")
	pollMs := flag.Int("poll-interval", 500, "Port scan interval in milliseconds")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: uauth [flags] -- <command> [args...]\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	pollInterval = time.Duration(*pollMs) * time.Millisecond

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Detect SSH session
	clientIP := detectSSHClient()
	if clientIP == "" {
		logVerbose("Not in SSH session, executing directly")
		execPassthrough(args)
		// execPassthrough doesn't return
	}

	logVerbose("SSH session detected, client IP: %s", clientIP)
	logVerbose("Reverse SSH user: %s", sshUser)

	// Start child process
	child, err := startChild(args)
	if err != nil {
		log.Fatalf("Failed to start child process: %v", err)
	}

	// Set up signal forwarding
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Track tunnels
	tm := newTunnelManager(sshUser, clientIP)
	knownPorts := make(map[int]bool)

	// Poll loop
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	childDone := make(chan int, 1)
	go func() {
		code := child.Wait()
		childDone <- code
	}()

	for {
		select {
		case code := <-childDone:
			logVerbose("Child process exited with code %d", code)
			tm.teardownAll()
			os.Exit(code)

		case sig := <-sigCh:
			logVerbose("Received signal %v, forwarding to child", sig)
			child.Signal(sig)

		case <-ticker.C:
			ports, err := findListeningPorts(child.PGID())
			if err != nil {
				logVerbose("Port scan error: %v", err)
				continue
			}

			currentPorts := make(map[int]bool)
			for _, p := range ports {
				currentPorts[p] = true
				if !knownPorts[p] {
					knownPorts[p] = true
					log.Printf("New listener detected on port %d, establishing tunnel", p)
					tm.establish(p)
				}
			}

			// Tear down tunnels for ports that disappeared
			for p := range knownPorts {
				if !currentPorts[p] {
					delete(knownPorts, p)
					log.Printf("Port %d no longer listening, tearing down tunnel", p)
					tm.teardown(p)
				}
			}
		}
	}
}

func logVerbose(format string, args ...any) {
	if verbose {
		log.Printf(format, args...)
	}
}
