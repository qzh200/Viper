//go:build tools

// Command nvprobe looks for the entry points of the graphics driver.
//
// The driver library offers two sets of calls, the management library and
// the interface of the driver itself. This tool reports which of the known
// names the library on this machine really exports.
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func main() {
	for _, library := range []string{"nvml.dll", "nvapi64.dll"} {
		fmt.Printf("=== %s ===\n", library)
		handle, err := syscall.LoadLibrary(library)
		if err != nil {
			fmt.Println("  load failed:", err)
			continue
		}

		candidates := map[string][]string{
			"nvml.dll": {
				"nvmlInit", "nvmlInit_v2",
				"nvmlShutdown",
				"nvmlDeviceGetCount", "nvmlDeviceGetCount_v2",
				"nvmlDeviceGetHandleByIndex", "nvmlDeviceGetHandleByIndex_v2",
				"nvmlDeviceGetUtilizationRates",
				"nvmlDeviceGetClockInfo", "nvmlDeviceGetClockInfo_v2",
				"nvmlDeviceGetMaxClockInfo",
				"nvmlDeviceGetPState",
				"nvmlDeviceGetName",
				"nvmlDeviceGetMemoryInfo",
				"nvmlDeviceGetMemoryInfo_v2",
				"nvmlSystemGetDriverVersion", "nvmlSystemGetDriverVersion_v2",
				"nvmlSystemGetNVMLVersion", "nvmlSystemGetNVMLVersion_v2",
				"nvmlErrorString",
				"nvmlDeviceGetGraphicsRunningProcesses_v3",
				"nvmlDeviceGetGpuInstanceId",
			},
			"nvapi64.dll": {
				"nvapi_Initialize",
				"nvapi_EnumPhysicalGPUs_v2",
				"nvapi_GPU_GetUsages_v3",
				"nvapi_GPU_GetAllClockFrequencies",
				"nvapi_GPU_GetPstates20",
				"nvapi_GPU_GetDynamicPstatesInfoEx",
				"nvapi_GPU_GetTopPerformanceState",
				"nvapi_GPU_GetThermalSettings",
				"nvapi_GPU_GetCoolerInformation",
				"nvapi_QueryInterface",
				"nvapi_EnumPhysicalAdapters",
				"nvapi_GPU_GetBoardInfo",
				"nvapi_GPU_GetMemoryUsage",
			},
		}

		for _, name := range candidates[library] {
			address, err := syscall.GetProcAddress(handle, name)
			if err != nil {
				continue
			}
			fmt.Printf("  [x] %-40s 0x%X\n", name, uintptr(address))
		}
		syscall.FreeLibrary(handle)
	}
	fmt.Println()
	_ = unsafe.Pointer(nil)
}
