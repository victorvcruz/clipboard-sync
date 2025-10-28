package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	syncTypes "github.com/victorvcruz/clipboard-sync/internal/sync"
)

type ClipboardSync struct {
	peerID        string
	port          string
	discoveryPort int
	discovery     *Discovery
	server        *Server
	httpClient    *http.Client
	peers         map[string]*syncTypes.PeerInfo
	peersMutex    sync.RWMutex
	onReceive     func(string, string)
	lastSent      map[string]time.Time
	lastSentMutex sync.RWMutex
	running       bool
	cleanupStop   chan struct{}
}

func NewClipboardSync(peerID, port string, discoveryPort int) *ClipboardSync {
	p2p := &ClipboardSync{
		peerID:        peerID,
		port:          port,
		discoveryPort: discoveryPort,
		peers:         make(map[string]*syncTypes.PeerInfo),
		lastSent:      make(map[string]time.Time),
		cleanupStop:   make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}

	p2p.discovery = NewDiscovery(peerID, port, discoveryPort)
	p2p.discovery.SetOnPeerFound(p2p.onPeerFound)
	p2p.discovery.SetOnPeerLost(p2p.onPeerLost)

	p2p.server = NewServer(port, peerID)
	p2p.server.SetOnReceive(p2p.handleReceiveClipboard)

	return p2p
}

func (p2p *ClipboardSync) SetOnReceive(callback func(string, string)) {
	p2p.onReceive = callback
}

func (p2p *ClipboardSync) Start(ctx context.Context) error {
	if p2p.running {
		return fmt.Errorf("P2P sync already running")
	}

	p2p.running = true

	p2p.server.StartAsync()

	if err := p2p.discovery.Start(); err != nil {
		return fmt.Errorf("failed to start peer discovery: %w", err)
	}

	go p2p.startMemoryCleanup()

	log.Printf(
		"P2P Clipboard Sync started on port %s (discovery port %d)",
		p2p.port,
		p2p.discoveryPort,
	)

	go func() {
		<-ctx.Done()
		p2p.Stop()
	}()

	return nil
}

func (p2p *ClipboardSync) Stop() {
	if !p2p.running {
		return
	}

	p2p.running = false
	p2p.discovery.Stop()

	p2p.server.Stop()

	close(p2p.cleanupStop)

	log.Printf("P2P Clipboard Sync stopped")
}

func (p2p *ClipboardSync) BroadcastClipboard(content string) error {
	if !p2p.running {
		return fmt.Errorf("P2P sync not running")
	}

	p2p.peersMutex.RLock()
	peers := make([]*syncTypes.PeerInfo, 0, len(p2p.peers))
	for _, peer := range p2p.peers {
		peers = append(peers, peer)
	}
	p2p.peersMutex.RUnlock()

	if len(peers) == 0 {
		log.Printf("No peers available to broadcast clipboard")
		return nil
	}

	contentHash := fmt.Sprintf("%x", content)
	p2p.lastSentMutex.RLock()
	if lastTime, exists := p2p.lastSent[contentHash]; exists {
		if time.Since(lastTime) < 2*time.Second {
			p2p.lastSentMutex.RUnlock()
			log.Printf("Skipping duplicate clipboard broadcast")
			return nil
		}
	}
	p2p.lastSentMutex.RUnlock()

	p2p.lastSentMutex.Lock()
	p2p.lastSent[contentHash] = time.Now()
	p2p.lastSentMutex.Unlock()

	data := syncTypes.ClipboardData{
		Content:   content,
		Timestamp: time.Now(),
		Source:    p2p.peerID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal clipboard data: %w", err)
	}

	var wg sync.WaitGroup
	successCount := 0
	var successMutex sync.Mutex

	for _, peer := range peers {
		wg.Add(1)
		go func(peer *syncTypes.PeerInfo) {
			defer wg.Done()

			url := fmt.Sprintf("http://%s:%s/clipboard", peer.IP, peer.Port)

			resp, err := p2p.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf(
					"Failed to send clipboard to peer %s (%s:%s): %v",
					peer.ID,
					peer.IP,
					peer.Port,
					err,
				)
				return
			}
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode == http.StatusOK {
				successMutex.Lock()
				successCount++
				successMutex.Unlock()
				log.Printf(
					"Successfully sent clipboard to peer %s (%s:%s)",
					peer.ID,
					peer.IP,
					peer.Port,
				)
			} else {
				log.Printf("Peer %s (%s:%s) returned status %d", peer.ID, peer.IP, peer.Port, resp.StatusCode)
			}
		}(peer)
	}

	wg.Wait()

	if successCount > 0 {
		log.Printf("Clipboard broadcasted to %d/%d peers", successCount, len(peers))
	} else {
		return fmt.Errorf("failed to send clipboard to any peer")
	}

	return nil
}

