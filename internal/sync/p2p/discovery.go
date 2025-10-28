package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	syncTypes "github.com/victorvcruz/clipboard-sync/internal/sync"
)

type Discovery struct {
	peerID        string
	port          string
	discoveryPort int
	peers         map[string]*syncTypes.PeerInfo
	peersMutex    sync.RWMutex
	onPeerFound   func(*syncTypes.PeerInfo)
	onPeerLost    func(string)
	running       bool
	stopChan      chan struct{}
}

func NewDiscovery(peerID, port string, discoveryPort int) *Discovery {
	return &Discovery{
		peerID:        peerID,
		port:          port,
		discoveryPort: discoveryPort,
		peers:         make(map[string]*syncTypes.PeerInfo),
		stopChan:      make(chan struct{}),
	}
}

func (pd *Discovery) SetOnPeerFound(callback func(*syncTypes.PeerInfo)) {
	pd.onPeerFound = callback
}

func (pd *Discovery) SetOnPeerLost(callback func(string)) {
	pd.onPeerLost = callback
}

func (pd *Discovery) Start() error {
	if pd.running {
		return fmt.Errorf("discovery already running")
	}

	pd.running = true

	go pd.listenForPeers()
	go pd.announcePresence()
	go pd.cleanupStaleePeers()

	log.Printf("Peer discovery started on port %d", pd.discoveryPort)
	return nil
}

func (pd *Discovery) Stop() {
	if !pd.running {
		return
	}

	pd.running = false
	close(pd.stopChan)
	log.Printf("Peer discovery stopped")
}

func (pd *Discovery) GetPeers() map[string]*syncTypes.PeerInfo {
	pd.peersMutex.RLock()
	defer pd.peersMutex.RUnlock()

	peers := make(map[string]*syncTypes.PeerInfo)
	for id, peer := range pd.peers {
		peerCopy := *peer
		peers[id] = &peerCopy
	}
	return peers
}

func (pd *Discovery) listenForPeers() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", pd.discoveryPort))
	if err != nil {
		log.Printf("Error resolving UDP address: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("Error listening on UDP port %d: %v", pd.discoveryPort, err)
		return
	}
	defer conn.Close() //nolint:errcheck

	log.Printf("UDP listener started on port %d", pd.discoveryPort)
	buffer := make([]byte, 1024)

	for pd.running {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second)) //nolint:errcheck
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) {
				continue
			}
			if pd.running {
				log.Printf("Error reading UDP message: %v", err)
			}
			continue
		}

		log.Printf("Received UDP message from %s: %s", clientAddr.IP.String(), string(buffer[:n]))

		var msg syncTypes.DiscoveryMessage
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			log.Printf("Error unmarshaling discovery message: %v", err)
			continue
		}

		if msg.PeerID == pd.peerID {
			log.Printf("Ignoring message from self: %s", msg.PeerID)
			continue
		}

		log.Printf(
			"Processing discovery message: Type=%s, PeerID=%s, Port=%s",
			msg.Type,
			msg.PeerID,
			msg.Port,
		)
		pd.handleDiscoveryMessage(&msg, clientAddr.IP.String())
	}
}

func (pd *Discovery) handleDiscoveryMessage(msg *syncTypes.DiscoveryMessage, senderIP string) {
	switch msg.Type {
	case "announce":
		pd.addOrUpdatePeer(msg.PeerID, senderIP, msg.Port)
		pd.sendResponse(senderIP)
	case "response":
		pd.addOrUpdatePeer(msg.PeerID, senderIP, msg.Port)
	}
}

func (pd *Discovery) addOrUpdatePeer(peerID, ip, port string) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	isNew := false
	peer, exists := pd.peers[peerID]
	if !exists {
		peer = &syncTypes.PeerInfo{
			ID:   peerID,
			IP:   ip,
			Port: port,
		}
		pd.peers[peerID] = peer
		isNew = true
	}

	peer.LastSeen = time.Now()

	if isNew {
		log.Printf("Discovered new peer: %s (%s:%s)", peerID, ip, port)
		if pd.onPeerFound != nil {
			go pd.onPeerFound(peer)
		}
	}
}

func (pd *Discovery) announcePresence() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	pd.broadcastAnnouncement()
	pd.multicastAnnouncement()

	for {
		select {
		case <-pd.stopChan:
			return
		case <-ticker.C:
			pd.broadcastAnnouncement()
			pd.multicastAnnouncement()
		}
	}
}

