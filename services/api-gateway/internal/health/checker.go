package health

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

type Manager struct {
	serviceName string
	version     string
	startTime   time.Time
	checkers    map[string]Checker
	mu          sync.RWMutex
	cache       map[string]cachedResult
	cacheTTL    time.Duration
}

type cachedResult struct {
	result    CheckResult
	timestamp time.Time
}

func NewManager(serviceName, version string) *Manager {
	return &Manager{
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
		checkers:    make(map[string]Checker),
		cache:       make(map[string]cachedResult),
		cacheTTL:    5 * time.Second,
	}
}

func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers[checker.Name()] = checker
}

func (m *Manager) Health(ctx context.Context, includeChecks bool) Response {
	resp := Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Service:   m.serviceName,
		Version:   m.version,
		Uptime:    time.Since(m.startTime).String(),
	}

	if includeChecks {
		resp.Checks = m.runChecks(ctx)
		for _, check := range resp.Checks {
			if check.Status == StatusUnhealthy {
				resp.Status = StatusUnhealthy
				break
			} else if check.Status == StatusDegraded && resp.Status == StatusHealthy {
				resp.Status = StatusDegraded
			}
		}
	}

	return resp
}

func (m *Manager) HealthWithServices(ctx context.Context, services map[string]CheckResult) Response {
	resp := m.Health(ctx, true)
	resp.Services = services

	for _, svc := range services {
		if svc.Status == StatusUnhealthy {
			resp.Status = StatusDegraded
		}
	}

	return resp
}

func (m *Manager) Liveness(ctx context.Context) Response {
	return Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Service:   m.serviceName,
		Version:   m.version,
		Uptime:    time.Since(m.startTime).String(),
	}
}

func (m *Manager) Readiness(ctx context.Context) Response {
	resp := m.Health(ctx, true)
	if resp.Status == StatusUnhealthy {
		resp.Status = StatusUnhealthy
	}
	return resp
}

func (m *Manager) Detailed(ctx context.Context) Response {
	resp := m.Health(ctx, true)
	resp.System = m.collectSystemMetrics()
	return resp
}

func (m *Manager) collectSystemMetrics() *SystemMetrics {
	metrics := &SystemMetrics{}

	// Collect CPU metrics
	metrics.CPU = m.collectCPUMetrics()

	// Collect Memory metrics
	metrics.Memory = m.collectMemoryMetrics()

	// Collect Disk metrics
	metrics.Disk = m.collectDiskMetrics()

	// Collect Network metrics
	metrics.Network = m.collectNetworkMetrics()

	// Collect Process metrics
	metrics.Process = m.collectProcessMetrics()

	// Collect Host metrics
	metrics.Host = m.collectHostMetrics()

	// Collect Go Runtime metrics
	metrics.Runtime = m.collectRuntimeMetrics()

	return metrics
}

func (m *Manager) collectCPUMetrics() CPUMetrics {
	cpuMetrics := CPUMetrics{
		Cores: runtime.NumCPU(),
	}

	// Get CPU usage per core
	if percents, err := cpu.Percent(time.Second, true); err == nil {
		cpuMetrics.UsagePercent = percents
		// Calculate average
		var sum float64
		for _, p := range percents {
			sum += p
		}
		if len(percents) > 0 {
			cpuMetrics.AvgUsage = sum / float64(len(percents))
		}
	}

	// Get CPU info
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		cpuMetrics.ModelName = cpuInfo[0].ModelName
		cpuMetrics.CacheSize = cpuInfo[0].CacheSize
		cpuMetrics.Frequency = cpuInfo[0].Mhz
	}

	return cpuMetrics
}

