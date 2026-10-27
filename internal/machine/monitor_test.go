package machine

import (
	"errors"
	"testing"

	"viper/internal/agentservice"
)

// TestTheReadingsOfTheMachine 监视数据里的每一项都要读出来。
//
// 读数是原厂服务给的：处理器的使用率与频率、每核频率、显卡的使用率与频率、
// 内存的占用与频率、网络的上下行。
func TestTheReadingsOfTheMachine(t *testing.T) {
	service := newFakeService()
	service.monitor = answer(map[string]any{
		"CPU_TEMPERATURE":    90,
		"CPU_FANSPEED":       2722,
		"CPU_USAGE":          13.9,
		"CPU_FREQUENCY":      3506.5,
		"CPU_MAX_FREQUENCY":  5000,
		"CPU_CORE_FREQUENCY": []any{3666.9, 2786.4, 3767.8},
		"SYS1_TEMPERATURE":   63,
		"GPU1_TEMPERATURE":   58,
		"GPU1_FANSPEED":      2705,
		"GPU1_USAGE":         39,
		"GPU1_FREQUENCY":     420,
		"GPU1_MAX_FREQUENCY": 3105,
		"GPU1_IDLE_STATUS":   1,
		"RAM_TOTAL":          32487.86,
		"RAM_USAGE":          42.8,
		"RAM_FREQUENCY":      5600,
		"WIFI_DOWNLOAD":      94.7,
		"WIFI_UPLOAD":        220.5,
		"ETHERNET_DOWNLOAD":  1.5,
		"ETHERNET_UPLOAD":    2.5,
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	data, err := backend.Monitor()
	if err != nil {
		t.Fatalf("读取读数失败: %v", err)
	}

	if data.CPUTemperatureC != 90 {
		t.Errorf("处理器温度是 %d，不是 90", data.CPUTemperatureC)
	}
	if data.CPUFanRPM != 2722 {
		t.Errorf("处理器风扇转速是 %d，不是 2722", data.CPUFanRPM)
	}
	if data.CPUUsagePercent < 13 || data.CPUUsagePercent > 15 {
		t.Errorf("处理器使用率是 %v，不是 13.9", data.CPUUsagePercent)
	}
	if data.CPUFrequencyMHz != 3506 {
		t.Errorf("处理器频率是 %d，不是 3506", data.CPUFrequencyMHz)
	}
	if data.CPUMaxFrequencyMHz != 5000 {
		t.Errorf("处理器睿频上限是 %d，不是 5000", data.CPUMaxFrequencyMHz)
	}
	if len(data.CPUCoreFrequencyMHz) != 3 {
		t.Fatalf("每核频率有 %d 个，不是 3 个", len(data.CPUCoreFrequencyMHz))
	}
	if data.CPUCoreFrequencyMHz[0] != 3666 {
		t.Errorf("第一核的频率是 %d，不是 3666", data.CPUCoreFrequencyMHz[0])
	}
	if data.SystemTempC != 63 {
		t.Errorf("主板温度是 %d，不是 63", data.SystemTempC)
	}
	if data.GPUTemperatureC != 58 {
		t.Errorf("显卡温度是 %d，不是 58", data.GPUTemperatureC)
	}
	if data.GPUFanRPM != 2705 {
		t.Errorf("显卡风扇转速是 %d，不是 2705", data.GPUFanRPM)
	}
	if data.GPUUsagePercent != 39 {
		t.Errorf("显卡使用率是 %d，不是 39", data.GPUUsagePercent)
	}
	if data.GPUFrequencyMHz != 420 {
		t.Errorf("显卡频率是 %d，不是 420", data.GPUFrequencyMHz)
	}
	if data.GPUMaxFrequencyMHz != 3105 {
		t.Errorf("显卡最高频率是 %d，不是 3105", data.GPUMaxFrequencyMHz)
	}
	if !data.GPUIdle {
		t.Error("显卡报的是空闲")
	}
	if data.RAMTotalMB < 32487 || data.RAMTotalMB > 32489 {
		t.Errorf("内存总量是 %v，不是 32487", data.RAMTotalMB)
	}
	if data.RAMUsagePercent < 42 || data.RAMUsagePercent > 43 {
		t.Errorf("内存占用是 %v，不是 42.8", data.RAMUsagePercent)
	}
	if data.RAMFrequencyMHz != 5600 {
		t.Errorf("内存频率是 %d，不是 5600", data.RAMFrequencyMHz)
	}
	if data.WIFIDownloadKbps < 94 || data.WIFIDownloadKbps > 95 {
		t.Errorf("无线下行是 %v，不是 94.7", data.WIFIDownloadKbps)
	}
	if data.WIFIUploadKbps < 220 || data.WIFIUploadKbps > 221 {
		t.Errorf("无线上行是 %v，不是 220.5", data.WIFIUploadKbps)
	}
	if data.EthernetDownloadKbps != 1.5 {
		t.Errorf("有线下行是 %v，不是 1.5", data.EthernetDownloadKbps)
	}
	if data.EthernetUploadKbps != 2.5 {
		t.Errorf("有线上行是 %v，不是 2.5", data.EthernetUploadKbps)
	}
}

// TestTheSensorsComeFromTheReadings 传感器是监视数据里的一部分。
func TestTheSensorsComeFromTheReadings(t *testing.T) {
	service := newFakeService()
	service.monitor = answer(map[string]any{
		"CPU_TEMPERATURE":  91,
		"CPU_FANSPEED":     2722,
		"SYS1_TEMPERATURE": 63,
		"GPU1_TEMPERATURE": 58,
		"GPU1_FANSPEED":    2705,
	}, agentservice.ResultOK)

	backend := newBackend(t, service)

	sensors, err := backend.Sensors()
	if err != nil {
		t.Fatalf("读取传感器失败: %v", err)
	}
	if sensors.CPUTempC != 91 || sensors.CPUFanRPM != 2722 {
		t.Errorf("传感器是 %+v，不是温度 91 转速 2722", sensors)
	}
	if sensors.GPUTempC != 58 || sensors.GPUFanRPM != 2705 {
		t.Errorf("传感器是 %+v，不是显卡温度 58 转速 2705", sensors)
	}
	if sensors.SystemTempC != 63 {
		t.Errorf("主板温度是 %d，不是 63", sensors.SystemTempC)
	}
}

// TestTheReadingsThatCannotBeRead 读不到的时候要报错。
func TestTheReadingsThatCannotBeRead(t *testing.T) {
	service := newFakeService()
	service.getErr = errors.New("服务不可达")
	backend := newBackend(t, service)

	if _, err := backend.Monitor(); err == nil {
		t.Fatal("读不到读数的时候该报错")
	}
	if _, err := backend.Sensors(); err == nil {
		t.Fatal("读不到读数的时候该报错")
	}
}
