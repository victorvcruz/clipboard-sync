package conn

import (
	"context"
	"cp-sync/internal/clipboard"
	"fmt"
	"log"
	"net"
)

func StartServer() {
	addr := net.UDPAddr{
		Port: 8080,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	clients := make(map[string]*net.UDPAddr)

	ip, err := GetLocalIP()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server started at:", ip)

	buf := make([]byte, 1024)
	go func() {
		for {
			n, clientAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Println("Error reading from UDP:", err)
				continue
			}
			clients[clientAddr.String()] = clientAddr

			log.Println("Received data from", clientAddr.String(), ":", string(buf[:n]))
			clipboard.Write(buf[:n])
			for _, addr := range clients {
				if addr.String() != clientAddr.String() {
					_, err := conn.WriteToUDP(buf[:n], addr)
					if err != nil {
						log.Println("Error writing to UDP:", err)
					}
				}
			}
		}
	}()

	watch := clipboard.Watch(context.Background())
	for {
		select {
		case data, ok := <-watch:
			if !ok {
				log.Println("Watch channel closed")
				return
			}
			log.Println("Clipboard data changed:", string(data))
			for _, addr := range clients {
				_, err := conn.WriteToUDP(data, addr)
				if err != nil {
					log.Println("Error sending clipboard data to", addr, ":", err)
				}
			}
			clipboard.Write(data)
		}
	}
}

func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		// Checa se o endereço é do tipo IP
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			// Exclui endereços IPv6
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("não foi possível encontrar o IP local")
}
