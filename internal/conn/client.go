package conn

import (
	"context"
	"cp-sync/internal/clipboard"
	"log"
	"net"
	"time"
)

func StartClient(ip string) {
	serverAddr := net.UDPAddr{
		Port: 8080,
		IP:   net.ParseIP(ip),
	}
	conn, err := net.DialUDP("udp", nil, &serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	go receiveClipboardUpdates(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	watch := clipboard.Watch(context.Background())
	for {
		select {
		case data, ok := <-watch:
			if !ok {
				log.Println("Watch channel closed")
				return
			}
			log.Println("Clipboard data changed:", string(data))
			_, err = conn.Write(data)
			if err != nil {
				log.Println("Error sending clipboard data:", err)
			}
		}
	}
}

func receiveClipboardUpdates(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("Error receiving clipboard data:", err)
			continue
		}
		clipboard.Write(buf[:n])
	}
}
