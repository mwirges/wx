package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

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
