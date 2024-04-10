package smwasm

//
// go get -u github.com/bytecodealliance/wasmtime-go/v17@v17.0.0
//
// sudo apt install gcc
// sudo apt install g++
//

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v17"
)

type WasmItem struct {
	sn        int
	WasmPath  string
	MaxPage   int
	Smcall    *wasmtime.Func
	Smalloc   *wasmtime.Func
	Smdealloc *wasmtime.Func
	Store     *wasmtime.Store
	Mem       *wasmtime.Memory
}

func (w WasmItem) fd_write(d0, d1, d2, d3 int32) int32 {
	fmt.Println("--- fd_write[4] ---")
	return 0
}

func (w WasmItem) hostgetms() int64 {
	now := time.Now()
	milliseconds := now.UnixNano() / 1000000
	return milliseconds
}

func (w WasmItem) hostcallsm(ptr int32) int32 {
	txt := w.getBufferText(ptr)
	var result map[string]interface{}
	json.Unmarshal([]byte(txt), &result)

	key := result["$usage"].(string)
	ret := Call(key, result)
	bytes, _ := json.Marshal(ret)
	ptr_ret := w.setBufferText(bytes)
	return ptr_ret
}

func (w WasmItem) hostputmemory(ptr int32, ty int32) {
	if ty != 10 {
		return
	}
	var txt = w.getBufferText(ptr)
	fmt.Println(w.sn, txt)
}

func (w WasmItem) hostdebug(d1 int32, d2 int32) {
	fmt.Println(w.sn, "--- < < ---", d1, "---", d2, "---")
}

func (w WasmItem) getBufferText(ptr int32) string {
	wasm_u8a := w.Mem.UnsafeData(w.Store)
	var len int32 = int32(binary.LittleEndian.Uint32(wasm_u8a[ptr : ptr+4]))
	str := string(wasm_u8a[ptr+4 : ptr+4+len])
	return str
}

func (w WasmItem) setBufferText(u8a []byte) int32 {
	len := len(u8a)

	ret, _ := w.Smalloc.Call(w.Store, len)
	ptr := ret.(int32)

	wasm_u8a := w.Mem.UnsafeData(w.Store)
	dest := wasm_u8a[ptr+4 : int(ptr)+4+len]
	copy(dest, u8a)

	return ptr
}

func (w WasmItem) Load() {
	wasmBytes, _ := readBinary(w.WasmPath)
	fmt.Println("--- binary size ---", w.WasmPath, "---", len(wasmBytes))
	w.Store = wasmtime.NewStore(wasmtime.NewEngine())
	module, _ := wasmtime.NewModule(w.Store.Engine, wasmBytes)

	funcs := []wasmtime.AsExtern{}
	for _, imp := range module.Imports() {
		if *imp.Type() == *imp.Type().FuncType().AsExternType() {
			fn := *imp.Name()
			if strings.HasPrefix(fn, "__wbg_") {
				len_fn := len(fn)
				fn = fn[6 : len_fn-17]
			}

			if fn == "hostgetms" {
				funcs = append(funcs, wasmtime.WrapFunc(w.Store, func() int64 { return w.hostgetms() }))
			} else if fn == "hostputmemory" {
				funcs = append(funcs, wasmtime.WrapFunc(w.Store, func(ptr, ty int32) { w.hostputmemory(ptr, ty) }))
			} else if fn == "hostcallsm" {
				funcs = append(funcs, wasmtime.WrapFunc(w.Store, func(ptr int32) int32 { return w.hostcallsm(ptr) }))
			} else if fn == "hostdebug" {
				funcs = append(funcs, wasmtime.WrapFunc(w.Store, func(d1, d2 int32) { w.hostdebug(d1, d2) }))
			} else if fn == "fd_write" {
				funcs = append(funcs, wasmtime.WrapFunc(w.Store, w.fd_write))
			}
		}
	}

	inst, err := wasmtime.NewInstance(w.Store, module, funcs)
	check(err)

	w.Mem = inst.GetExport(w.Store, "memory").Memory()

	sminit := inst.GetFunc(w.Store, "sminit")
	fmt.Println("--- app sminit[0] ---")
	sminit.Call(w.Store, 0)

	w.Smalloc = inst.GetFunc(w.Store, "smalloc")
	w.Smdealloc = inst.GetFunc(w.Store, "smdealloc")
	w.Smcall = inst.GetFunc(w.Store, "smcall")

	txt := `{"$usage": "smker.get.all"}`
	alltxt := w.callWasm([]byte(txt))

	var result map[string]interface{}
	json.Unmarshal([]byte(alltxt), &result)

	for key, _ := range result {
		if !strings.HasPrefix(key, "smker.") {
			fmt.Println("--- key ---", key)
			Register(key, w)
		}
	}
	fmt.Println("--- all text ---", result)

}

func (w WasmItem) Call(input interface{}) interface{} {
	bytes, _ := json.Marshal(input)
	ptr := w.setBufferText(bytes)

	ret, _ := w.Smcall.Call(w.Store, ptr, 1)
	ptr_ret := ret.(int32)

	txt := w.getBufferText(ptr_ret)
	var result map[string]interface{}
	json.Unmarshal([]byte(txt), &result)
	return result
}

func (w WasmItem) callWasm(u8a []byte) string {
	ptr := w.setBufferText(u8a)

	ret, _ := w.Smcall.Call(w.Store, ptr, 1)
	ptr_ret := ret.(int32)
	txt := w.getBufferText(ptr_ret)

	w.Smdealloc.Call(w.Store, ptr_ret)
	return txt
}
