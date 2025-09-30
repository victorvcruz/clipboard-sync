package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClipboardManager handles clipboard operations for Linux
type ClipboardManager struct {
	lastContent string
	onChange    func(string)
}

// NewClipboardManager creates a new clipboard manager instance
func NewClipboardManager() *ClipboardManager {
	return &ClipboardManager{
		lastContent: "",
	}
}

// SetOnChange sets the callback function for clipboard changes
func (cm *ClipboardManager) SetOnChange(callback func(string)) {
	cm.onChange = callback
}

// GetClipboard retrieves the current clipboard content using xclip
func (cm *ClipboardManager) GetClipboard() (string, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	var out bytes.Buffer
	cmd.Stdout = &out
	
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get clipboard content: %w", err)
	}
	
	return strings.TrimSpace(out.String()), nil
}

// SetClipboard sets the clipboard content using xclip
func (cm *ClipboardManager) SetClipboard(content string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(content)
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to set clipboard content: %w", err)
	}
	
	return nil
}

// StartMonitoring starts monitoring clipboard changes
func (cm *ClipboardManager) StartMonitoring(ctx context.Context) error {
	// Get initial content
	initialContent, err := cm.GetClipboard()
	if err != nil {
		// If we can't get initial content, start with empty string
		initialContent = ""
	}
	cm.lastContent = initialContent
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentContent, err := cm.GetClipboard()
			if err != nil {
				// Log error but continue monitoring
				fmt.Printf("Error getting clipboard content: %v\n", err)
				continue
			}
			
			// Check if content has changed
			if currentContent != cm.lastContent {
				cm.lastContent = currentContent
				if cm.onChange != nil && currentContent != "" {
					cm.onChange(currentContent)
				}
			}
		}
	}
}

// SetClipboardExternal sets clipboard content from external source (to avoid triggering onChange)
func (cm *ClipboardManager) SetClipboardExternal(content string) error {
	err := cm.SetClipboard(content)
	if err != nil {
		return err
	}
	
	// Update lastContent to prevent triggering onChange callback
	cm.lastContent = content
	return nil
}