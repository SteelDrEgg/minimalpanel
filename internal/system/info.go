package system

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
)

// GetSystemInfo returns general system information
func GetSystemInfo() (*SystemInfo, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	users, err := host.Users()
	if err != nil {
		users = nil // Continue without user info
	}

	var userHost string
	if len(users) > 0 && users[0].User != "" {
		userHost = users[0].User + "@" + hostInfo.Hostname
	} else {
		userHost = hostInfo.Hostname
	}

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	} else {
		cpuModel = "Unknown CPU"
	}

	// GPU info is limited without additional libraries
	// For now, we'll use a placeholder or runtime info
	gpuInfo := getGPUInfo()

	return &SystemInfo{
		User:   userHost,
		Host:   hostInfo.Hostname,
		OS:     fmt.Sprintf("%s %s %s", hostInfo.Platform, hostInfo.PlatformVersion, hostInfo.KernelArch),
		Kernel: fmt.Sprintf("%s %s", hostInfo.OS, hostInfo.KernelVersion),
		CPU:    cpuModel,
		GPU:    gpuInfo.Name,
		VRAM:   gpuInfo.VRAM,
	}, nil
}

// GPUInfo represents GPU information
type GPUInfo struct {
	Name string
	VRAM string
}

// getGPUInfo attempts to get GPU information
// This is a basic implementation since detailed GPU info requires platform-specific libraries
func getGPUInfo() GPUInfo {
	// Basic GPU detection based on OS
	switch runtime.GOOS {
	case "darwin":
		return GPUInfo{Name: "Apple GPU (Metal)", VRAM: ""}
	case "linux":
		return GPUInfo{Name: "Linux GPU", VRAM: ""}
	case "windows":
		return GPUInfo{Name: "Windows GPU", VRAM: ""}
	default:
		return GPUInfo{Name: "Unknown GPU", VRAM: ""}
	}
}

// GetHostInfo returns formatted host information
func GetHostInfo() (string, error) {
	hostStat, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get host info: %w", err)
	}

	users, _ := host.Users() // Ignore error for users
	hostName := hostStat.Hostname

	if len(users) > 0 && users[0].User != "" {
		return users[0].User + "@" + hostName, nil
	}
	return hostName, nil
}

// GetOSInfo returns formatted OS information
func GetOSInfo() (string, error) {
	hostStat, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get host info: %w", err)
	}

	return fmt.Sprintf("%s %s %s", hostStat.Platform, hostStat.PlatformVersion, hostStat.KernelArch), nil
}

// GetKernelInfo returns formatted kernel information
func GetKernelInfo() (string, error) {
	hostStat, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get host info: %w", err)
	}

	return fmt.Sprintf("%s %s", hostStat.OS, hostStat.KernelVersion), nil
}

// GetCPUInfo returns formatted CPU information
func GetCPUInfo() (string, error) {
	cpuStat, err := cpu.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get CPU info: %w", err)
	}

	if len(cpuStat) == 0 {
		return "Unknown CPU", nil
	}

	name := cpuStat[0].ModelName
	threads := fmt.Sprintf("%v", cpuStat[0].Cores)
	count := ""
	if len(cpuStat) > 1 {
		count = fmt.Sprintf("x%v", len(cpuStat))
	}

	return fmt.Sprintf("%s (%s%s)", name, threads, count), nil
}
