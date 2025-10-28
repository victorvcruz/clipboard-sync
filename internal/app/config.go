package app

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port          string
	DiscoveryPort int
	SourceID      string
	P2PMode       bool
	TargetIP      string
	TargetPort    string
	ShowVersion   bool
	ShowIP        bool
}

func ParseConfig() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.Port, "port", "8080", "Port to listen on")
	flag.IntVar(&config.DiscoveryPort, "discovery-port", 9090, "UDP port for peer discovery")
	flag.StringVar(&config.SourceID, "source-id", "", "Source device identifier (auto-generated if empty)")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")
	flag.BoolVar(&config.ShowIP, "ip", false, "Show accessible IP address for other devices")
	flag.BoolVar(&config.P2PMode, "p2p", true, "Enable P2P mode (default: true)")
	flag.StringVar(&config.TargetIP, "target", "", "Target device IP address (legacy client-server mode)")
	flag.StringVar(&config.TargetPort, "target-port", "8080", "Target device port (legacy client-server mode)")

	flag.Parse()

	if err := config.validate(); err != nil {
		return nil, err
	}

	config.generateSourceID()
	return config, nil
}

func (c *Config) validate() error {
	if !c.P2PMode && c.TargetIP == "" {
		return fmt.Errorf("target IP address is required for client-server mode")
	}
	return nil
}

func (c *Config) generateSourceID() {
	if c.SourceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		c.SourceID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix()%10000)
	}
}
