/*
 * native.c - Native X11 clipboard implementation for clipboard-sync
 * 
 * This file implements clipboard operations using X11 APIs through dynamic loading.
 * It provides functions to read from and write to the system clipboard without
 * requiring external tools like xclip.
 */

#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include <dlfcn.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// Global handle to dynamically loaded X11 library
void *libX11 = NULL;

// Function pointers to X11 functions (loaded dynamically)
Display* (*P_XOpenDisplay)(const char*);
void (*P_XCloseDisplay)(Display*);
Window (*P_XDefaultRootWindow)(Display*);
Window (*P_XCreateSimpleWindow)(Display*, Window, int, int, int, int, int, int, int);
Atom (*P_XInternAtom)(Display*, const char*, Bool);
void (*P_XSetSelectionOwner)(Display*, Atom, Window, Time);
Window (*P_XGetSelectionOwner)(Display*, Atom);
int (*P_XChangeProperty)(Display*, Window, Atom, Atom, int, int, const unsigned char*, int);
int (*P_XGetWindowProperty)(Display*, Window, Atom, long, long, Bool, Atom, Atom*, int*, unsigned long*, unsigned long*, unsigned char**);
void (*P_XFree)(void*);
void (*P_XDeleteProperty)(Display*, Window, Atom);
void (*P_XConvertSelection)(Display*, Atom, Atom, Atom, Window, Time);
int (*P_XPending)(Display*);
void (*P_XNextEvent)(Display*, XEvent*);
int (*P_XSendEvent)(Display*, Window, Bool, long, XEvent*);
void (*P_XFlush)(Display*);

/**
 * Initialize X11 library by dynamically loading libX11.so
 * Returns 1 on success, 0 on failure
 */
int initX11() {
    if (libX11) {
        return 1; // Already initialized
    }
    
    // Try to load libX11.so with different possible names
    libX11 = dlopen("libX11.so", RTLD_LAZY);
    if (!libX11) {
        libX11 = dlopen("libX11.so.6", RTLD_LAZY);
    }
    if (!libX11) {
        return 0; // Failed to load library
    }
    
    // Load all required X11 function pointers
    P_XOpenDisplay = (Display* (*)(const char*)) dlsym(libX11, "XOpenDisplay");
    P_XCloseDisplay = (void (*)(Display*)) dlsym(libX11, "XCloseDisplay");
    P_XDefaultRootWindow = (Window (*)(Display*)) dlsym(libX11, "XDefaultRootWindow");
    P_XCreateSimpleWindow = (Window (*)(Display*, Window, int, int, int, int, int, int, int)) dlsym(libX11, "XCreateSimpleWindow");
    P_XInternAtom = (Atom (*)(Display*, const char*, Bool)) dlsym(libX11, "XInternAtom");
    P_XSetSelectionOwner = (void (*)(Display*, Atom, Window, Time)) dlsym(libX11, "XSetSelectionOwner");
    P_XGetSelectionOwner = (Window (*)(Display*, Atom)) dlsym(libX11, "XGetSelectionOwner");
    P_XChangeProperty = (int (*)(Display*, Window, Atom, Atom, int, int, const unsigned char*, int)) dlsym(libX11, "XChangeProperty");
    P_XGetWindowProperty = (int (*)(Display*, Window, Atom, long, long, Bool, Atom, Atom*, int*, unsigned long*, unsigned long*, unsigned char**)) dlsym(libX11, "XGetWindowProperty");
    P_XFree = (void (*)(void*)) dlsym(libX11, "XFree");
    P_XDeleteProperty = (void (*)(Display*, Window, Atom)) dlsym(libX11, "XDeleteProperty");
    P_XConvertSelection = (void (*)(Display*, Atom, Atom, Atom, Window, Time)) dlsym(libX11, "XConvertSelection");
    P_XPending = (int (*)(Display*)) dlsym(libX11, "XPending");
    P_XNextEvent = (void (*)(Display*, XEvent*)) dlsym(libX11, "XNextEvent");
    P_XSendEvent = (int (*)(Display*, Window, Bool, long, XEvent*)) dlsym(libX11, "XSendEvent");
    P_XFlush = (void (*)(Display*)) dlsym(libX11, "XFlush");
    
    return 1; // Success
}

/**
 * Test if X11 display can be opened
 * Returns 0 on success, -1 on failure
 */
int clipboard_test() {
    if (!initX11()) {
        return -1;
    }

    Display* d = (*P_XOpenDisplay)(NULL);
    if (d == NULL) {
        return -1;
    }
    (*P_XCloseDisplay)(d);
    return 0;
}

