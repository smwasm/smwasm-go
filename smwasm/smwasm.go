package smwasm

/*
#cgo LDFLAGS: -L./lib -lsmwasmgo_rustlib
#include <stdint.h>
#include <stdlib.h>
extern int32_t smwasm_load(const char *szWasm, int32_t space);
extern char *smwasm_call(const char *szInput);
extern int32_t *smwasm_register(const char *szDefine);
*/
import "C"
import (
	"unsafe"
)

func RegisterUsage(itdef string, fnc NativeFunc) {
	prepareSmartModule()

	name := getItem(itdef, "$usage")
	if len(name) > 0 {
		nativeFuncMap[name] = fnc

		cdef := C.CString(itdef)
		defer C.free(unsafe.Pointer(cdef))
		C.smwasm_register(cdef)
	}
}

func LoadWasm(wasm string, space int) int {
	prepareSmartModule()
	path := C.CString(wasm)
	defer C.free(unsafe.Pointer(path))

	ret := C.smwasm_load(path, C.int32_t(space))
	return int(ret)
}

func Call(inputText string) string {
	_input := C.CString(inputText)
	defer C.free(unsafe.Pointer(_input))

	_output := callNative(_input)
	defer C.free(unsafe.Pointer(_output))

	txt := C.GoString(_output)
	if len(txt) > 0 {
		return txt
	}

	_input2 := C.CString(inputText)
	_output2 := C.smwasm_call(_input2)
	defer C.free(unsafe.Pointer(_output2))

	txt = C.GoString(_output2)
	return txt
}
