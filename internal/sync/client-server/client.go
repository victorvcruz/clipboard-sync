package clientserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	syncTypes "github.com/victorvcruz/clipboard-sync/internal/sync"
)

type Client struct {
	targetIP   string
	targetPort string
	sourceID   string
	httpClient *http.Client
}

func NewClient(targetIP, targetPort, sourceID string) *Client {
	return &Client{
		targetIP:   targetIP,
		targetPort: targetPort,
		sourceID:   sourceID,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (sc *Client) SendClipboard(content string) error {
	data := syncTypes.ClipboardData{
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
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	fmt.Printf("Successfully sent clipboard to %s:%s\n", sc.targetIP, sc.targetPort)
	return nil
}
