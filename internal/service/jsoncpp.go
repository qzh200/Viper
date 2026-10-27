// Package service implements the PredatorSense service.
//
// The official application talks to a service of the original software on
// a local socket. Viper can offer the same service, so that the official
// application talks to Viper instead.
//
// The protocol is simple. A request is the word "ACER", the identifier of
// the packet as a little endian word and a JSON document. The answer is a
// JSON document on its own.
package service

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// The packet identifiers of the protocol.
const (
	// PacketInitialization asks for the version and for the abilities of
	// the machine.
	PacketInitialization uint32 = 0
	// PacketGetMonitorData asks for the live numbers.
	PacketGetMonitorData uint32 = 10
	// PacketGetUpdatedData asks for the value of one setting.
	PacketGetUpdatedData uint32 = 20
	// PacketGetConnectedDevice asks for the devices of the machine.
	PacketGetConnectedDevice uint32 = 30
	// PacketGetLightingEffect asks for the lighting patterns.
	PacketGetLightingEffect uint32 = 40
	// PacketSetDeviceData writes one setting.
	PacketSetDeviceData uint32 = 100
)

// writeJSON renders a value the way the original service does.
//
// The original service uses the styled writer of a common C++ library: the
// members of an object come in the order of their bytes, the indent is a
// tab, and a number with a fraction always shows one.
func writeJSON(value any) string {
	var out strings.Builder
	writeValue(&out, value, 0)
	out.WriteByte('\n')
	return out.String()
}

// writeValue renders one value.
func writeValue(out *strings.Builder, value any, depth int) {
	switch typed := value.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		if typed {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case string:
		writeString(out, typed)
	case int:
		out.WriteString(strconv.Itoa(typed))
	case int32:
		out.WriteString(strconv.FormatInt(int64(typed), 10))
	case int64:
		out.WriteString(strconv.FormatInt(typed, 10))
	case uint32:
		out.WriteString(strconv.FormatUint(uint64(typed), 10))
	case uint64:
		out.WriteString(strconv.FormatUint(typed, 10))
	case float64:
		out.WriteString(writeDouble(typed))
	case float32:
		out.WriteString(writeDouble(float64(typed)))
	case []any:
		writeArray(out, typed, depth)
	case map[string]any:
		writeObject(out, typed, depth)
	default:
		// Anything else is written as a text, the values of the protocol
		// are known.
		writeString(out, fmt.Sprintf("%v", typed))
	}
}

// writeObject renders an object.
func writeObject(out *strings.Builder, value map[string]any, depth int) {
	if len(value) == 0 {
		out.WriteString("{}")
		return
	}

	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out.WriteString("{\n")
	for i, key := range keys {
		writeIndent(out, depth+1)
		writeString(out, key)
		out.WriteString(" : ")
		// A member that holds an object or an array is written on the
		// next line, the original service does the same.
		if isComposite(value[key]) {
			out.WriteByte('\n')
			writeIndent(out, depth+1)
		}
		writeValue(out, value[key], depth+1)
		if i < len(keys)-1 {
			out.WriteByte(',')
		}
		out.WriteByte('\n')
	}
	writeIndent(out, depth)
	out.WriteByte('}')
}

// isComposite reports whether a value is written over several lines.
func isComposite(value any) bool {
	switch typed := value.(type) {
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}

// writeArray renders an array.
func writeArray(out *strings.Builder, value []any, depth int) {
	if len(value) == 0 {
		out.WriteString("[]")
		return
	}

	out.WriteString("[\n")
	for i, item := range value {
		writeIndent(out, depth+1)
		writeValue(out, item, depth+1)
		if i < len(value)-1 {
			out.WriteByte(',')
		}
		out.WriteByte('\n')
	}
	writeIndent(out, depth)
	out.WriteByte(']')
}

// writeIndent writes the indent of a level.
func writeIndent(out *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		out.WriteByte('\t')
	}
}

// writeString renders a text.
func writeString(out *strings.Builder, value string) {
	out.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			out.WriteString("\\\"")
		case '\\':
			out.WriteString("\\\\")
		case '\b':
			out.WriteString("\\b")
		case '\f':
			out.WriteString("\\f")
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			if r < 0x20 {
				out.WriteString(fmt.Sprintf("\\u%04x", r))
				continue
			}
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
}

// writeDouble renders a number with a fraction.
//
// The original service prints the number with seventeen significant digits
// and makes sure that a whole number still shows its fraction.
func writeDouble(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "null"
	}

	text := strconv.FormatFloat(value, 'g', 17, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}
