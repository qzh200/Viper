package wmi

import (
	"fmt"
	"strconv"

	"viper/internal/com"
)

// 程序用到的 WMI 命名空间。
//
// 它们写在这里，而不是放在这个包的 Windows 部分，这样调用者不必知道自己
// 跑在哪个平台上。
const (
	// NamespaceACPI holds the ACPI WMI providers, including
	// AcerGamingFunction and APGeAction.
	NamespaceACPI = `root\WMI`

	// NamespaceCIM is the standard Windows CIM namespace.
	NamespaceCIM = `root\CIMV2`
)

// Kind describes the type of a WMI parameter value.
type Kind int

// The supported parameter kinds.
const (
	// KindUint64 is an unsigned 64 bit integer.
	KindUint64 Kind = iota
	// KindUint32 is an unsigned 32 bit integer.
	KindUint32
	// KindInt32 is a signed 32 bit integer.
	KindInt32
	// KindString is a string.
	KindString
	// KindBytes is a byte array, carried as an OLE SAFEARRAY.
	KindBytes
)

// Value is a WMI parameter value.
type Value struct {
	Kind  Kind
	U64   uint64
	Str   string
	Bytes []byte
}

// Uint64 returns an unsigned 64 bit integer value.
func Uint64(v uint64) Value { return Value{Kind: KindUint64, U64: v} }

// Uint32 returns a 32 bit unsigned integer value.
func Uint32(v uint32) Value { return Value{Kind: KindUint32, U64: uint64(v)} }

// Int32 returns a signed 32 bit integer value.
func Int32(v int32) Value { return Value{Kind: KindInt32, U64: uint64(uint32(v))} }

// String returns a string value.
func String(v string) Value { return Value{Kind: KindString, Str: v} }

// Bytes returns a byte array value.
func Bytes(v []byte) Value { return Value{Kind: KindBytes, Bytes: v} }

// String renders the value for display.
func (v Value) String() string {
	switch v.Kind {
	case KindString:
		return v.Str
	case KindBytes:
		return fmt.Sprintf("%d bytes", len(v.Bytes))
	default:
		return strconv.FormatUint(v.U64, 10)
	}
}

// UInt64 returns the value as an unsigned integer. Strings are parsed.
func (v Value) UInt64() uint64 {
	switch v.Kind {
	case KindString:
		n, _ := strconv.ParseUint(v.Str, 0, 64)
		return n
	case KindInt32:
		return uint64(uint32(v.U64))
	default:
		return v.U64
	}
}

// candidate is one attempt at writing a parameter.
type candidate struct {
	name  string
	build func() (com.VARIANT, error)
}

// valueCandidates lists the VARIANT representations to try for a value, in
// the order the Acer providers accept them.
//
// 提供程序把大部分参数声明成无符号的整数，可真正存下来的时候只收有符号
// 的整数或者字符串。
func valueCandidates(value Value) []candidate {
	if value.Kind == KindString {
		return []candidate{
			{"BSTR", func() (com.VARIANT, error) { return com.NewVariantBSTR(value.Str) }},
		}
	}

	out := make([]candidate, 0, 4)
	if value.U64 <= 0x7FFFFFFF {
		// 提供程序真正存下来的是有符号的 32 位整数。
		out = append(out, candidate{"I4", func() (com.VARIANT, error) {
			return com.NewVariantI4(int32(value.U64)), nil
		}})
	}
	out = append(out, candidate{"UI8", func() (com.VARIANT, error) {
		return com.NewVariantUI8(value.U64), nil
	}})
	out = append(out, candidate{"UI4", func() (com.VARIANT, error) {
		return com.NewVariantUI4(uint32(value.U64)), nil
	}})
	out = append(out, candidate{"BSTR", func() (com.VARIANT, error) {
		return com.NewVariantBSTR(strconv.FormatUint(value.U64, 10))
	}})
	return out
}

// conventionalOutputNames are the names of the output parameters of the
// WMI providers.
//
// Acer 的提供程序给 AcerGamingFunction 这个类用 gmOutput，给 APGeAction 用
// uiOutput，那些同时给出一个缓冲和一个状态字节的方法用 gmReturn 作为第二
// 个值。CPU 超频的方法给出 ReturnCode、ReturnOCProfile 和 OCStructure。
//
// 属性表在每个版本的 WMI 上都读不出来，所以这个名单是读输出参数的主路。
var conventionalOutputNames = []string{
	"gmOutput", "uiOutput", "gmReturn", "uiReturn",
	"ReturnValue", "Output", "Result",
	"ReturnCode", "ReturnOCProfile", "OCStructure",
}
