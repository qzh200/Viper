package hardware

// MonitorData is what the machine reports about itself.
//
// 读数来自原厂服务的监视数据：处理器的温度、使用率、频率与每核频率，显卡的
// 温度、使用率与频率，内存的占用与频率，还有网络的上下行。
//
// 单位：温度是度，占用是百分比，频率是 MHz，内存总量是 MB——这些都核对过。
// 网络那两个数是 kbps（官方软件用的就是这个单位）：实测下载时网卡走 17.9
// KB/s = 143 kbps，服务报的是 121.5，对得上。
type MonitorData struct {
	// CPUTemperatureC is the temperature of the processor in degrees.
	CPUTemperatureC int
	// CPUFanRPM is the speed of the fan of the processor.
	CPUFanRPM int
	// CPUUsagePercent is the load of the processor in percent.
	CPUUsagePercent float64
	// CPUFrequencyMHz is the clock of the processor in megahertz.
	CPUFrequencyMHz int
	// CPUMaxFrequencyMHz is the highest clock the processor reaches.
	CPUMaxFrequencyMHz int
	// CPUCoreFrequencyMHz is the clock of every core, in megahertz.
	CPUCoreFrequencyMHz []int
	// SystemTempC is the temperature of the motherboard in degrees.
	SystemTempC int
	// GPUTemperatureC is the temperature of the graphics card in degrees.
	GPUTemperatureC int
	// GPUFanRPM is the speed of the fan of the graphics card.
	GPUFanRPM int
	// GPUUsagePercent is the load of the graphics processor in percent.
	GPUUsagePercent int
	// GPUFrequencyMHz is the clock of the graphics processor in megahertz.
	GPUFrequencyMHz int
	// GPUMaxFrequencyMHz is the highest clock the graphics processor reaches.
	GPUMaxFrequencyMHz int
	// GPUIdle reports whether the graphics processor is doing nothing.
	GPUIdle bool
	// RAMTotalMB is the memory of the machine in megabytes.
	RAMTotalMB float64
	// RAMUsagePercent is the load of the memory in percent.
	RAMUsagePercent float64
	// RAMFrequencyMHz is the clock of the memory in megahertz.
	RAMFrequencyMHz int
	// WIFIDownloadKbps is the traffic of the wireless adapter in kilobits
	// per second.
	WIFIDownloadKbps float64
	// WIFIUploadKbps is the traffic of the wireless adapter in kilobits per
	// second.
	WIFIUploadKbps float64
	// EthernetDownloadKbps is the traffic of the wired adapter in kilobits
	// per second.
	EthernetDownloadKbps float64
	// EthernetUploadKbps is the traffic of the wired adapter in kilobits per
	// second.
	EthernetUploadKbps float64
}
