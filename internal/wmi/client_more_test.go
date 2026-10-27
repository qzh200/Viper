//go:build windows

package wmi

import (
	"sync"
	"testing"
)

// 每个客户端都有自己的线程和公寓。下面的测试检查客户端之间不会互相
// 打扰，以及关掉客户端的时候还有调用在跑也是安全的。

func TestTwoClientsAreIndependent(t *testing.T) {
	first, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("命名空间连不上: %v", err)
	}
	second, err := Connect(NamespaceCIM)
	if err != nil {
		first.Close()
		t.Skipf("命名空间连不上: %v", err)
	}

	// 关掉一个客户端不能把另一个客户端的公寓也关掉。
	first.Close()

	if _, err := second.Instances("Win32_Processor"); err != nil {
		t.Errorf("第二个客户端列举不了实例: %v", err)
	}
	second.Close()
}

func TestTheCallsWhileTheClientIsClosed(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("命名空间连不上: %v", err)
	}

	// 调用在客户端被关掉的时候还在跑。它们要么有回话，要么报告客户端已经
	// 关闭，但不能把程序带倒。
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 4; j++ {
				if _, err := client.Instances("Win32_Processor"); err != nil {
					continue
				}
			}
		}()
	}
	client.Close()
	group.Wait()

	// 客户端已经关闭，所以每一个调用都得被拒绝。
	if _, err := client.Instances("Win32_Processor"); err == nil {
		t.Error("关掉的客户端还在列举实例")
	}
	if _, err := client.ReadProperty("path", "Name"); err == nil {
		t.Error("关掉的客户端还在读属性")
	}
	if _, err := client.ReadBytes("path", "Name"); err == nil {
		t.Error("关掉的客户端还在读字节数组")
	}
	if _, err := client.Call("Win32_Processor", "GetOwner", "path", nil); err == nil {
		t.Error("关掉的客户端还在调方法")
	}
}

func TestTheClientIsClosedOnlyOnce(t *testing.T) {
	client, err := Connect(NamespaceCIM)
	if err != nil {
		t.Skipf("命名空间连不上: %v", err)
	}

	// 关一个已经关掉的客户端，不能把 COM 的对象再放一次。
	client.Close()
	client.Close()
	client.Close()
}
