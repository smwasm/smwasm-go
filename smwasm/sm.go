package smwasm

/*
#cgo LDFLAGS: -L./rustlib/target/release -lsmwasmgo_rustlib
#include <stdlib.h>
typedef char *(*GoCallNativeFunc)(const char *szInput);
extern void smwasm_set_above(GoCallNativeFunc);
extern char* callNative(char*);
*/
import "C"
import (
	"encoding/json"
	"sync"
)

type GoCallNativeFuncWrapper C.GoCallNativeFunc

type NativeFunc func(string) string

var nativeFuncMap map[string]NativeFunc

var once sync.Once

func prepareSmartModule() {
	once.Do(func() {
		nativeFuncMap = make(map[string]NativeFunc)

		C.smwasm_set_above(GoCallNativeFuncWrapper(C.callNative))
	})
}

func getItem(itdef string, item string) string {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(itdef), &data)
	if err != nil {
		return ""
	}
	v, exists := data[item]
	if !exists {
		return ""
	}
	value, ok := v.(string)
	if !ok {
		return ""
	}

	return value
}

//export callNative
func callNative(_inputText *C.char) *C.char {
	inputText := C.GoString(_inputText)
	name := getItem(inputText, "$usage")
	if name == "" {
		return C.CString("")
	}

	fnc, exists := nativeFuncMap[name]
	if !exists {
		return C.CString("")
	}

	result := fnc(inputText)
	cRet := C.CString(result)
	return cRet
}
