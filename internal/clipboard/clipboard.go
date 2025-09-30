package clipboard

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ClipboardManager handles clipboard operations using native CGO implementation
type ClipboardManager struct {
	native   *NativeClipboard
	onChange func(string)
}

// NewClipboardManager creates a new clipboard manager instance
func NewClipboardManager() *ClipboardManager {
	native, err := NewNativeClipboard()
	if err != nil {
		log.Printf("Error: Failed to initialize native clipboard: %v", err)
		log.Printf("Make sure X11 is running and DISPLAY environment variable is set")
		return nil
	}

	log.Printf("Native clipboard initialized successfully")
	return &ClipboardManager{
		native: native,
	}
}

// SetOnChange sets the callback function for clipboard changes
func (cm *ClipboardManager) SetOnChange(callback func(string)) {
	cm.onChange = callback
}

// GetClipboard retrieves the current clipboard content using native implementation
func (cm *ClipboardManager) GetClipboard() (string, error) {
	if cm.native == nil {
		return "", fmt.Errorf("native clipboard not initialized")
	}

	content := cm.native.GetContent()
	return content, nil
}

// SetClipboard sets the clipboard content using native implementation
func (cm *ClipboardManager) SetClipboard(content string) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	return cm.native.SetContent(content)
}

// StartMonitoring starts monitoring clipboard changes using native implementation
func (cm *ClipboardManager) StartMonitoring(ctx context.Context) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	log.Printf("Starting clipboard monitoring...")

	// Get initial content
	initialContent := cm.native.GetContent()
	cm.native.lastContent = initialContent

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if cm.native.HasChanged() {
				currentContent := cm.native.GetContent()
				if cm.onChange != nil && currentContent != "" {
					cm.onChange(currentContent)
				}
			}
		}
	}
}

// SetClipboardExternal sets clipboard content from external source (to avoid triggering onChange)
func (cm *ClipboardManager) SetClipboardExternal(content string) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	err := cm.native.SetContent(content)
	if err != nil {
		return err
	}

	// Update lastContent to prevent triggering onChange callback
	cm.native.lastContent = content
	return nil
}

// Close cleans up native clipboard resources
func (cm *ClipboardManager) Close() {
	if cm.native != nil {
		cm.native.Close()
	}
}