func (p2p *ClipboardSync) GetPeers() map[string]*syncTypes.PeerInfo {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()

	peers := make(map[string]*syncTypes.PeerInfo)
	for id, peer := range p2p.peers {
		peerCopy := *peer
		peers[id] = &peerCopy
	}
	return peers
}

func (p2p *ClipboardSync) GetPeerCount() int {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()
	return len(p2p.peers)
}

func (p2p *ClipboardSync) onPeerFound(peer *syncTypes.PeerInfo) {
	p2p.peersMutex.Lock()
	p2p.peers[peer.ID] = peer
	p2p.peersMutex.Unlock()

	log.Printf("Added peer to P2P network: %s (%s:%s)", peer.ID, peer.IP, peer.Port)
}

func (p2p *ClipboardSync) onPeerLost(peerID string) {
	p2p.peersMutex.Lock()
	if peer, exists := p2p.peers[peerID]; exists {
		delete(p2p.peers, peerID)
		log.Printf("Removed peer from P2P network: %s (%s:%s)", peer.ID, peer.IP, peer.Port)
	}
	p2p.peersMutex.Unlock()
}

func (p2p *ClipboardSync) handleReceiveClipboard(content, source string) {
	log.Printf("Received clipboard from peer %s: %s", source, truncateForLog(content, 50))

	if p2p.onReceive != nil {
		p2p.onReceive(content, source)
	}

	go p2p.relayToOtherPeers(content, source)
}

func (p2p *ClipboardSync) relayToOtherPeers(content, originalSource string) {
	p2p.peersMutex.RLock()
	peers := make([]*syncTypes.PeerInfo, 0)
	for _, peer := range p2p.peers {
		if peer.ID != originalSource {
			peers = append(peers, peer)
		}
	}
	p2p.peersMutex.RUnlock()

	if len(peers) == 0 {
		return
	}

	contentHash := fmt.Sprintf("%x", content)
	p2p.lastSentMutex.RLock()
	if lastTime, exists := p2p.lastSent[contentHash]; exists {
		if time.Since(lastTime) < 1*time.Second {
			p2p.lastSentMutex.RUnlock()
			return
		}
	}
	p2p.lastSentMutex.RUnlock()

	p2p.lastSentMutex.Lock()
	p2p.lastSent[contentHash] = time.Now()
	p2p.lastSentMutex.Unlock()

	data := syncTypes.ClipboardData{
		Content:   content,
		Timestamp: time.Now(),
		Source:    originalSource,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal relay data: %v", err)
		return
	}

	for _, peer := range peers {
		go func(peer *syncTypes.PeerInfo) {
			url := fmt.Sprintf("http://%s:%s/clipboard", peer.IP, peer.Port)

			resp, err := p2p.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("Failed to relay clipboard to peer %s: %v", peer.ID, err)
				return
			}
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode == http.StatusOK {
				log.Printf("Relayed clipboard to peer %s", peer.ID)
			}
		}(peer)
	}
}

func (p2p *ClipboardSync) startMemoryCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("Memory cleanup routine started (interval: 5 minutes)")

	for {
		select {
		case <-p2p.cleanupStop:
			log.Printf("Memory cleanup routine stopped")
			return
		case <-ticker.C:
			p2p.cleanupOldEntries()
		}
	}
}

func (p2p *ClipboardSync) cleanupOldEntries() {
	p2p.lastSentMutex.Lock()
	defer p2p.lastSentMutex.Unlock()

	now := time.Now()
	initialCount := len(p2p.lastSent)

	maxAge := 10 * time.Minute

	for hash, timestamp := range p2p.lastSent {
		if now.Sub(timestamp) > maxAge {
			delete(p2p.lastSent, hash)
		}
	}

	finalCount := len(p2p.lastSent)
	removedCount := initialCount - finalCount

	if removedCount > 0 {
		log.Printf(
			"Memory cleanup: removed %d old entries, %d entries remaining",
			removedCount,
			finalCount,
		)
	}

	if finalCount > 1000 {
		log.Printf("Warning: lastSent map has %d entries, performing emergency cleanup", finalCount)
		p2p.emergencyCleanup()
	}
}

func (p2p *ClipboardSync) emergencyCleanup() {
	type entry struct {
		hash      string
		timestamp time.Time
	}

	entries := make([]entry, 0, len(p2p.lastSent))
	for hash, timestamp := range p2p.lastSent {
		entries = append(entries, entry{hash: hash, timestamp: timestamp})
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.Before(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	p2p.lastSent = make(map[string]time.Time)
	keepCount := 100
	if len(entries) < keepCount {
		keepCount = len(entries)
	}

	for i := 0; i < keepCount; i++ {
		p2p.lastSent[entries[i].hash] = entries[i].timestamp
	}

	removedCount := len(entries) - keepCount
	log.Printf(
		"Emergency cleanup: removed %d entries, kept %d most recent entries",
		removedCount,
		keepCount,
	)
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
