//go:build windows

package wmi

import (
	"testing"
)

// 键盘背光的缓冲区是作为一个字节数组读回来的，所以数组这条路得管用。
//
// 标准命名空间里的属性都是文本和数字，所以数组本身在 variant 这一层上查。
// 读一个带着数组的属性走的是同一段代码。

func TestTheBytesOfAPropertyThatIsNotAnArray(t *testing.T) {
	client := connection(t)
	paths := processorPaths(t)

	// 处理器的名字是文本，不是字节数组，所以客户端得说出来。
	if _, err := client.ReadBytes(paths[0], "Manufacturer"); err == nil {
		t.Error("一段文本被当成了字节数组")
	}
}

func TestTheBytesOfAPropertyThatIsNotThere(t *testing.T) {
	client := connection(t)
	paths := processorPaths(t)

	if _, err := client.ReadBytes(paths[0], "NoSuchProperty"); err == nil {
		t.Error("一个不存在的属性被读出来了")
	}
}
