package system

import (
	"fmt"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// GetSystemUsage returns current system resource usage
func GetSystemUsage() (*SystemUsage, error) {
	memUsage, err := GetMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}

	diskUsage, err := GetDiskUsage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	cpuUsage, err := GetCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	return &SystemUsage{
		Memory: *memUsage,
		Disk:   *diskUsage,
		CPU:    *cpuUsage,
	}, nil
}

// GetMemoryUsage returns memory usage information
func GetMemoryUsage() (*MemoryUsage, error) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		// If swap info fails, continue with 0 swap
		swapStat = &mem.SwapMemoryStat{Total: 0}
	}

	return &MemoryUsage{
		Total:       memStat.Total,
		Used:        memStat.Used,
		Available:   memStat.Available,
		UsedPercent: memStat.UsedPercent,
		Swap:        swapStat.Total,
	}, nil
}

// GetDiskUsage returns disk usage information for the specified path
func GetDiskUsage(path string) (*DiskUsage, error) {
	diskStat, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage for path %s: %w", path, err)
	}

	return &DiskUsage{
		Total:       diskStat.Total,
		Used:        diskStat.Used,
		Free:        diskStat.Free,
		UsedPercent: diskStat.UsedPercent,
	}, nil
}

// GetCPUUsage returns CPU usage information
func GetCPUUsage() (*CPUUsage, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	if len(cpuInfo) == 0 {
		return nil, fmt.Errorf("no CPU information available")
	}

	// Get CPU usage percentage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var totalPercent float64
	if len(cpuPercent) > 0 {
		totalPercent = cpuPercent[0]
	}

	return &CPUUsage{
		Model:   cpuInfo[0].ModelName,
		Cores:   cpuInfo[0].Cores,
		Count:   len(cpuInfo),
		Percent: totalPercent,
	}, nil
}

func properUnitHelper(bytes uint64, pow uint8, unit string) string {
	quotient := bytes >> pow
	temp := bytes & ((1 << pow) - 1)
	temp = ((temp * 10) + ((1 << pow) >> 1)) >> pow
	if temp == 10 {
		temp = 0
		quotient += 1
	}
	return strconv.FormatUint(quotient, 10) +
		"." + strconv.FormatUint(temp, 10) + " " + unit
}

// ProperUnit converts bytes to human readable format
func ProperUnit(byteNum uint64) (formatted string) {
	if byteNum >= 1<<40 { // TiB
		return properUnitHelper(byteNum, 40, "TiB")
	} else if byteNum >= 1<<30 { // GiB
		return properUnitHelper(byteNum, 30, "GiB")
	} else if byteNum >= 1<<20 { // MiB
		return properUnitHelper(byteNum, 20, "MiB")
	} else if byteNum >= 1<<10 { // KiB
		return properUnitHelper(byteNum, 10, "KiB")
	}
	return strconv.FormatUint(byteNum, 10) + " B"
}

// Float2string converts float to string with specified precision
func Float2string(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}
