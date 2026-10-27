//go:build windows

package com

import (
	"runtime"
	"testing"
)

// COM 的公寓是按线程的，所以要试几种公寓的测试得待在同一个线程上。

func TestInitCOMAndUninitCOMArePaired(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	owned, err := InitCOMOwned()
	if err != nil {
		t.Fatalf("COM 的运行时起不来: %v", err)
	}
	if !owned {
		t.Error("这个公寓不是这次调用开出来的")
	}
	UninitCOM()

	// 线程空出来了，所以还能再开一个公寓。
	owned, err = InitCOMOwned()
	if err != nil {
		t.Fatalf("COM 的运行时第二次起不来: %v", err)
	}
	if !owned {
		t.Error("第二个公寓不是这次调用开出来的")
	}
	UninitCOM()
}

func TestTheApartmentOfAnotherKindIsAccepted(t *testing.T) {
	// 线程上已经跑着单线程公寓，所以第二次调用改不了公寓的种类。但那个
	// 公寓还能用，所以这个调用照样得被接受。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := InitCOMApartment(ApartmentSingleThreaded); err != nil {
		t.Fatalf("公寓起不来: %v", err)
	}
	defer UninitCOM()

	owned, err := InitCOMOwned()
	if err != nil {
		t.Fatalf("另一种公寓被拒绝了: %v", err)
	}
	if owned {
		t.Error("另一种公寓被当成了新开的公寓")
	}
}

func TestTheSecondCallOnTheSameApartmentIsAccepted(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := InitCOM(); err != nil {
		t.Fatalf("公寓起不来: %v", err)
	}
	defer UninitCOM()

	// 线程上已经有同一种公寓，这个调用会回答它已经在那里，不会再开一个。
	owned, err := InitCOMOwned()
	if err != nil {
		t.Fatalf("公寓被拒绝了: %v", err)
	}
	if owned {
		t.Error("同一个线程上开了第二个公寓")
	}
}

func TestAPointerThatIsNotAnArrayIsRefused(t *testing.T) {
	// 一次形状不对的调用可能往输出参数里塞一个不是数组的东西。读它不能
	// 把程序带倒。
	for _, pointer := range []uintptr{0x8, 0x1000, 0xFFFFFFFFFFFF} {
		if _, err := SafeArrayBytes(pointer); err == nil {
			t.Errorf("地址 %#x 被当成了字节数组", pointer)
		}
		if _, err := SafeArrayBSTRs(pointer); err == nil {
			t.Errorf("地址 %#x 被当成了字符串数组", pointer)
		}
	}
}

func TestTheDestroyOfAnArrayTwiceIsHarmless(t *testing.T) {
	array, err := NewSafeArrayUI1([]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	array.Destroy()
	// 数组已经没了，第二次调用不能再放它一次。
	array.Destroy()
	if !array.IsNil() {
		t.Error("数组上还留着地址")
	}
}
