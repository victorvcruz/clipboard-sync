package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	syncTypes "github.com/victorvcruz/clipboard-sync/internal/sync"
)

type Server struct {
	port           string
	mux            *http.ServeMux
	onReceive      func(string, string)
	lastReceived   map[string]time.Time
	lastReceivedMu sync.RWMutex
	sourceID       string
	cleanupStop    chan struct{}
	running        bool
}

func NewServer(port, sourceID string) *Server {
	server := &Server{
		port:         port,
		mux:          http.NewServeMux(),
		lastReceived: make(map[string]time.Time),
		sourceID:     sourceID,
		cleanupStop:  make(chan struct{}),
		running:      false,
	}

	server.setupRoutes()
	return server
}

func (ss *Server) SetOnReceive(callback func(string, string)) {
	ss.onReceive = callback
}

func (ss *Server) setupRoutes() {
	ss.mux.HandleFunc("/clipboard", ss.handleClipboard)
	ss.mux.HandleFunc("/health", ss.handleHealth)
}

func (ss *Server) handleClipboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data syncTypes.ClipboardData

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if data.Source == ss.sourceID {
		fmt.Printf("Ignoring clipboard data from same source: %s\n", data.Source)
		w.WriteHeader(http.StatusOK)
		return
	}

	ss.lastReceivedMu.RLock()
	lastTime, exists := ss.lastReceived[data.Source]
	ss.lastReceivedMu.RUnlock()

	if exists && time.Since(lastTime) < time.Second {
		fmt.Printf("Ignoring duplicate clipboard data from %s\n", data.Source)
		w.WriteHeader(http.StatusOK)
		return
	}

	ss.lastReceivedMu.Lock()
	ss.lastReceived[data.Source] = time.Now()
	ss.lastReceivedMu.Unlock()

	fmt.Printf("Received clipboard from %s: %s\n", data.Source, data.Content)

	if ss.onReceive != nil {
		ss.onReceive(data.Content, data.Source)
	}

	w.WriteHeader(http.StatusOK)
}

func (ss *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK")) //nolint:errcheck
}

func (ss *Server) Start() error {
	fmt.Printf("Starting sync server on port %s\n", ss.port)
	return http.ListenAndServe(":"+ss.port, ss.mux)
}

func (ss *Server) StartAsync() {
	ss.running = true

	go ss.startMemoryCleanup()

	go func() {
		if err := ss.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
}

func (ss *Server) Stop() {
	if ss.running {
		ss.running = false
		close(ss.cleanupStop)
		log.Printf("SyncServer stopped")
	}
}

func (ss *Server) startMemoryCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	log.Printf("SyncServer memory cleanup routine started")

	for {
		select {
		case <-ss.cleanupStop:
			log.Printf("SyncServer memory cleanup routine stopped")
			return
		case <-ticker.C:
			ss.cleanupOldReceivedEntries()
		}
	}
}

func (ss *Server) cleanupOldReceivedEntries() {
	ss.lastReceivedMu.Lock()
	defer ss.lastReceivedMu.Unlock()

	now := time.Now()
	initialCount := len(ss.lastReceived)

	maxAge := 30 * time.Minute

	for source, timestamp := range ss.lastReceived {
		if now.Sub(timestamp) > maxAge {
			delete(ss.lastReceived, source)
		}
	}

	finalCount := len(ss.lastReceived)
	removedCount := initialCount - finalCount

	if removedCount > 0 {
		log.Printf(
			"SyncServer cleanup: removed %d old source entries, %d remaining",
			removedCount,
			finalCount,
		)
	}

	if finalCount > 500 {
		log.Printf(
			"Warning: SyncServer has %d source entries, performing emergency cleanup",
			finalCount,
		)
		ss.emergencyCleanupReceived()
	}
}

func (ss *Server) emergencyCleanupReceived() {
	type sourceEntry struct {
		source    string
		timestamp time.Time
	}

	entries := make([]sourceEntry, 0, len(ss.lastReceived))
	for source, timestamp := range ss.lastReceived {
		entries = append(entries, sourceEntry{source: source, timestamp: timestamp})
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.Before(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	ss.lastReceived = make(map[string]time.Time)
	keepCount := 50
	if len(entries) < keepCount {
		keepCount = len(entries)
	}

	for i := 0; i < keepCount; i++ {
		ss.lastReceived[entries[i].source] = entries[i].timestamp
	}

	removedCount := len(entries) - keepCount
	log.Printf(
		"SyncServer emergency cleanup: removed %d source entries, kept %d most recent",
		removedCount,
		keepCount,
	)
}
