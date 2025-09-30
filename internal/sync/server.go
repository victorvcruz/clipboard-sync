package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SyncServer handles receiving clipboard data from remote devices
type SyncServer struct {
	port          string
	mux           *http.ServeMux
	onReceive     func(string, string) // callback(content, source)
	lastReceived  map[string]time.Time // track last received time by source
	sourceID      string
}

// NewSyncServer creates a new sync server instance
func NewSyncServer(port, sourceID string) *SyncServer {
	server := &SyncServer{
		port:         port,
		mux:          http.NewServeMux(),
		lastReceived: make(map[string]time.Time),
		sourceID:     sourceID,
	}
	
	server.setupRoutes()
	return server
}

// SetOnReceive sets the callback function for receiving clipboard data
func (ss *SyncServer) SetOnReceive(callback func(string, string)) {
	ss.onReceive = callback
}

// setupRoutes configures the HTTP routes
func (ss *SyncServer) setupRoutes() {
	ss.mux.HandleFunc("/clipboard", ss.handleClipboard)
	ss.mux.HandleFunc("/health", ss.handleHealth)
}

// handleClipboard handles incoming clipboard data
func (ss *SyncServer) handleClipboard(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var data ClipboardData
	
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Prevent processing data from the same source (avoid loops)
	if data.Source == ss.sourceID {
		fmt.Printf("Ignoring clipboard data from same source: %s\n", data.Source)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Check if this is a duplicate (within 1 second)
	if lastTime, exists := ss.lastReceived[data.Source]; exists {
		if time.Since(lastTime) < time.Second {
			fmt.Printf("Ignoring duplicate clipboard data from %s\n", data.Source)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	
	// Update last received time
	ss.lastReceived[data.Source] = time.Now()
	
	fmt.Printf("Received clipboard from %s: %s\n", data.Source, data.Content)
	
	// Call the callback function if set
	if ss.onReceive != nil {
		ss.onReceive(data.Content, data.Source)
	}
	
	w.WriteHeader(http.StatusOK)
}

// handleHealth handles health check requests
func (ss *SyncServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the HTTP server
func (ss *SyncServer) Start() error {
	fmt.Printf("Starting sync server on port %s\n", ss.port)
	return http.ListenAndServe(":"+ss.port, ss.mux)
}

// StartAsync starts the HTTP server in a goroutine
func (ss *SyncServer) StartAsync() {
	go func() {
		if err := ss.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
}