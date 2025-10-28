package clipboard

import (
	"context"
	"fmt"
	"log"
	"time"
)

type ClipboardManager struct {
	native   *NativeClipboard
	onChange func(string)
}

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

func (cm *ClipboardManager) SetOnChange(callback func(string)) {
	cm.onChange = callback
}

func (cm *ClipboardManager) GetClipboard() (string, error) {
	if cm.native == nil {
		return "", fmt.Errorf("native clipboard not initialized")
	}

	content := cm.native.GetContent()
	return content, nil
}

func (cm *ClipboardManager) SetClipboard(content string) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	return cm.native.SetContent(content)
}

func (cm *ClipboardManager) StartMonitoring(ctx context.Context) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	log.Printf("Starting clipboard monitoring...")

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

func (cm *ClipboardManager) SetClipboardExternal(content string) error {
	if cm.native == nil {
		return fmt.Errorf("native clipboard not initialized")
	}

	err := cm.native.SetContent(content)
	if err != nil {
		return err
	}

	cm.native.lastContent = content
	return nil
}

func (cm *ClipboardManager) Close() {
	if cm.native != nil {
		cm.native.Close()
	}
}
