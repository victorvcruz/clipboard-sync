package sync

import "context"

type ClipboardSyncer interface {
	Start(ctx context.Context) error
	Stop()
	SetOnReceive(callback func(content, source string))
}

type ClipboardBroadcaster interface {
	BroadcastClipboard(content string) error
}

type PeerManager interface {
	GetPeerCount() int
	GetPeers() map[string]*PeerInfo
}

type ClipboardSender interface {
	SendClipboard(content string) error
}
