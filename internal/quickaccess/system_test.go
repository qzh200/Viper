package quickaccess

import (
	"testing"
)

// 本机原厂 Quick Access 服务支持的另外几项设置：透明效果、任务栏自动
// 隐藏、屏保与屏保时间。
//
// 服务自己报出来的形状是照着本机实测的应答写的：透明效果与任务栏自动
// 隐藏报的是数字（1 开、0 关），屏保报的是开关，屏保时间报的是秒数。

// TestTheTransparentEffect 透明效果读回来是个数字。
func TestTheTransparentEffect(t *testing.T) {
	cases := map[float64]bool{1: true, 0: false}

	for answer, want := range cases {
		server := newTestServer(t)
		server.answer(CommandTransparentEffect, answer)

		client, err := Connect(Options{Address: server.address()})
		if err != nil {
			t.Fatal(err)
		}

		value, err := client.TransparentEffect()
		if err != nil {
			t.Errorf("透明效果读取失败: %v", err)
		} else if value != want {
			t.Errorf("透明效果是 %v，答案是 %v，想要 %v", value, answer, want)
		}
		client.Close()
	}
}

// TestTheAutoHideTaskbar 任务栏自动隐藏读回来是个数字。
func TestTheAutoHideTaskbar(t *testing.T) {
	cases := map[float64]bool{1: true, 0: false}

	for answer, want := range cases {
		server := newTestServer(t)
		server.answer(CommandAutoHideTaskbar, answer)

		client, err := Connect(Options{Address: server.address()})
		if err != nil {
			t.Fatal(err)
		}

		value, err := client.AutoHideTaskbar()
		if err != nil {
			t.Errorf("任务栏自动隐藏读取失败: %v", err)
		} else if value != want {
			t.Errorf("任务栏自动隐藏是 %v，答案是 %v，想要 %v", value, answer, want)
		}
		client.Close()
	}
}

// TestTheScreenSaver 屏保读回来是个开关。
func TestTheScreenSaver(t *testing.T) {
	cases := map[bool]bool{true: true, false: false}

	for answer, want := range cases {
		server := newTestServer(t)
		server.answer(CommandScreenSaver, answer)

		client, err := Connect(Options{Address: server.address()})
		if err != nil {
			t.Fatal(err)
		}

		value, err := client.ScreenSaver()
		if err != nil {
			t.Errorf("屏保读取失败: %v", err)
		} else if value != want {
			t.Errorf("屏保是 %v，答案是 %v，想要 %v", value, answer, want)
		}
		client.Close()
	}
}

// TestTheScreenSaverTimeout 屏保时间读回来是秒数。
func TestTheScreenSaverTimeout(t *testing.T) {
	server := newTestServer(t)
	server.answer(CommandScreenSaverTimeout, 600)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	seconds, err := client.ScreenSaverTimeout()
	if err != nil {
		t.Fatal(err)
	}
	if seconds != 600 {
		t.Errorf("屏保时间是 %d 秒，想要 600 秒", seconds)
	}
}

// TestTheWriteOfTheNewCommands 写下去的值要跟服务自己报的形状一样。
func TestTheWriteOfTheNewCommands(t *testing.T) {
	server := newTestServer(t)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SetTransparentEffect(true); err != nil {
		t.Fatal(err)
	}
	if err := client.SetAutoHideTaskbar(true); err != nil {
		t.Fatal(err)
	}
	if err := client.SetScreenSaver(true); err != nil {
		t.Fatal(err)
	}
	if err := client.SetScreenSaverTimeout(600); err != nil {
		t.Fatal(err)
	}

	if len(server.requests) != 4 {
		t.Fatalf("服务收到了 %d 个请求，想要四个", len(server.requests))
	}

	want := []struct {
		command string
		value   any
	}{
		{CommandTransparentEffect, float64(1)},
		{CommandAutoHideTaskbar, float64(1)},
		{CommandScreenSaver, true},
		{CommandScreenSaverTimeout, float64(600)},
	}
	for i, c := range want {
		request := server.requests[i]
		if request["Command"] != c.command {
			t.Errorf("第 %d 个请求是关于 %v，想要 %s", i, request["Command"], c.command)
		}
		if request["Action"] != ActionSet {
			t.Errorf("第 %d 个请求的动作是 %v，想要 %s", i, request["Action"], ActionSet)
		}
		if request["Param1"] != c.value {
			t.Errorf("第 %d 个请求的值是 %v，想要 %v", i, request["Param1"], c.value)
		}
	}
}

// TestSetTheScreenSaverTimeoutOfANegativeTime 负的秒数在出门之前就被拒绝。
func TestSetTheScreenSaverTimeoutOfANegativeTime(t *testing.T) {
	server := newTestServer(t)

	client, err := Connect(Options{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SetScreenSaverTimeout(-1); err == nil {
		t.Error("一个负的屏保时间被接受了")
	}
	if len(server.requests) != 0 {
		t.Errorf("服务收到了 %d 个请求，想要一个都没有", len(server.requests))
	}
}

// TestSwitchValueOfTheKinds 服务有的报开关，有的报数字。
func TestSwitchValueOfTheKinds(t *testing.T) {
	cases := []struct {
		value any
		want  bool
	}{
		{true, true},
		{false, false},
		{float64(1), true},
		{float64(0), false},
		{int(1), true},
		{int(0), false},
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"on", true},
		{"off", false},
	}
	for _, c := range cases {
		got, err := switchValue(Result{Value: c.value})
		if err != nil {
			t.Errorf("值 %v 被拒绝了: %v", c.value, err)
			continue
		}
		if got != c.want {
			t.Errorf("值 %v 是 %v，想要 %v", c.value, got, c.want)
		}
	}
}

// TestSwitchValueOfSomethingElse 不是开关的值要被拒绝。
func TestSwitchValueOfSomethingElse(t *testing.T) {
	cases := []any{nil, []any{1}, map[string]any{}, "perhaps"}
	for _, value := range cases {
		if _, err := switchValue(Result{Value: value}); err == nil {
			t.Errorf("值 %v 被当成了开关", value)
		}
	}
}

// TestSwitchNumber 开关变成服务用的数字。
func TestSwitchNumber(t *testing.T) {
	if switchNumber(true) != 1 {
		t.Error("开着的开关不是 1")
	}
	if switchNumber(false) != 0 {
		t.Error("关着的开关不是 0")
	}
}
