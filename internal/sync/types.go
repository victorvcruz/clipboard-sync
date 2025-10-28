package sync

import "time"

type PeerInfo struct {
	ID       string    `json:"id"`
	IP       string    `json:"ip"`
	Port     string    `json:"port"`
	LastSeen time.Time `json:"last_seen"`
}

type ClipboardData struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

type DiscoveryMessage struct {
	Type    string `json:"type"`
	PeerID  string `json:"peer_id"`
	Port    string `json:"port"`
	Version string `json:"version"`
}