/**
 * Write content to clipboard
 * 
 * @param buf Buffer containing the text to write
 * @param n Length of the buffer
 * @return 0 on success, negative on error
 */
int clipboard_write_simple(const unsigned char *buf, size_t n) {
    if (!initX11()) {
        return -1;
    }

    Display* d = (*P_XOpenDisplay)(NULL);
    if (d == NULL) {
        return -1;
    }
    
    // Create a simple window to own the selection
    Window w = (*P_XCreateSimpleWindow)(d, (*P_XDefaultRootWindow)(d), 0, 0, 1, 1, 0, 0, 0);
    
    // Get required atoms
    Atom sel = (*P_XInternAtom)(d, "CLIPBOARD", False);
    Atom atomString = (*P_XInternAtom)(d, "UTF8_STRING", False);
    Atom atomText = (*P_XInternAtom)(d, "TEXT", False);
    Atom atomTargets = (*P_XInternAtom)(d, "TARGETS", False);

    // Store the data in a window property
    (*P_XChangeProperty)(d, w, sel, atomString, 8, PropModeReplace, buf, n);
    
    // Set ourselves as the selection owner
    (*P_XSetSelectionOwner)(d, sel, w, CurrentTime);
    (*P_XFlush)(d);
    
    // Verify we became the owner
    if ((*P_XGetSelectionOwner)(d, sel) != w) {
        (*P_XCloseDisplay)(d);
        return -2;
    }
    
    // Handle selection requests for a brief period
    // This is crucial for the clipboard to work properly with other applications
    int timeout = 50; // 500ms timeout
    while (timeout-- > 0) {
        if ((*P_XPending)(d)) {
            XEvent event;
            (*P_XNextEvent)(d, &event);
            
            if (event.type == SelectionRequest) {
                XSelectionRequestEvent *req = &event.xselectionrequest;
                XSelectionEvent reply;
                
                // Prepare reply event
                reply.type = SelectionNotify;
                reply.display = req->display;
                reply.requestor = req->requestor;
                reply.selection = req->selection;
                reply.target = req->target;
                reply.property = req->property;
                reply.time = req->time;
                
                // Handle different target types
                if (req->target == atomString || req->target == atomText) {
                    // Provide the clipboard content
                    (*P_XChangeProperty)(req->display, req->requestor, req->property,
                                       atomString, 8, PropModeReplace, buf, n);
                } else if (req->target == atomTargets) {
                    // Provide list of supported targets
                    Atom targets[] = {atomTargets, atomString, atomText};
                    (*P_XChangeProperty)(req->display, req->requestor, req->property,
                                       XA_ATOM, 32, PropModeReplace,
                                       (unsigned char*)targets, 3);
                } else {
                    // Unsupported target
                    reply.property = None;
                }
                
                // Send the reply
                (*P_XSendEvent)(req->display, req->requestor, False, 0, (XEvent*)&reply);
                (*P_XFlush)(req->display);
            }
        }
        usleep(10000); // 10ms
    }
    
    (*P_XCloseDisplay)(d);
    return 0;
}

/**
 * Read content from clipboard
 * 
 * @param buf Pointer to buffer that will be allocated and filled with clipboard content
 * @return Number of bytes read, 0 if empty or error
 * 
 * Note: Caller is responsible for freeing the allocated buffer
 */
unsigned long clipboard_read_simple(char **buf) {
    if (!initX11()) {
        return 0;
    }

    Display* d = (*P_XOpenDisplay)(NULL);
    if (d == NULL) {
        return 0;
    }

    // Create a window to receive the selection
    Window w = (*P_XCreateSimpleWindow)(d, (*P_XDefaultRootWindow)(d), 0, 0, 1, 1, 0, 0, 0);
    
    // Get required atoms
    Atom sel = (*P_XInternAtom)(d, "CLIPBOARD", False);
    Atom prop = (*P_XInternAtom)(d, "SYNC_PROP", False);
    Atom target = (*P_XInternAtom)(d, "UTF8_STRING", False);
    
    // Request clipboard content
    (*P_XConvertSelection)(d, sel, target, prop, w, CurrentTime);
    (*P_XFlush)(d);
    
    // Wait for SelectionNotify event with timeout
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
            
            // Get the clipboard data from the window property
            int result = (*P_XGetWindowProperty)(d, w, prop, 0, 1024, False, 
                                               AnyPropertyType, &actual, &format, 
                                               &nitems, &bytes_after, &data);
            
            if (result == Success && data && nitems > 0) {
                // Allocate buffer and copy data
                *buf = (char*)malloc(nitems + 1);
                memcpy(*buf, data, nitems);
                (*buf)[nitems] = '\0'; // Null terminate
                
                // Clean up
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