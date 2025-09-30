package clipboard

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>

int clipboard_test();
int clipboard_write(
	char*          typ,
	unsigned char* buf,
	size_t         n,
	uintptr_t      handle
);
unsigned long clipboard_read(char* typ, char **out);
*/
import "C"
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

const (
	FmtText = "UTF8_STRING"
)

var (
	lock           = sync.Mutex{}
	initOnce       sync.Once
	initError      error
	errUnavailable = errors.New("clipboard unavailable")
)

func Init() error {
	initOnce.Do(func() {
		ok := C.clipboard_test()
		if ok != 0 {
			initError = errUnavailable
		}
	})
	return initError
}

func Read() []byte {
	lock.Lock()
	defer lock.Unlock()

	buf, err := read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read clipboard err: %v\n", err)
		return nil
	}
	return buf
}

func Write(buf []byte) <-chan struct{} {
	lock.Lock()
	defer lock.Unlock()

	changed, err := write(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write to clipboard err: %v\n", err)
		return nil
	}
	return changed
}

func Watch(ctx context.Context) <-chan []byte {
	return watch(ctx)
}

func read() ([]byte, error) {
	ct := C.CString(FmtText)
	defer C.free(unsafe.Pointer(ct))

	var data *C.char
	n := C.clipboard_read(ct, &data)
	if data == nil {
		return nil, errUnavailable
	}
	defer C.free(unsafe.Pointer(data))
	switch {
	case n == 0:
		return nil, nil
	default:
		return C.GoBytes(unsafe.Pointer(data), C.int(n)), nil
	}
}

func write(buf []byte) (<-chan struct{}, error) {
	start := make(chan int)
	done := make(chan struct{}, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		cs := C.CString(FmtText)
		defer C.free(unsafe.Pointer(cs))

		h := cgo.NewHandle(start)
		var ok C.int
		if len(buf) == 0 {
			ok = C.clipboard_write(cs, nil, 0, C.uintptr_t(h))
		} else {
			ok = C.clipboard_write(cs, (*C.uchar)(unsafe.Pointer(&(buf[0]))), C.size_t(len(buf)), C.uintptr_t(h))
		}
		if ok != C.int(0) {
			fmt.Fprintf(os.Stderr, "write failed with status: %d\n", int(ok))
		}
		done <- struct{}{}
		close(done)
	}()

	status := <-start
	if status < 0 {
		return nil, errUnavailable
	}
	return done, nil
}

func watch(ctx context.Context) <-chan []byte {
	recv := make(chan []byte, 1)
	ti := time.NewTicker(time.Second)
	last := Read()
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(recv)
				return
			case <-ti.C:
				b := Read()
				if b == nil {
					continue
				}
				if !bytes.Equal(last, b) {
					recv <- b
					last = b
				}
			}
		}
	}()
	return recv
}

//export syncStatus
func syncStatus(h uintptr, val int) {
	v := cgo.Handle(h).Value().(chan int)
	v <- val
	cgo.Handle(h).Delete()
}
