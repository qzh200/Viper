package hardware

// MonitorData 汇集整机实时硬件监控遥测指标。
type MonitorData struct {
	// CPUTemperatureC 表示 CPU 封装温度（摄氏度）。
	CPUTemperatureC int

	// CPUFanRPM 表示 CPU 风扇当前转速。
	CPUFanRPM int

	// CPUUsagePercent 表示 CPU 总体占用率百分比。
	CPUUsagePercent float64

	// CPUFrequencyMHz 表示 CPU 当前运行频率（MHz）。
	CPUFrequencyMHz int

	// CPUMaxFrequencyMHz 表示 CPU 理论最大主频（MHz）。
	CPUMaxFrequencyMHz int

	// CPUCoreFrequencyMHz 记录各 CPU 核心的即时频率。
	CPUCoreFrequencyMHz []int

	// SystemTempC 表示主板/系统综合温度。
	SystemTempC int

	// GPUTemperatureC 表示独立 GPU 核心温度。
	GPUTemperatureC int

	// GPUFanRPM 表示 GPU 风扇当前转速。
	GPUFanRPM int

	// GPUUsagePercent 表示 GPU 核心负载百分比。
	GPUUsagePercent int

	// GPUFrequencyMHz 表示 GPU 核心当前时钟频率。
	GPUFrequencyMHz int

	// GPUMaxFrequencyMHz 表示 GPU 最大时钟频率。
	GPUMaxFrequencyMHz int

	// GPUIdle 标识独立 GPU 当前是否处于空闲待机状态。
	GPUIdle bool

	// RAMTotalMB 表示物理内存总容量（MB）。
	RAMTotalMB float64

	// RAMUsagePercent 表示物理内存使用率百分比。
	RAMUsagePercent float64

	// RAMFrequencyMHz 表示内存工作频率（MHz）。
	RAMFrequencyMHz int

	// WIFIDownloadKbps 记录无线网络即时下载速率（Kbps）。
	WIFIDownloadKbps float64

	// WIFIUploadKbps 记录无线网络即时上传速率（Kbps）。
	WIFIUploadKbps float64

	// EthernetDownloadKbps 记录有线以太网即时下载速率（Kbps）。
	EthernetDownloadKbps float64

	// EthernetUploadKbps 记录有线以太网即时上传速率（Kbps）。
	EthernetUploadKbps float64
}
