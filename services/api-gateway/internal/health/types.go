package health

import "time"

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type Response struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	Services  map[string]CheckResult `json:"services,omitempty"`
	System    *SystemMetrics         `json:"system,omitempty"`
}

type CheckResult struct {
	Status       Status  `json:"status"`
	Message      string  `json:"message,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	LastChecked  string  `json:"last_checked,omitempty"`
	Error        string  `json:"error,omitempty"`
}

type SystemMetrics struct {
	CPU     CPUMetrics     `json:"cpu"`
	Memory  MemoryMetrics  `json:"memory"`
	Disk    []DiskMetrics  `json:"disk"`
	Network NetworkMetrics `json:"network"`
	Process ProcessMetrics `json:"process"`
	Host    HostMetrics    `json:"host"`
	Runtime RuntimeMetrics `json:"runtime"`
}

type CPUMetrics struct {
	Cores        int       `json:"cores"`
	UsagePercent []float64 `json:"usage_percent"` // Per core usage
	AvgUsage     float64   `json:"avg_usage"`
	ModelName    string    `json:"model_name,omitempty"`
	CacheSize    int32     `json:"cache_size,omitempty"`
	Frequency    float64   `json:"frequency_mhz,omitempty"`
}

type MemoryMetrics struct {
	// System Memory
	TotalMB      uint64  `json:"total_mb"`
	UsedMB       uint64  `json:"used_mb"`
	FreeMB       uint64  `json:"free_mb"`
	AvailableMB  uint64  `json:"available_mb"`
	UsagePercent float64 `json:"usage_percent"`
	CachedMB     uint64  `json:"cached_mb,omitempty"`
	BuffersMB    uint64  `json:"buffers_mb,omitempty"`

	// Swap Memory
	SwapTotalMB      uint64  `json:"swap_total_mb"`
	SwapUsedMB       uint64  `json:"swap_used_mb"`
	SwapFreeMB       uint64  `json:"swap_free_mb"`
	SwapUsagePercent float64 `json:"swap_usage_percent"`
}

type DiskMetrics struct {
	Device       string  `json:"device"`
	Mountpoint   string  `json:"mountpoint"`
	Fstype       string  `json:"fstype"`
	TotalGB      uint64  `json:"total_gb"`
	UsedGB       uint64  `json:"used_gb"`
	FreeGB       uint64  `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
	InodesTotal  uint64  `json:"inodes_total,omitempty"`
	InodesUsed   uint64  `json:"inodes_used,omitempty"`
	InodesFree   uint64  `json:"inodes_free,omitempty"`
}

type NetworkMetrics struct {
	Interfaces       []NetworkInterface `json:"interfaces"`
	TotalBytesSent   uint64             `json:"total_bytes_sent"`
	TotalBytesRecv   uint64             `json:"total_bytes_recv"`
	TotalPacketsSent uint64             `json:"total_packets_sent"`
	TotalPacketsRecv uint64             `json:"total_packets_recv"`
}

type NetworkInterface struct {
	Name        string   `json:"name"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	PacketsSent uint64   `json:"packets_sent"`
	PacketsRecv uint64   `json:"packets_recv"`
	ErrorsIn    uint64   `json:"errors_in"`
	ErrorsOut   uint64   `json:"errors_out"`
	DropIn      uint64   `json:"drop_in"`
	DropOut     uint64   `json:"drop_out"`
	Addrs       []string `json:"addrs,omitempty"`
}

type ProcessMetrics struct {
	PID            int32   `json:"pid"`
	Name           string  `json:"name"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryUsedMB   float64 `json:"memory_used_mb"`
	MemoryPercent  float32 `json:"memory_percent"`
	NumThreads     int32   `json:"num_threads"`
	NumFDs         int32   `json:"num_fds,omitempty"` // File descriptors (Unix only)
	NumConnections int     `json:"num_connections"`
	CreateTime     int64   `json:"create_time"`
	Status         string  `json:"status"`
}

type HostMetrics struct {
	Hostname             string `json:"hostname"`
	Uptime               uint64 `json:"uptime_seconds"`
	UptimeFormatted      string `json:"uptime_formatted"`
	OS                   string `json:"os"`
	Platform             string `json:"platform"`
	PlatformFamily       string `json:"platform_family"`
	PlatformVersion      string `json:"platform_version"`
	KernelVersion        string `json:"kernel_version"`
	KernelArch           string `json:"kernel_arch"`
	VirtualizationSystem string `json:"virtualization_system,omitempty"`
	VirtualizationRole   string `json:"virtualization_role,omitempty"`
	BootTime             uint64 `json:"boot_time"`
	Procs                uint64 `json:"total_processes"`
}

type RuntimeMetrics struct {
	GoVersion      string  `json:"go_version"`
	GoroutineCount int     `json:"goroutine_count"`
	GoMaxProcs     int     `json:"gomaxprocs"`
	HeapAllocMB    float64 `json:"heap_alloc_mb"`
	HeapSysMB      float64 `json:"heap_sys_mb"`
	HeapIdleMB     float64 `json:"heap_idle_mb"`
	HeapInuseMB    float64 `json:"heap_inuse_mb"`
	HeapReleasedMB float64 `json:"heap_released_mb"`
	StackInuseMB   float64 `json:"stack_inuse_mb"`
	StackSysMB     float64 `json:"stack_sys_mb"`
	GCSys          float64 `json:"gc_sys_mb"`
	NumGC          uint32  `json:"num_gc"`
	LastGCTime     string  `json:"last_gc_time,omitempty"`
	NextGC         float64 `json:"next_gc_mb"`
	PauseNs        uint64  `json:"last_pause_ns"`
	PauseTotal     float64 `json:"pause_total_ms"`
}
