//go:build windows

package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

type resourceStats struct {
	HasCPU            bool
	TotalCPUPercent   float64
	ProcessCPUPercent float64

	TotalMemoryBytes     uint64
	UsedMemoryBytes      uint64
	ProcessMemoryBytes   uint64
	TotalMemoryPercent   float64
	ProcessMemoryPercent float64

	Err error
}

type resourceSampler struct {
	prevSystem systemTimes
	prevProc   uint64
	hasPrev    bool
}

type systemTimes struct {
	idle   uint64
	kernel uint64
	user   uint64
}

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

var (
	modKernel32              = windows.NewLazySystemDLL("kernel32.dll")
	modPsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetSystemTimes       = modKernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
	procGetProcessMemoryInfo = modPsapi.NewProc("GetProcessMemoryInfo")
)

func newResourceSampler() *resourceSampler {
	return &resourceSampler{}
}

func (s *resourceSampler) Sample() resourceStats {
	stats := resourceStats{}

	mem, err := readMemoryStatus()
	if err != nil {
		stats.Err = err
	} else {
		stats.TotalMemoryBytes = mem.totalPhys
		stats.UsedMemoryBytes = mem.totalPhys - mem.availPhys
		if mem.totalPhys > 0 {
			stats.TotalMemoryPercent = float64(stats.UsedMemoryBytes) / float64(mem.totalPhys) * 100
		}
	}

	procMem, err := readProcessMemory()
	if err != nil && stats.Err == nil {
		stats.Err = err
	} else {
		stats.ProcessMemoryBytes = procMem
		if stats.TotalMemoryBytes > 0 {
			stats.ProcessMemoryPercent = float64(procMem) / float64(stats.TotalMemoryBytes) * 100
		}
	}

	sys, err := readSystemTimes()
	if err != nil {
		if stats.Err == nil {
			stats.Err = err
		}
		return stats
	}
	proc, err := readProcessCPUTime()
	if err != nil {
		if stats.Err == nil {
			stats.Err = err
		}
		return stats
	}
	if !s.hasPrev {
		s.prevSystem = sys
		s.prevProc = proc
		s.hasPrev = true
		return stats
	}

	idleDelta := float64(sys.idle - s.prevSystem.idle)
	kernelDelta := float64(sys.kernel - s.prevSystem.kernel)
	userDelta := float64(sys.user - s.prevSystem.user)
	totalDelta := kernelDelta + userDelta
	procDelta := float64(proc - s.prevProc)
	s.prevSystem = sys
	s.prevProc = proc

	if totalDelta > 0 {
		stats.HasCPU = true
		stats.TotalCPUPercent = clampPercent((totalDelta - idleDelta) / totalDelta * 100)
		stats.ProcessCPUPercent = clampPercent(procDelta / totalDelta * 100)
	}
	_ = runtime.NumCPU()
	return stats
}

func readSystemTimes() (systemTimes, error) {
	var idle, kernel, user windows.Filetime
	r, _, callErr := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return systemTimes{}, fmt.Errorf("GetSystemTimes: %w", callErr)
	}
	return systemTimes{
		idle:   filetimeTicks(idle),
		kernel: filetimeTicks(kernel),
		user:   filetimeTicks(user),
	}, nil
}

func readProcessCPUTime() (uint64, error) {
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(windows.CurrentProcess(), &creation, &exit, &kernel, &user); err != nil {
		return 0, fmt.Errorf("GetProcessTimes: %w", err)
	}
	return filetimeTicks(kernel) + filetimeTicks(user), nil
}

func readMemoryStatus() (memoryStatusEx, error) {
	var mem memoryStatusEx
	mem.length = uint32(unsafe.Sizeof(mem))
	r, _, callErr := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if r == 0 {
		return memoryStatusEx{}, fmt.Errorf("GlobalMemoryStatusEx: %w", callErr)
	}
	return mem, nil
}

func readProcessMemory() (uint64, error) {
	var counters processMemoryCounters
	counters.cb = uint32(unsafe.Sizeof(counters))
	r, _, callErr := procGetProcessMemoryInfo.Call(
		uintptr(windows.CurrentProcess()),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.cb),
	)
	if r == 0 {
		return 0, fmt.Errorf("GetProcessMemoryInfo: %w", callErr)
	}
	return uint64(counters.workingSetSize), nil
}

func filetimeTicks(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
