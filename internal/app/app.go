package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/victorvcruz/clipboard-sync/internal/clipboard"
	"github.com/victorvcruz/clipboard-sync/internal/network/ip"
	"github.com/victorvcruz/clipboard-sync/internal/sync"
	clientserver "github.com/victorvcruz/clipboard-sync/internal/sync/client-server"
	"github.com/victorvcruz/clipboard-sync/internal/sync/p2p"
)

type App struct {
	version string
	config  *Config
}

func New(version string) *App {
	return &App{
		version: version,
	}
}

func (a *App) Run(ctx context.Context) error {
	config, err := ParseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	a.config = config

	if config.ShowVersion {
		fmt.Println(a.version)
		return nil
	}

	if config.ShowIP {
		ip.ShowAccessibleIP()
		return nil
	}

	return a.runSync(ctx)
}

func (a *App) runSync(ctx context.Context) error {
	a.printStartupInfo()

	clipboardManager := a.initializeClipboard()
	defer clipboardManager.Close()

	if a.config.P2PMode {
		return a.runP2PMode(ctx, clipboardManager)
	}
	return a.runClientServerMode(ctx, clipboardManager)
}

func (a *App) printStartupInfo() {
	fmt.Printf("Starting Clipboard Sync\n")
	fmt.Printf("Source ID: %s\n", a.config.SourceID)

	if a.config.P2PMode {
		fmt.Printf("Mode: P2P (Peer-to-Peer)\n")
		fmt.Printf("Listening on port: %s\n", a.config.Port)
		fmt.Printf("Discovery port: %d\n", a.config.DiscoveryPort)
		fmt.Printf("Auto-discovering peers on local network...\n")
	} else {
		fmt.Printf("Mode: Client-Server\n")
		fmt.Printf("Listening on port: %s\n", a.config.Port)
		fmt.Printf("Target: %s:%s\n", a.config.TargetIP, a.config.TargetPort)
	}

	fmt.Println("Press Ctrl+C to stop")
}

func (a *App) initializeClipboard() *clipboard.ClipboardManager {
	clipboardManager := clipboard.NewClipboardManager()
	if clipboardManager == nil {
		log.Fatalf("Failed to initialize clipboard manager")
	}
	return clipboardManager
}

func (a *App) runP2PMode(ctx context.Context, clipboardManager *clipboard.ClipboardManager) error {
	p2pSync := p2p.NewClipboardSync(a.config.SourceID, a.config.Port, a.config.DiscoveryPort)

	a.setupClipboardHandlers(clipboardManager, p2pSync, nil)

	if err := p2pSync.Start(ctx); err != nil {
		return fmt.Errorf("failed to start P2P sync: %w", err)
	}

	a.startPeerStatusReporter(ctx, p2pSync)
	a.startClipboardMonitoring(ctx, clipboardManager)

	<-ctx.Done()
	fmt.Println("\nShutting down...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Goodbye!")

	return nil
}

func (a *App) runClientServerMode(ctx context.Context, clipboardManager *clipboard.ClipboardManager) error {
	syncClient := clientserver.NewClient(a.config.TargetIP, a.config.TargetPort, a.config.SourceID)
	syncServer := clientserver.NewServer(a.config.Port, a.config.SourceID)

	a.setupClipboardHandlers(clipboardManager, nil, syncClient)

	syncServer.SetOnReceive(func(content, source string) {
		fmt.Printf("Received clipboard from %s: %s\n", source, a.truncateString(content, 50))
		if err := clipboardManager.SetClipboardExternal(content); err != nil {
			fmt.Printf("Failed to set clipboard: %v\n", err)
		}
	})

	syncServer.StartAsync()
	time.Sleep(100 * time.Millisecond)

	a.startClipboardMonitoring(ctx, clipboardManager)

	<-ctx.Done()
	fmt.Println("\nShutting down...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Goodbye!")

	return nil
}

func (a *App) setupClipboardHandlers(
	clipboardManager *clipboard.ClipboardManager,
	p2pSync sync.ClipboardBroadcaster,
	syncClient sync.ClipboardSender,
) {
	clipboardManager.SetOnChange(func(content string) {
		fmt.Printf("Clipboard changed: %s\n", a.truncateString(content, 50))

		if p2pSync != nil {
			if err := p2pSync.BroadcastClipboard(content); err != nil {
				fmt.Printf("Failed to broadcast clipboard: %v\n", err)
			}
		}

		if syncClient != nil {
			if err := syncClient.SendClipboard(content); err != nil {
				fmt.Printf("Failed to sync clipboard: %v\n", err)
			}
		}
	})

	if p2pSyncer, ok := p2pSync.(sync.ClipboardSyncer); ok {
		p2pSyncer.SetOnReceive(func(content, source string) {
			fmt.Printf("Received clipboard from peer %s: %s\n", source, a.truncateString(content, 50))
			if err := clipboardManager.SetClipboardExternal(content); err != nil {
				fmt.Printf("Failed to set clipboard: %v\n", err)
			}
		})
	}
}

func (a *App) startPeerStatusReporter(ctx context.Context, p2pSync sync.PeerManager) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				peerCount := p2pSync.GetPeerCount()
				if peerCount > 0 {
					fmt.Printf("Connected to %d peer(s)\n", peerCount)
				} else {
					fmt.Printf("No peers discovered yet...\n")
				}
			}
		}
	}()
}

func (a *App) startClipboardMonitoring(ctx context.Context, clipboardManager *clipboard.ClipboardManager) {
	go func() {
		if err := clipboardManager.StartMonitoring(ctx); err != nil && err != context.Canceled {
			log.Printf("Clipboard monitoring error: %v", err)
		}
	}()
}

func (a *App) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