func (m *Manager) collectMemoryMetrics() MemoryMetrics {
	memMetrics := MemoryMetrics{}

	// Virtual memory (RAM)
	if vmStat, err := mem.VirtualMemory(); err == nil {
		memMetrics.TotalMB = vmStat.Total / 1024 / 1024
		memMetrics.UsedMB = vmStat.Used / 1024 / 1024
		memMetrics.FreeMB = vmStat.Free / 1024 / 1024
		memMetrics.AvailableMB = vmStat.Available / 1024 / 1024
		memMetrics.UsagePercent = vmStat.UsedPercent
		memMetrics.CachedMB = vmStat.Cached / 1024 / 1024
		memMetrics.BuffersMB = vmStat.Buffers / 1024 / 1024
	}

	// Swap memory
	if swapStat, err := mem.SwapMemory(); err == nil {
		memMetrics.SwapTotalMB = swapStat.Total / 1024 / 1024
		memMetrics.SwapUsedMB = swapStat.Used / 1024 / 1024
		memMetrics.SwapFreeMB = swapStat.Free / 1024 / 1024
		memMetrics.SwapUsagePercent = swapStat.UsedPercent
	}

	return memMetrics
}

func (m *Manager) collectDiskMetrics() []DiskMetrics {
	var diskMetrics []DiskMetrics

	partitions, err := disk.Partitions(false)
	if err != nil {
		return diskMetrics
	}

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		dm := DiskMetrics{
			Device:       partition.Device,
			Mountpoint:   partition.Mountpoint,
			Fstype:       partition.Fstype,
			TotalGB:      usage.Total / 1024 / 1024 / 1024,
			UsedGB:       usage.Used / 1024 / 1024 / 1024,
			FreeGB:       usage.Free / 1024 / 1024 / 1024,
			UsagePercent: usage.UsedPercent,
			InodesTotal:  usage.InodesTotal,
			InodesUsed:   usage.InodesUsed,
			InodesFree:   usage.InodesFree,
		}

		diskMetrics = append(diskMetrics, dm)
	}

	return diskMetrics
}

func (m *Manager) collectNetworkMetrics() NetworkMetrics {
	netMetrics := NetworkMetrics{
		Interfaces: []NetworkInterface{},
	}

	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return netMetrics
	}

	for _, ioCounter := range ioCounters {
		// Get interface addresses
		var addrs []string
		if interfaces, err := net.Interfaces(); err == nil {
			for _, iface := range interfaces {
				if iface.Name == ioCounter.Name {
					for _, addr := range iface.Addrs {
						addrs = append(addrs, addr.Addr)
					}
					break
				}
			}
		}

		netInterface := NetworkInterface{
			Name:        ioCounter.Name,
			BytesSent:   ioCounter.BytesSent,
			BytesRecv:   ioCounter.BytesRecv,
			PacketsSent: ioCounter.PacketsSent,
			PacketsRecv: ioCounter.PacketsRecv,
			ErrorsIn:    ioCounter.Errin,
			ErrorsOut:   ioCounter.Errout,
			DropIn:      ioCounter.Dropin,
			DropOut:     ioCounter.Dropout,
			Addrs:       addrs,
		}

		netMetrics.Interfaces = append(netMetrics.Interfaces, netInterface)
		netMetrics.TotalBytesSent += ioCounter.BytesSent
		netMetrics.TotalBytesRecv += ioCounter.BytesRecv
		netMetrics.TotalPacketsSent += ioCounter.PacketsSent
		netMetrics.TotalPacketsRecv += ioCounter.PacketsRecv
	}

	return netMetrics
}

func (m *Manager) collectProcessMetrics() ProcessMetrics {
	procMetrics := ProcessMetrics{
		PID: int32(os.Getpid()),
	}

	proc, err := process.NewProcess(procMetrics.PID)
	if err != nil {
		return procMetrics
	}

	// Process name
	if name, err := proc.Name(); err == nil {
		procMetrics.Name = name
	}

	// CPU percent
	if cpuPercent, err := proc.CPUPercent(); err == nil {
		procMetrics.CPUPercent = cpuPercent
	}

	// Memory info
	if memInfo, err := proc.MemoryInfo(); err == nil {
		procMetrics.MemoryUsedMB = float64(memInfo.RSS) / 1024 / 1024
	}

	// Memory percent
	if memPercent, err := proc.MemoryPercent(); err == nil {
		procMetrics.MemoryPercent = memPercent
	}

	// Number of threads
	if numThreads, err := proc.NumThreads(); err == nil {
		procMetrics.NumThreads = numThreads
	}

	// Number of file descriptors (Unix only)
	if numFDs, err := proc.NumFDs(); err == nil {
		procMetrics.NumFDs = numFDs
	}

	// Number of connections
	if connections, err := proc.Connections(); err == nil {
		procMetrics.NumConnections = len(connections)
	}

	// Create time
	if createTime, err := proc.CreateTime(); err == nil {
		procMetrics.CreateTime = createTime
	}

	// Status
	if status, err := proc.Status(); err == nil {
		if len(status) > 0 {
			procMetrics.Status = status[0]
		}
	}

	return procMetrics
}

