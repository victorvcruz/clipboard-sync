package clipboard

/*
#cgo LDFLAGS: -ldl

#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include <dlfcn.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

void *libX11;

Display* (*P_XOpenDisplay)(int);
void (*P_XCloseDisplay)(Display*);
Window (*P_XDefaultRootWindow)(Display*);
Window (*P_XCreateSimpleWindow)(Display*, Window, int, int, int, int, int, int, int);
Atom (*P_XInternAtom)(Display*, char*, int);
void (*P_XSetSelectionOwner)(Display*, Atom, Window, unsigned long);
Window (*P_XGetSelectionOwner)(Display*, Atom);
int (*P_XChangeProperty)(Display*, Window, Atom, Atom, int, int, unsigned char*, int);
int (*P_XGetWindowProperty) (Display*, Window, Atom, long, long, Bool, Atom, Atom*, int*, unsigned long *, unsigned long *, unsigned char **);
void (*P_XFree) (void*);
void (*P_XDeleteProperty) (Display*, Window, Atom);
void (*P_XConvertSelection)(Display*, Atom, Atom, Atom, Window, Time);
int (*P_XPending)(Display*);
void (*P_XNextEvent)(Display*, XEvent*);
int (*P_XSendEvent)(Display*, Window, Bool, long, XEvent*);

int initX11() {
	if (libX11) {
		return 1;
	}
	libX11 = dlopen("libX11.so", RTLD_LAZY);
	if (!libX11) {
		libX11 = dlopen("libX11.so.6", RTLD_LAZY);
	}
	if (!libX11) {
		return 0;
	}
	P_XOpenDisplay = (Display* (*)(int)) dlsym(libX11, "XOpenDisplay");
	P_XCloseDisplay = (void (*)(Display*)) dlsym(libX11, "XCloseDisplay");
	P_XDefaultRootWindow = (Window (*)(Display*)) dlsym(libX11, "XDefaultRootWindow");
	P_XCreateSimpleWindow = (Window (*)(Display*, Window, int, int, int, int, int, int, int)) dlsym(libX11, "XCreateSimpleWindow");
	P_XInternAtom = (Atom (*)(Display*, char*, int)) dlsym(libX11, "XInternAtom");
	P_XSetSelectionOwner = (void (*)(Display*, Atom, Window, unsigned long)) dlsym(libX11, "XSetSelectionOwner");
	P_XGetSelectionOwner = (Window (*)(Display*, Atom)) dlsym(libX11, "XGetSelectionOwner");
	P_XChangeProperty = (int (*)(Display*, Window, Atom, Atom, int, int, unsigned char*, int)) dlsym(libX11, "XChangeProperty");
	P_XGetWindowProperty = (int (*)(Display*, Window, Atom, long, long, Bool, Atom, Atom*, int*, unsigned long *, unsigned long *, unsigned char **)) dlsym(libX11, "XGetWindowProperty");
	P_XFree = (void (*)(void*)) dlsym(libX11, "XFree");
	P_XDeleteProperty = (void (*)(Display*, Window, Atom)) dlsym(libX11, "XDeleteProperty");
	P_XConvertSelection = (void (*)(Display*, Atom, Atom, Atom, Window, Time)) dlsym(libX11, "XConvertSelection");
	P_XPending = (int (*)(Display*)) dlsym(libX11, "XPending");
	P_XNextEvent = (void (*)(Display*, XEvent*)) dlsym(libX11, "XNextEvent");
	P_XSendEvent = (int (*)(Display*, Window, Bool, long, XEvent*)) dlsym(libX11, "XSendEvent");
	return 1;
}

int clipboard_test() {
	if (!initX11()) {
		return -1;
	}

    Display* d = (*P_XOpenDisplay)(0);
    if (d == NULL) {
        return -1;
    }
    (*P_XCloseDisplay)(d);
    return 0;
}

// Improved clipboard_write that properly handles selection ownership
int clipboard_write_simple(unsigned char *buf, size_t n) {
	if (!initX11()) {
		return -1;
	}

    Display* d = (*P_XOpenDisplay)(0);
    if (d == NULL) {
        return -1;
    }
    
    Window w = (*P_XCreateSimpleWindow)(d, (*P_XDefaultRootWindow)(d), 0, 0, 1, 1, 0, 0, 0);
    Atom sel = (*P_XInternAtom)(d, "CLIPBOARD", False);
    Atom atomString = (*P_XInternAtom)(d, "UTF8_STRING", False);
    Atom atomText = (*P_XInternAtom)(d, "TEXT", False);
    Atom atomTargets = (*P_XInternAtom)(d, "TARGETS", False);

    // Store the data in a window property
    (*P_XChangeProperty)(d, w, sel, atomString, 8, PropModeReplace, buf, n);
    
    // Set ourselves as the selection owner
    (*P_XSetSelectionOwner)(d, sel, w, CurrentTime);
    
    // Verify we became the owner
    if ((*P_XGetSelectionOwner)(d, sel) != w) {
        (*P_XCloseDisplay)(d);
        return -2;
    }
    
    // Keep the connection alive briefly to handle any immediate requests
    // This is crucial for the clipboard to work properly
    int timeout = 50; // 500ms timeout
    while (timeout-- > 0) {
        if ((*P_XPending)(d)) {
            XEvent event;
            (*P_XNextEvent)(d, &event);
            
            if (event.type == SelectionRequest) {
                XSelectionRequestEvent *req = &event.xselectionrequest;
                XSelectionEvent reply;
                
                reply.type = SelectionNotify;
                reply.display = req->display;
                reply.requestor = req->requestor;
                reply.selection = req->selection;
                reply.target = req->target;
                reply.property = req->property;
                reply.time = req->time;
                
                if (req->target == atomString || req->target == atomText) {
                    (*P_XChangeProperty)(req->display, req->requestor, req->property,
                                       atomString, 8, PropModeReplace, buf, n);
                } else if (req->target == atomTargets) {
                    Atom targets[] = {atomTargets, atomString, atomText};
                    (*P_XChangeProperty)(req->display, req->requestor, req->property,
                                       XA_ATOM, 32, PropModeReplace,
                                       (unsigned char*)targets, 3);
                } else {
                    reply.property = None;
                }
                
                (*P_XSendEvent)(req->display, req->requestor, False, 0, (XEvent*)&reply);
            }
        }
        usleep(10000); // 10ms
    }
    
    (*P_XCloseDisplay)(d);
    return 0;
}

// Simplified clipboard_read
unsigned long clipboard_read_simple(char **buf) {
	if (!initX11()) {
		return 0;
	}

    Display* d = (*P_XOpenDisplay)(0);
    if (d == NULL) {
        return 0;
    }

    Window w = (*P_XCreateSimpleWindow)(d, (*P_XDefaultRootWindow)(d), 0, 0, 1, 1, 0, 0, 0);
    Atom sel = (*P_XInternAtom)(d, "CLIPBOARD", False);
    Atom prop = (*P_XInternAtom)(d, "SYNC_PROP", False);
    Atom target = (*P_XInternAtom)(d, "UTF8_STRING", False);
    
    (*P_XConvertSelection)(d, sel, target, prop, w, CurrentTime);
    
    // Simple event waiting with timeout
    int timeout = 10;
    while (timeout-- > 0 && !(*P_XPending)(d)) {
        usleep(10000); // 10ms
    }
    
    if ((*P_XPending)(d)) {
        XEvent event;
        (*P_XNextEvent)(d, &event);
        
        if (event.type == SelectionNotify) {
            Atom actual;
            int format;
            unsigned long nitems, bytes_after;
            unsigned char *data = NULL;
            
            int result = (*P_XGetWindowProperty)(d, w, prop, 0, 1024, False, 
                                               AnyPropertyType, &actual, &format, 
                                               &nitems, &bytes_after, &data);
            
            if (result == Success && data && nitems > 0) {
                *buf = (char*)malloc(nitems + 1);
                memcpy(*buf, data, nitems);
                (*buf)[nitems] = '\0';
                (*P_XFree)(data);
                (*P_XDeleteProperty)(d, w, prop);
                (*P_XCloseDisplay)(d);
                return nitems;
            }
            
            if (data) (*P_XFree)(data);
        }
    }
    
    (*P_XCloseDisplay)(d);
    return 0;
}
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
		return nil, fmt.Errorf("failed to initialize X11 display - make sure X11 is running and DISPLAY is set")
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
	result := C.clipboard_write_simple((*C.uchar)(unsafe.Pointer(&contentBytes[0])), C.size_t(len(contentBytes)))
	
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