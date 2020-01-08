package wasm

import (
	"encoding/binary"
	"errors"
	"unicode"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

func allocateInner(instance wasm.Instance, data []byte) (int64, error) {
	sz := len(data)
	res, err := instance.Exports["__allocate"](sz)
	if err != nil {
		return 0, err
	}
	ptr := res.ToI32()
	mem := instance.Memory.Data()[ptr:]
	if len(mem) < sz {
		return 0, errors.New("allocateInner: invalid memory size")
	}
	copy(mem, data)
	return int64(sz<<32) | int64(ptr), nil
}

func allocate(instance wasm.Instance, data [][]byte) (int64, error) {
	sz := len(data)
	res, err := instance.Exports["__allocate"](8 * sz)
	if err != nil {
		return 0, err
	}
	ptr := res.ToI32()
	mem := instance.Memory.Data()[ptr:]
	if len(mem) < 8*sz {
		return 0, errors.New("allocate: invalid memory size")
	}
	for idx, each := range data {
		loc, err := allocateInner(instance, each)
		if err != nil {
			return 0, err
		}
		binary.LittleEndian.PutUint64(mem[8*idx:8*idx+8], uint64(loc))
	}
	return int64(sz<<32) | int64(ptr), nil
}

func parseOutput(instance wasm.Instance, ptr int64) ([]byte, error) {
	sz, pointer := int(ptr>>32), (ptr & ((1 << 32) - 1))
	mem := instance.Memory.Data()[pointer:]
	if len(mem) < sz {
		return nil, errors.New("parseOutput: invalid memory size")
	}
	res := make([]byte, sz)
	for idx := 0; idx < sz; idx++ {
		res[idx] = mem[idx]
	}
	return res, nil
}

func storeParams(instance wasm.Instance, params []byte) (int64, error) {
	return allocateInner(instance, params)
}

func Name(code []byte) (string, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return "", err
	}
	defer instance.Close()
	fn := instance.Exports["__name"]
	if fn == nil {
		return "", errors.New("__name not implemented")
	}
	ptr, err := fn()
	if err != nil {
		return "", err
	}
	rawResult, err := parseOutput(instance, ptr.ToI64())
	if err != nil {
		return "", err
	}
	for _, ch := range string(rawResult) {
		if !unicode.IsPrint(ch) {
			return "", errors.New("Invalid name character")
		}
	}
	return string(rawResult), nil
}

func ParamsInfo(code []byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	fn := instance.Exports["__params_info"]
	if fn == nil {
		return nil, errors.New("__params_info not implemented")
	}
	ptr, err := fn()
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func ParseParams(code []byte, params []byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	paramsInput, err := storeParams(instance, params)
	if err != nil {
		return nil, err
	}
	fn := instance.Exports["__parse_params"]
	if fn == nil {
		return nil, errors.New("__parse_params not implemented")
	}
	ptr, err := fn(paramsInput)
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func RawDataInfo(code []byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	fn := instance.Exports["__raw_data_info"]
	if fn == nil {
		return nil, errors.New("__raw_data_info not implemented")
	}
	ptr, err := fn()
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func ParseRawData(code []byte, params []byte, data []byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	paramsInput, err := storeParams(instance, params)
	dataInput, err := allocateInner(instance, data)
	if err != nil {
		return nil, err
	}
	fn := instance.Exports["__parse_raw_data"]
	if fn == nil {
		return nil, errors.New("__parse_raw_data not implemented")
	}
	ptr, err := fn(paramsInput, dataInput)
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func Prepare(code []byte, params []byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	paramsInput, err := storeParams(instance, params)
	if err != nil {
		return nil, err
	}
	fn := instance.Exports["__prepare"]
	if fn == nil {
		return nil, errors.New("__prepare not implemented")
	}
	ptr, err := fn(paramsInput)
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func Execute(code []byte, params []byte, inputs [][]byte) ([]byte, error) {
	instance, err := wasm.NewInstance(code)
	if err != nil {
		return nil, err
	}
	defer instance.Close()
	paramsInput, err := storeParams(instance, params)
	if err != nil {
		return nil, err
	}
	wasmInput, err := allocate(instance, inputs)
	if err != nil {
		return nil, err
	}
	fn := instance.Exports["__execute"]
	if fn == nil {
		return nil, errors.New("__execute not implemented")
	}
	ptr, err := fn(paramsInput, wasmInput)
	if err != nil {
		return nil, err
	}
	return parseOutput(instance, ptr.ToI64())
}

func ReadBytes(filename string) ([]byte, error) {
	return wasm.ReadBytes(filename)
}
