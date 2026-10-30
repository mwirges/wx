package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func init() {
	// 40 MiB heap and a tighter GC keep this in-process runtime under iOS jetsam.
	applyMobileLimits()
}

// WxSetCacheDir tells the Go cache where to store JSON files.
// Call once at app startup before any WxRunJSON. Swift setenv is not
// enough: the c-archive Go runtime snapshots the process environment at
// dyld init, so libc setenv after load is invisible to os.Getenv. This
// export calls os.Setenv on the Go side so resolveDir sees WX_CACHE_DIR.
//
// dir must be an absolute path. An empty or relative path is ignored and
// the previous value (if any) is left alone. The function returns 0 on
// success and -1 when the path is rejected.
//
//export WxSetCacheDir
func WxSetCacheDir(dir *C.char) C.int {
	if dir == nil {
		return -1
	}
	path := strings.TrimSpace(C.GoString(dir))
	if path == "" || !filepath.IsAbs(path) {
		return -1
	}
	if err := os.Setenv("WX_CACHE_DIR", path); err != nil {
		return -1
	}
	return 0
}

// WxRunJSON runs one allow-listed wx command.
// argsJSON is a JSON array of strings, for example ["forecast","New Haven, IN"].
// The returned string is JSON. On failure it is an error object and the
// process keeps running. The caller frees the result with WxFreeString.
//
//export WxRunJSON
func WxRunJSON(argsJSON *C.char) (out *C.char) {
	defer func() {
		if rec := recover(); rec != nil {
			out = C.CString(errorJSON("panic", fmt.Sprint(rec)))
		}
	}()
	out = C.CString(RunJSON(C.GoString(argsJSON)))
	return out
}

// WxFreeString releases a string returned by WxRunJSON.
// A NULL pointer is ignored.
//
//export WxFreeString
func WxFreeString(p *C.char) {
	C.free(unsafe.Pointer(p))
}

func main() {}
