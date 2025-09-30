#ifndef CLIPBOARD_NATIVE_H
#define CLIPBOARD_NATIVE_H

#include <stdlib.h>
#include <stddef.h>

// Function declarations for clipboard operations
int clipboard_test(void);
int clipboard_write_simple(const unsigned char *buf, size_t n);
unsigned long clipboard_read_simple(char **buf);

#endif // CLIPBOARD_NATIVE_H