package clipboard

/*
#cgo LDFLAGS: -ldl
#include "native.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

// NativeClipboard implementa acesso nativo ao clipboard usando X11 via CGO
type NativeClipboard struct {
	lastContent string
}

// NewNativeClipboard cria uma nova instância do clipboard nativo
func NewNativeClipboard() (*NativeClipboard, error) {
	// Testa se o X11 está disponível
	if C.clipboard_test() != 0 {
		return nil, fmt.Errorf(
			"failed to initialize X11 display - make sure X11 is running and DISPLAY is set",
		)
	}

	nc := &NativeClipboard{}

	// Tenta obter o conteúdo inicial
	if content := nc.GetContent(); content != "" {
		nc.lastContent = content
	}

	return nc, nil
}

// Close limpa os recursos
func (nc *NativeClipboard) Close() {
	// Recursos são limpos automaticamente
}

// SetContent define o conteúdo do clipboard
func (nc *NativeClipboard) SetContent(content string) error {
	if content == "" {
		return nil
	}

	contentBytes := []byte(content)
	result := C.clipboard_write_simple(
		(*C.uchar)(unsafe.Pointer(&contentBytes[0])),
		C.size_t(len(contentBytes)),
	)

	if result != 0 {
		return fmt.Errorf("failed to set clipboard content, status: %d", int(result))
	}

	nc.lastContent = content

	// Verificar se realmente foi definido lendo de volta
	// Aguardar um pouco para o sistema processar
	time.Sleep(50 * time.Millisecond)

	return nil
}

// GetContent obtém o conteúdo atual do clipboard
func (nc *NativeClipboard) GetContent() string {
	var data *C.char
	n := C.clipboard_read_simple(&data)

	if n == 0 || data == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(data))

	return C.GoString(data)
}

// HasChanged verifica se o clipboard mudou desde a última verificação
func (nc *NativeClipboard) HasChanged() bool {
	currentContent := nc.GetContent()
	if currentContent != nc.lastContent {
		nc.lastContent = currentContent
		return true
	}
	return false
}

// WatchClipboard monitora mudanças no clipboard e executa callback
func (nc *NativeClipboard) WatchClipboard(callback func(string)) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if nc.HasChanged() {
			if content := nc.GetContent(); content != "" {
				callback(content)
			}
		}
	}
}
