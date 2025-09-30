package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"clipboard-sync/internal/clipboard"
	"clipboard-sync/internal/sync"
)

func main() {
	// Command line flags
	var (
		targetIP   = flag.String("target", "", "Target device IP address (required)")
		port       = flag.String("port", "8080", "Port to listen on")
		targetPort = flag.String("target-port", "8080", "Target device port")
		sourceID   = flag.String("source-id", "", "Source device identifier (auto-generated if empty)")
	)
	flag.Parse()

	// Validate required parameters
	if *targetIP == "" {
		fmt.Println("Error: target IP address is required")
		fmt.Println("Usage: clipboard-sync -target <IP_ADDRESS> [-port <PORT>] [-target-port <PORT>] [-source-id <ID>]")
		fmt.Println("Example: clipboard-sync -target 192.168.1.100")
		os.Exit(1)
	}

	// Generate source ID if not provided
	if *sourceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		*sourceID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix()%10000)
	}

	// Check if xclip is installed
	if err := checkXClipInstalled(); err != nil {
		log.Fatalf("xclip is required but not installed: %v", err)
	}

	fmt.Printf("Starting Clipboard Sync\n")
	fmt.Printf("Source ID: %s\n", *sourceID)
	fmt.Printf("Listening on port: %s\n", *port)
	fmt.Printf("Target: %s:%s\n", *targetIP, *targetPort)
	fmt.Println("Press Ctrl+C to stop")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	clipboardManager := clipboard.NewClipboardManager()
	syncClient := sync.NewSyncClient(*targetIP, *targetPort, *sourceID)
	syncServer := sync.NewSyncServer(*port, *sourceID)

	// Set up clipboard change handler
	clipboardManager.SetOnChange(func(content string) {
		fmt.Printf("Clipboard changed: %s\n", truncateString(content, 50))
		
		// Send to target device
		if err := syncClient.SendClipboard(content); err != nil {
			fmt.Printf("Failed to sync clipboard: %v\n", err)
		}
	})

	// Set up server receive handler
	syncServer.SetOnReceive(func(content, source string) {
		fmt.Printf("Received clipboard from %s: %s\n", source, truncateString(content, 50))
		
		// Set clipboard content (this won't trigger the onChange callback)
		if err := clipboardManager.SetClipboardExternal(content); err != nil {
			fmt.Printf("Failed to set clipboard: %v\n", err)
		}
	})

	// Start server
	syncServer.StartAsync()
	
	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Start clipboard monitoring in a goroutine
	go func() {
		if err := clipboardManager.StartMonitoring(ctx); err != nil && err != context.Canceled {
			log.Printf("Clipboard monitoring error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	
	// Give a moment for cleanup
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Goodbye!")
}

// checkXClipInstalled checks if xclip is available
func checkXClipInstalled() error {
	_, err := exec.LookPath("xclip")
	if err != nil {
		return fmt.Errorf("xclip not found in PATH. Install it with: sudo apt-get install xclip")
	}
	return nil
}

// truncateString truncates a string to a maximum length for display
func truncateString(s string, maxLen int) string {
	// Replace newlines with spaces for better display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}