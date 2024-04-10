package smwasm

import (
	"fmt"
	"os"
)

type SmItem interface {
	Call(input interface{}) interface{}
}

var gsn int = 0

var smMap = make(map[string]SmItem)

func readBinary(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	buffer := make([]byte, fileSize)

	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func LoadWasm(wasmPath string, maxPage int) {
	gsn++
	wi := WasmItem{sn: gsn, WasmPath: wasmPath, MaxPage: maxPage}
	wi.Load()
}

func Register(usage string, item SmItem) {
	smMap[usage] = item
}

func Call(usage string, input interface{}) interface{} {
	item := smMap[usage]
	if item == nil {
		var ret interface{}
		return ret
	}
	ret := item.Call(input)
	fmt.Println("--- smmo call --- {0} --- {1} --- {2} ---", usage, input, ret)
	return ret
}
