package ip

import (
	"fmt"
	"net"
	"strings"
)

func ShowAccessibleIP() {
	ip := getAccessibleIP()
	if ip == "" {
		fmt.Println("No accessible IP found")
		return
	}
	fmt.Println("Accessible IP:", ip)
}

func getAccessibleIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if strings.HasPrefix(iface.Name, "br-") || strings.HasPrefix(iface.Name, "veth") ||
			strings.HasPrefix(iface.Name, "docker") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipv4 := ipnet.IP.To4(); ipv4 != nil {
					ipStr := ipv4.String()
					if isLocalNetworkIP(ipStr) && isPreferredInterface(iface.Name) {
						return ipStr
					}
				}
			}
		}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if strings.HasPrefix(iface.Name, "br-") || strings.HasPrefix(iface.Name, "veth") ||
			strings.HasPrefix(iface.Name, "docker") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipv4 := ipnet.IP.To4(); ipv4 != nil {
					ipStr := ipv4.String()
					if isLocalNetworkIP(ipStr) {
						return ipStr
					}
				}
			}
		}
	}

	return ""
}

func isLocalNetworkIP(ip string) bool {
	return strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") ||
		(strings.HasPrefix(ip, "172.") && isPrivate172(ip))
}

func isPrivate172(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) < 2 {
		return false
	}

	if parts[0] != "172" {
		return false
	}

	second := parts[1]
	return second >= "16" && second <= "31"
}

func isPreferredInterface(name string) bool {
	preferredPrefixes := []string{"wl", "eth", "en", "wlan", "wifi"}

	for _, prefix := range preferredPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}