func (pd *Discovery) broadcastAnnouncement() {
	msg := syncTypes.DiscoveryMessage{
		Type:    "announce",
		PeerID:  pd.peerID,
		Port:    pd.port,
		Version: "1.0",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling announcement: %v", err)
		return
	}

	broadcastAddrs := pd.getBroadcastAddresses()
	log.Printf("Broadcasting to %d addresses: %v", len(broadcastAddrs), broadcastAddrs)

	successCount := 0
	for _, broadcastAddr := range broadcastAddrs {
		addr, err := net.ResolveUDPAddr(
			"udp",
			fmt.Sprintf("%s:%d", broadcastAddr, pd.discoveryPort),
		)
		if err != nil {
			log.Printf(
				"Failed to resolve broadcast address %s:%d: %v",
				broadcastAddr,
				pd.discoveryPort,
				err,
			)
			continue
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Printf("Failed to dial UDP to %s:%d: %v", broadcastAddr, pd.discoveryPort, err)
			continue
		}

		_, err = conn.Write(data)
		conn.Close() //nolint:errcheck

		if err != nil {
			log.Printf("Failed to write to %s:%d: %v", broadcastAddr, pd.discoveryPort, err)
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		log.Printf("Warning: Failed to broadcast to any address")
	}
}

func (pd *Discovery) sendResponse(targetIP string) {
	msg := syncTypes.DiscoveryMessage{
		Type:    "response",
		PeerID:  pd.peerID,
		Port:    pd.port,
		Version: "1.0",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, pd.discoveryPort))
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close() //nolint:errcheck

	conn.Write(data) //nolint:errcheck
}

func (pd *Discovery) getBroadcastAddresses() []string {
	var broadcastAddrs []string
	seen := make(map[string]bool)

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error getting interfaces: %v", err)
		return []string{"255.255.255.255"}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}

		if strings.HasPrefix(iface.Name, "br-") || strings.HasPrefix(iface.Name, "veth") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipv4 := ipnet.IP.To4(); ipv4 != nil {
					mask := ipnet.Mask
					if len(mask) != 4 {
						continue
					}

					broadcast := make(net.IP, 4)
					for i := 0; i < 4; i++ {
						broadcast[i] = ipv4[i] | ^mask[i]
					}

					broadcastAddr := broadcast.String()

					if !seen[broadcastAddr] && isValidBroadcastAddr(broadcastAddr) {
						log.Printf(
							"Interface %s (%s): broadcast %s",
							iface.Name,
							ipv4.String(),
							broadcastAddr,
						)
						broadcastAddrs = append(broadcastAddrs, broadcastAddr)
						seen[broadcastAddr] = true
					}
				}
			}
		}
	}

	if !seen["255.255.255.255"] {
		broadcastAddrs = append(broadcastAddrs, "255.255.255.255")
	}

	log.Printf("Final broadcast addresses: %v", broadcastAddrs)
	return broadcastAddrs
}

func isValidBroadcastAddr(addr string) bool {
	invalid := []string{
		"0.0.0.255",
		"0.0.255.255",
		"0.255.255.255",
		"127.255.255.255",
	}

	for _, inv := range invalid {
		if addr == inv {
			return false
		}
	}

	return strings.HasSuffix(addr, ".255")
}

func (pd *Discovery) multicastAnnouncement() {
	multicastAddr := "224.0.0.251"

	msg := syncTypes.DiscoveryMessage{
		Type:    "announce",
		PeerID:  pd.peerID,
		Port:    pd.port,
		Version: "1.0",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling multicast announcement: %v", err)
		return
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastAddr, pd.discoveryPort))
	if err != nil {
		log.Printf("Failed to resolve multicast address: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Failed to dial multicast UDP: %v", err)
		return
	}
	defer conn.Close() //nolint:errcheck

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("Failed to send multicast announcement: %v", err)
	} else {
		log.Printf("Sent multicast announcement to %s:%d", multicastAddr, pd.discoveryPort)
	}
}

func (pd *Discovery) cleanupStaleePeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pd.stopChan:
			return
		case <-ticker.C:
			pd.peersMutex.Lock()
			now := time.Now()
			for peerID, peer := range pd.peers {
				if now.Sub(peer.LastSeen) > 60*time.Second {
					delete(pd.peers, peerID)
					log.Printf("Peer %s (%s:%s) timed out", peerID, peer.IP, peer.Port)
					if pd.onPeerLost != nil {
						go pd.onPeerLost(peerID)
					}
				}
			}
			pd.peersMutex.Unlock()
		}
	}
}
