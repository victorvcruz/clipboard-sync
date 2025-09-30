package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ClipboardData represents the clipboard content structure
type ClipboardData struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// SyncClient handles sending clipboard data to remote devices
type SyncClient struct {
	targetIP   string
	targetPort string
	sourceID   string
	httpClient *http.Client
}

// NewSyncClient creates a new sync client instance
func NewSyncClient(targetIP, targetPort, sourceID string) *SyncClient {
	return &SyncClient{
		targetIP:   targetIP,
		targetPort: targetPort,
		sourceID:   sourceID,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendClipboard sends clipboard content to the target device
func (sc *SyncClient) SendClipboard(content string) error {
	data := ClipboardData{
		Content:   content,
		Timestamp: time.Now(),
		Source:    sc.sourceID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal clipboard data: %w", err)
	}

	url := fmt.Sprintf("http://%s:%s/clipboard", sc.targetIP, sc.targetPort)

	resp, err := sc.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send clipboard data to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	fmt.Printf("Successfully sent clipboard to %s:%s\n", sc.targetIP, sc.targetPort)
	return nil
}
