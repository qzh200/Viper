package quickaccess

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestLiveProbeOfTheCommands(t *testing.T) {
	if os.Getenv("VIPER_LIVE_QUICKACCESS") == "" {
		t.Skip("设置 VIPER_LIVE_QUICKACCESS=1 之后才会连本机的原厂服务")
	}

	client, err := Connect(Options{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("原厂服务连不上: %v", err)
	}
	defer client.Close()

	commands := []string{
		"USBCharge",
		"USBChargeSwitch",
		"SystemUsageOSD",
		"DarkMode",
		"TransparentEffect",
		"AutoHideTaskbar",
		"ScreenSaver",
		"ScreenSaverTimeout",
		"HighlightFeature",
		"HDR",
		"DisplayModeSwitch",
	}

	for _, command := range commands {
		raw, err := client.probe(command)
		if err != nil {
			t.Logf("%-20s 读取失败: %v", command, err)
			continue
		}
		t.Logf("%-20s %s", command, raw)
	}
}

func TestLiveProbeOfTheMethods(t *testing.T) {
	if os.Getenv("VIPER_LIVE_QUICKACCESS") == "" {
		t.Skip("设置 VIPER_LIVE_QUICKACCESS=1 之后才会连本机的原厂服务")
	}

	client, err := Connect(Options{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("原厂服务连不上: %v", err)
	}
	defer client.Close()

	if value, err := client.TransparentEffect(); err != nil {
		t.Errorf("透明效果读取失败: %v", err)
	} else {
		t.Logf("TransparentEffect   = %v", value)
	}
	if value, err := client.AutoHideTaskbar(); err != nil {
		t.Errorf("任务栏自动隐藏读取失败: %v", err)
	} else {
		t.Logf("AutoHideTaskbar     = %v", value)
	}
	if value, err := client.ScreenSaver(); err != nil {
		t.Errorf("屏保读取失败: %v", err)
	} else {
		t.Logf("ScreenSaver         = %v", value)
	}
	if value, err := client.ScreenSaverTimeout(); err != nil {
		t.Errorf("屏保时间读取失败: %v", err)
	} else {
		t.Logf("ScreenSaverTimeout  = %v", value)
	}
	if value, err := client.DarkMode(); err != nil {
		t.Errorf("暗黑模式读取失败: %v", err)
	} else {
		t.Logf("DarkMode            = %v", value)
	}
	if value, err := client.DisplayMode(); err != nil {
		t.Errorf("显示模式读取失败: %v", err)
	} else {
		t.Logf("DisplayMode         = %v", value)
	}
	if value, err := client.USBChargeEnabled(); err != nil {
		t.Errorf("USB 充电开关读取失败: %v", err)
	} else {
		t.Logf("USBChargeEnabled    = %v", value)
	}
	if value, err := client.USBChargeLimit(); err != nil {
		t.Errorf("USB 充电阈值读取失败: %v", err)
	} else {
		t.Logf("USBChargeLimit      = %v", value)
	}
}

func (c *Client) probe(command string) (string, error) {
	err := c.writeJSON(map[string]any{
		"PacketType": packetRequest,
		"Version":    1,
		"Session":    sessionID(),
		"Command":    command,
		"Action":     ActionGet,
	})
	if err != nil {
		return "", err
	}

	c.conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	answer, err := c.readJSON()
	if err != nil {
		return "", err
	}
	text, err := json.Marshal(answer)
	if err != nil {
		return "", err
	}
	return string(text), nil
}
