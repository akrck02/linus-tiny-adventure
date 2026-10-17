//go:build nintendo3ds

package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <sys/stat.h>

// readWhole reads a file whole into a buffer it allocates, or gives NULL.
static char *readWhole(const char *path, long *size) {
	FILE *file = fopen(path, "rb");
	if (file == NULL) return NULL;
	char *data = NULL;
	if (fseek(file, 0, SEEK_END) == 0 && (*size = ftell(file)) >= 0 && fseek(file, 0, SEEK_SET) == 0) {
		data = malloc(*size + 1);
		if (data != NULL && fread(data, 1, *size, file) != (size_t)*size) {
			free(data);
			data = NULL;
		}
	}
	fclose(file);
	return data;
}

// writeWhole writes a file whole: first beside it, then over it, so a write cut short leaves the
// old one.
static int writeWhole(const char *folder, const char *path, const char *partial, const char *data, long size) {
	mkdir(folder, 0777);
	FILE *file = fopen(partial, "wb");
	if (file == NULL) return -1;
	int written = fwrite(data, 1, size, file) == (size_t)size;
	if (fclose(file) != 0 || !written) return -1;
	remove(path);
	return rename(partial, path);
}
*/
import "C"

import "unsafe"

// The saves live on the SD card, in a folder of the game's own next to the homebrew, as the
// original keeps them in its user folder.
const saveFolder = "sdmc:/3ds/linus-tiny-adventure"

// readSave reads one of the game's files, or nothing if it is not there.
func readSave(name string) []byte {
	path := C.CString(saveFolder + "/" + name)
	defer C.free(unsafe.Pointer(path))
	var size C.long
	data := C.readWhole(path, &size)
	if data == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(data))
	return C.GoBytes(unsafe.Pointer(data), C.int(size))
}

// writeSave writes one of the game's files; it tells whether it could.
func writeSave(name string, data []byte) bool {
	folder := C.CString(saveFolder)
	path := C.CString(saveFolder + "/" + name)
	partial := C.CString(saveFolder + "/" + name + ".partial")
	defer C.free(unsafe.Pointer(folder))
	defer C.free(unsafe.Pointer(path))
	defer C.free(unsafe.Pointer(partial))
	if len(data) == 0 {
		return false
	}
	return C.writeWhole(folder, path, partial, (*C.char)(unsafe.Pointer(&data[0])), C.long(len(data))) == 0
}
