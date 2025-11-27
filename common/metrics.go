package common

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemMetrics contiene métricas del sistema
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	Goroutines    int     `json:"goroutines"`
	Timestamp     int64   `json:"timestamp"`
}

// MetricsCollector recolecta métricas del sistema
type MetricsCollector struct {
	lastCPUTime  int64
	lastSysTime  int64
	lastUserTime int64
}

// NewMetricsCollector crea un nuevo recolector de métricas
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// Collect recolecta las métricas actuales del sistema
func (mc *MetricsCollector) Collect() *SystemMetrics {
	metrics := &SystemMetrics{
		Timestamp:  time.Now().Unix(),
		Goroutines: runtime.NumGoroutine(),
	}

	// Memoria del proceso Go
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsedMB = int64(m.Alloc / 1024 / 1024)
	metrics.MemoryTotalMB = int64(m.Sys / 1024 / 1024)

	// CPU usage (aproximado)
	metrics.CPUPercent = mc.getCPUPercent()

	return metrics
}

// getCPUPercent calcula el porcentaje de uso de CPU (aproximado)
func (mc *MetricsCollector) getCPUPercent() float64 {
	// En Linux, leer /proc/self/stat
	if runtime.GOOS == "linux" {
		return mc.getCPUPercentLinux()
	}

	// En otros sistemas, usar aproximación simple basada en goroutines
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()

	// Aproximación: si hay más goroutines que CPUs, asumir uso proporcional
	percent := float64(numGoroutines) / float64(numCPU) * 10.0
	if percent > 100.0 {
		percent = 100.0
	}

	return percent
}

// getCPUPercentLinux lee CPU usage desde /proc/self/stat en Linux
func (mc *MetricsCollector) getCPUPercentLinux() float64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0.0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0.0
	}

	// Fields: utime(13), stime(14)
	utime, _ := strconv.ParseInt(fields[13], 10, 64)
	stime, _ := strconv.ParseInt(fields[14], 10, 64)
	totalTime := utime + stime

	// Calcular diferencia desde última medición
	if mc.lastCPUTime > 0 {
		// Leer clock ticks del sistema
		clockTicks := int64(100) // Típicamente 100 en Linux

		deltaTime := totalTime - mc.lastCPUTime
		deltaWall := time.Now().Unix() - (mc.lastSysTime / clockTicks)

		if deltaWall > 0 {
			numCPU := int64(runtime.NumCPU())
			percent := float64(deltaTime) / float64(deltaWall*clockTicks*numCPU) * 100.0

			mc.lastCPUTime = totalTime
			mc.lastSysTime = time.Now().Unix() * clockTicks

			if percent > 100.0 {
				percent = 100.0
			}
			if percent < 0.0 {
				percent = 0.0
			}

			return percent
		}
	}

	// Primera medición
	mc.lastCPUTime = totalTime
	mc.lastSysTime = time.Now().Unix() * 100

	return 0.0
}

// FormatMetrics formatea las métricas como string legible
func (metrics *SystemMetrics) FormatMetrics() string {
	return fmt.Sprintf("CPU: %.1f%%, Memory: %dMB/%dMB, Goroutines: %d",
		metrics.CPUPercent,
		metrics.MemoryUsedMB,
		metrics.MemoryTotalMB,
		metrics.Goroutines)
}
