package quickaccess

import (
	"testing"
)

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

	if len(server.receivedRequests()) != 4 {
		t.Fatalf("服务收到了 %d 个请求，想要四个", len(server.receivedRequests()))
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
		request := server.receivedRequests()[i]
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
	if len(server.receivedRequests()) != 0 {
		t.Errorf("服务收到了 %d 个请求，想要一个都没有", len(server.receivedRequests()))
	}
}

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

func TestSwitchValueOfSomethingElse(t *testing.T) {
	cases := []any{nil, []any{1}, map[string]any{}, "perhaps"}
	for _, value := range cases {
		if _, err := switchValue(Result{Value: value}); err == nil {
			t.Errorf("值 %v 被当成了开关", value)
		}
	}
}

func TestSwitchNumber(t *testing.T) {
	if switchNumber(true) != 1 {
		t.Error("开着的开关不是 1")
	}
	if switchNumber(false) != 0 {
		t.Error("关着的开关不是 0")
	}
}
