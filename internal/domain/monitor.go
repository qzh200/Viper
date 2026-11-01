package domain

type MonitorData struct {
	CPUTemperatureC int

	CPUFanRPM int

	CPUUsagePercent float64

	CPUFrequencyMHz int

	CPUMaxFrequencyMHz int

	CPUCoreFrequencyMHz []int

	SystemTempC int

	GPUTemperatureC int

	GPUFanRPM int

	GPUUsagePercent int

	GPUFrequencyMHz int

	GPUMaxFrequencyMHz int

	GPUIdle bool

	RAMTotalMB float64

	RAMUsagePercent float64

	RAMFrequencyMHz int

	WIFIDownloadKbps float64

	WIFIUploadKbps float64

	EthernetDownloadKbps float64

	EthernetUploadKbps float64
}
