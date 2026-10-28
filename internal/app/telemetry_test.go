package app

import (
	"testing"
)

func TestTheTelemetryOfTheMachine(t *testing.T) {
	application, _ := newTestAppWith(t)

	telemetry, err := application.Telemetry()
	if err != nil {
		t.Fatalf("读取遥测失败: %v", err)
	}

	if telemetry.Monitor.CPUUsagePercent != 12.5 {
		t.Errorf("处理器使用率是 %v，不是 12.5", telemetry.Monitor.CPUUsagePercent)
	}
	if telemetry.Monitor.CPUFrequencyMHz != 3400 {
		t.Errorf("处理器频率是 %d，不是 3400", telemetry.Monitor.CPUFrequencyMHz)
	}
	if telemetry.Monitor.GPUUsagePercent != 39 {
		t.Errorf("显卡使用率是 %d，不是 39", telemetry.Monitor.GPUUsagePercent)
	}
	if telemetry.Monitor.GPUFrequencyMHz != 420 {
		t.Errorf("显卡频率是 %d，不是 420", telemetry.Monitor.GPUFrequencyMHz)
	}
	if telemetry.Monitor.RAMUsagePercent != 42.8 {
		t.Errorf("内存占用是 %v，不是 42.8", telemetry.Monitor.RAMUsagePercent)
	}
	if telemetry.Monitor.RAMFrequencyMHz != 5600 {
		t.Errorf("内存频率是 %d，不是 5600", telemetry.Monitor.RAMFrequencyMHz)
	}

	for _, card := range telemetry.Cards {
		if card.UsagePercent < 0 || card.UsagePercent > 100 {
			t.Errorf("显卡报的负载是 %d%%", card.UsagePercent)
		}
	}
}

func TestTheTelemetryOfAMachineThatIsNotThere(t *testing.T) {
	application := newTestAppWithoutMachine(t)

	if _, err := application.Telemetry(); err == nil {
		t.Fatal("机器不在的时候该报错")
	}
	if _, err := application.Monitor(); err == nil {
		t.Fatal("机器不在的时候该报错")
	}
}
