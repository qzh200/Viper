//go:build windows

package wmi

import (
	"fmt"
	"unsafe"

	"viper/internal/com"
)

// slotGetNames is IWbemClassObject::GetNames.
//
// 槽位是在跑程序的这台机器上量出来的，但 WMI 接口的布局在不同版本的
// Windows 上不一样：这个调用并不总是在每个机器上都返回一个字符串的数组。
// 所以它只是一条补充的路，读输出参数还是靠提供程序的名字。
const slotGetNames = 34

// failed reports whether a raw HRESULT indicates an error.
func failed(hr uintptr) bool { return int32(hr) < 0 }

// hresult renders a raw HRESULT for error messages.
func hresult(hr uintptr) error { return fmt.Errorf("HRESULT 0x%08X", uint32(hr)) }

// putValue writes an input parameter onto an in-parameters object.
//
// Acer 的提供程序把大部分参数声明成无符号的整数，可真正存下来的时候只收
// 有符号的整数或者字符串。所以先试首选的类型，再试别的。
func putValue(target com.COMObject, name string, value Value) error {
	key, err := com.NewBSTR(name)
	if err != nil {
		return err
	}
	defer key.Free()

	if value.Kind == KindBytes {
		return putBytes(target, key, value.Bytes)
	}

	var lastErr error
	for _, candidate := range valueCandidates(value) {
		variant, err := candidate.build()
		if err != nil {
			lastErr = err
			continue
		}
		r, _ := target.Call(slotPut,
			uintptr(key),
			0, // lFlags
			uintptr(unsafe.Pointer(&variant)),
			0, // lType
		)
		variant.Clear()
		if failed(r) {
			lastErr = fmt.Errorf("wmi: Put(%s) 用 %s 的写法失败: %w", name, candidate.name, hresult(r))
			continue
		}
		return nil
	}
	return fmt.Errorf("wmi: Put(%s): 没有一种写法被接受（%v）", name, lastErr)
}

// putBytes writes a byte array parameter using an OLE SAFEARRAY.
//
// SetGamingKBBacklight 和 SetGamingLED 的输入是一个字节的数组，而不是一个
// 整数。
func putBytes(target com.COMObject, key com.BSTR, data []byte) error {
	variant, err := com.NewVariantUI1Array(data)
	if err != nil {
		return err
	}
	defer variant.Clear()

	r, _ := target.Call(slotPut,
		uintptr(key),
		0, // lFlags
		uintptr(unsafe.Pointer(&variant)),
		0, // lType
	)
	if failed(r) {
		return fmt.Errorf("wmi: Put(bytes) 失败: %w", hresult(r))
	}
	return nil
}

// readValue reads a named property from a class object.
func readValue(obj com.COMObject, name string) (Value, error) {
	key, err := com.NewBSTR(name)
	if err != nil {
		return Value{}, err
	}
	defer key.Free()

	var v com.VARIANT
	v.Init()
	defer v.Clear()

	var flavor int32
	r, _ := obj.Call(slotGet,
		uintptr(key),
		0, // lFlags
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&flavor)),
		0, // ppContext
	)
	if failed(r) {
		return Value{}, fmt.Errorf("wmi: Get(%s) 失败: %w", name, hresult(r))
	}

	switch v.Type() {
	case com.VTBSTR:
		s, err := v.AsString()
		if err != nil {
			return Value{}, err
		}
		return String(s), nil
	case com.VTArray | com.VTUI1:
		b, err := v.AsBytes()
		if err != nil {
			return Value{}, err
		}
		return Bytes(b), nil
	default:
		n, err := v.AsUInt64()
		if err != nil {
			return Value{}, err
		}
		return Uint64(n), nil
	}
}

// propertyNames lists the properties of a class object.
//
// 这个调用只是一条补充的路，而且要对任何结果都有准备：GetNames 的槽位在
// 每个版本的 WMI 上都不是预期的那个，所以这个调用可能报失败，也可能返回
// 一个不是字符串数组的东西。所以数组的地址在读之前会被检查，属性表读不出来
// 的时候，调用者就退回提供程序的名字。
func propertyNames(obj com.COMObject) ([]string, error) {
	if obj == 0 {
		return nil, fmt.Errorf("wmi: GetNames: 空的对象")
	}
	var names, quals uintptr
	r, _ := obj.Call(slotGetNames,
		0, // lFlags
		uintptr(unsafe.Pointer(&names)),
		uintptr(unsafe.Pointer(&quals)),
	)
	if failed(r) {
		return nil, fmt.Errorf("wmi: GetNames 失败: %w", hresult(r))
	}
	if names == 0 {
		return nil, fmt.Errorf("wmi: GetNames: 没有拿到名字")
	}
	// 这里用不上限定符，但那个数组属于调用者，得放掉。
	com.DestroySafeArray(quals)
	return com.SafeArrayBSTRs(names)
}
