package system

// SystemInfo represents general system information
type SystemInfo struct {
	User   string `json:"user"`
	Host   string `json:"host"`
	OS     string `json:"os"`
	Kernel string `json:"kernel"`
	CPU    string `json:"cpu"`
	GPU    string `json:"gpu"`
	VRAM   string `json:"vram,omitempty"`
}

// SystemUsage represents system resource usage
type SystemUsage struct {
	Memory MemoryUsage `json:"memory"`
	Disk   DiskUsage   `json:"disk"`
	CPU    CPUUsage    `json:"cpu"`
}

// MemoryUsage represents memory usage information
type MemoryUsage struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	Swap        uint64  `json:"swap"`
}

// DiskUsage represents disk usage information
type DiskUsage struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// CPUUsage represents CPU usage information
type CPUUsage struct {
	Model   string  `json:"model"`
	Cores   int32   `json:"cores"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}