func (m *Manager) collectHostMetrics() HostMetrics {
	hostMetrics := HostMetrics{}

	// Host info
	if hostInfo, err := host.Info(); err == nil {
		hostMetrics.Hostname = hostInfo.Hostname
		hostMetrics.Uptime = hostInfo.Uptime
		hostMetrics.UptimeFormatted = formatDuration(time.Duration(hostInfo.Uptime) * time.Second)
		hostMetrics.OS = hostInfo.OS
		hostMetrics.Platform = hostInfo.Platform
		hostMetrics.PlatformFamily = hostInfo.PlatformFamily
		hostMetrics.PlatformVersion = hostInfo.PlatformVersion
		hostMetrics.KernelVersion = hostInfo.KernelVersion
		hostMetrics.KernelArch = hostInfo.KernelArch
		hostMetrics.VirtualizationSystem = hostInfo.VirtualizationSystem
		hostMetrics.VirtualizationRole = hostInfo.VirtualizationRole
		hostMetrics.BootTime = hostInfo.BootTime
		hostMetrics.Procs = hostInfo.Procs
	}

	return hostMetrics
}

func (m *Manager) collectRuntimeMetrics() RuntimeMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	runtimeMetrics := RuntimeMetrics{
		GoVersion:      runtime.Version(),
		GoroutineCount: runtime.NumGoroutine(),
		GoMaxProcs:     runtime.GOMAXPROCS(0),
		HeapAllocMB:    float64(memStats.Alloc) / 1024 / 1024,
		HeapSysMB:      float64(memStats.HeapSys) / 1024 / 1024,
		HeapIdleMB:     float64(memStats.HeapIdle) / 1024 / 1024,
		HeapInuseMB:    float64(memStats.HeapInuse) / 1024 / 1024,
		HeapReleasedMB: float64(memStats.HeapReleased) / 1024 / 1024,
		StackInuseMB:   float64(memStats.StackInuse) / 1024 / 1024,
		StackSysMB:     float64(memStats.StackSys) / 1024 / 1024,
		GCSys:          float64(memStats.GCSys) / 1024 / 1024,
		NumGC:          memStats.NumGC,
		NextGC:         float64(memStats.NextGC) / 1024 / 1024,
		PauseNs:        memStats.PauseNs[(memStats.NumGC+255)%256],
		PauseTotal:     float64(memStats.PauseTotalNs) / 1000000,
	}

	// Last GC time
	if memStats.NumGC > 0 {
		lastGC := time.Unix(0, int64(memStats.LastGC))
		runtimeMetrics.LastGCTime = lastGC.Format(time.RFC3339)
	}

	return runtimeMetrics
}

func formatDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func (m *Manager) runChecks(ctx context.Context) map[string]CheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]CheckResult)
	now := time.Now()

	for name, checker := range m.checkers {
		if cached, ok := m.cache[name]; ok && now.Sub(cached.timestamp) < m.cacheTTL {
			results[name] = cached.result
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		result := checker.Check(checkCtx)
		cancel()

		m.cache[name] = cachedResult{
			result:    result,
			timestamp: now,
		}
		results[name] = result
	}

	return results
}

func (m *Manager) HTTPStatus(status Status) int {
	switch status {
	case StatusHealthy:
		return http.StatusOK
	case StatusDegraded:
		return http.StatusOK
	case StatusUnhealthy:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
