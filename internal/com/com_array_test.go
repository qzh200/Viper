//go:build windows

package com

import (
	"testing"
)

// 厂商的提供程序把键盘背光的缓冲区交成一个字节的数组，所以数组这条路得
// 能把每一个字节带回来。

func TestTheArrayOfBytesInAVariant(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	variant, err := NewVariantUI1Array(data)
	if err != nil {
		t.Fatal(err)
	}
	defer variant.Clear()

	if variant.Type() != VTArray|VTUI1 {
		t.Errorf("这个 variant 的类型是 0x%04X, 要数组的类型", variant.Type())
	}

	read, err := variant.AsBytes()
	if err != nil {
		t.Fatalf("字节读不出来: %v", err)
	}
	if len(read) != len(data) {
		t.Fatalf("这个 variant 里有 %d 个字节, 要 %d 个", len(read), len(data))
	}
	for i := range data {
		if read[i] != data[i] {
			t.Errorf("第 %d 个字节是 %d, 要 %d", i, read[i], data[i])
		}
	}
}

func TestTheArrayOfBytesOfNothing(t *testing.T) {
	var variant VARIANT
	if _, err := variant.AsBytes(); err == nil {
		t.Error("什么都没有被当成了字节数组")
	}
}
