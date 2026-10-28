package main

import (
	"bytes"
	"testing"
)

func capture(t *testing.T, fn func()) string {
	t.Helper()

	var buffer bytes.Buffer
	old := stdout
	stdout = &buffer
	defer func() { stdout = old }()

	fn()
	return buffer.String()
}

func TestShow(t *testing.T) {
	table := buildTable()

	out := capture(t, func() { show(table) })

	for _, want := range []string{
		"processor information, length 26",
		"max speed   0x10: 4000",
		"core count  0x14: 16",
		"memory device, length 30",
		"size        0x10: 16384",
	} {
		if !contains(out, want) {
			t.Errorf("输出里没有 %q:\n%s", want, out)
		}
	}
	if !contains(out, "0x00:") {
		t.Errorf("输出里没有字节:\n%s", out)
	}
}

func TestDumpOfNothing(t *testing.T) {
	out := capture(t, func() { dump(nil) })
	if out != "" {
		t.Errorf("不该有输出:\n%s", out)
	}
}

func buildTable() []byte {
	processor := make([]byte, 26)
	processor[0] = 4
	processor[1] = 26
	processor[0x10] = 0xA0
	processor[0x11] = 0x0F
	processor[0x14] = 16

	memory := make([]byte, 30)
	memory[0] = 17
	memory[1] = 30
	memory[0x10] = 0x00
	memory[0x11] = 0x40
	memory[0x15] = 0xE8
	memory[0x16] = 0x03
	memory[0x1B] = 0xD0
	memory[0x1C] = 0x09

	table := append([]byte{}, processor...)
	table = append(table, 0x00, 0x00)
	table = append(table, memory...)
	table = append(table, 0x00, 0x00)
	return table
}

func contains(text, name string) bool {
	for i := 0; i+len(name) <= len(text); i++ {
		if text[i:i+len(name)] == name {
			return true
		}
	}
	return false
}
