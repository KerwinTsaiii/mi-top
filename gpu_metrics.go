package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type GPUMetrics struct {
	ID        int
	Power     float64
	GPUTemp   float64
	MemTemp   float64
	GFXUtil   float64
	GFXClock  float64
	MemUtil   float64
	MemClock  float64
	VRAMUsed  float64
	VRAMTotal float64
}

type ProcessInfo struct {
	GPU      int
	Name     string
	PID      string
	GTTMem   float64
	CPUMem   float64
	VRAMMem  float64
	TotalMem float64
	GFXUsage string
}

func getGPUMetrics() ([]GPUMetrics, error) {
	cmd := exec.Command("amd-smi", "monitor", "--csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var metrics []GPUMetrics
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header line
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")

		if len(fields) < 17 {
			continue
		}

		id, _ := strconv.Atoi(fields[0])
		power, _ := strconv.ParseFloat(fields[1], 64)
		gpuTemp, _ := strconv.ParseFloat(fields[2], 64)
		memTemp, _ := strconv.ParseFloat(fields[3], 64)
		gfxUtil, _ := strconv.ParseFloat(fields[4], 64)
		gfxClock, _ := strconv.ParseFloat(fields[5], 64)
		memUtil, _ := strconv.ParseFloat(fields[6], 64)
		memClock, _ := strconv.ParseFloat(fields[7], 64)
		vramUsed, _ := strconv.ParseFloat(fields[15], 64)
		vramTotal, _ := strconv.ParseFloat(fields[16], 64)

		metrics = append(metrics, GPUMetrics{
			ID:        id,
			Power:     power,
			GPUTemp:   gpuTemp,
			MemTemp:   memTemp,
			GFXUtil:   gfxUtil,
			GFXClock:  gfxClock,
			MemUtil:   memUtil,
			MemClock:  memClock,
			VRAMUsed:  vramUsed,
			VRAMTotal: vramTotal,
		})
	}

	return metrics, nil
}

func getProcessInfo() ([]ProcessInfo, error) {
	cmd := exec.Command("amd-smi", "process")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	currentGPU := -1
	var currentProcess ProcessInfo

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "GPU:") {
			fields := strings.Fields(line)
			currentGPU, _ = strconv.Atoi(fields[1])
		} else if strings.Contains(line, "NAME:") {
			if currentProcess.Name != "" {
				processes = append(processes, currentProcess)
			}
			currentProcess = ProcessInfo{GPU: currentGPU}
			currentProcess.Name = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(line, "PID:") {
			currentProcess.PID = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(line, "GTT_MEM:") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			currentProcess.GTTMem, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, " MB"), 64)
		} else if strings.Contains(line, "CPU_MEM:") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			currentProcess.CPUMem, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, " MB"), 64)
		} else if strings.Contains(line, "VRAM_MEM:") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			currentProcess.VRAMMem, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, " MB"), 64)
		} else if strings.Contains(line, "MEM_USAGE:") {
			memStr := strings.TrimSpace(strings.Split(line, ":")[1])
			currentProcess.TotalMem, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, " MB"), 64)
		} else if strings.Contains(line, "GFX:") {
			gfxStr := strings.TrimSpace(strings.Split(line, ":")[1])
			gfxStr = strings.TrimSuffix(gfxStr, " ns")
			if gfxTime, err := strconv.ParseInt(gfxStr, 10, 64); err == nil && gfxTime > 0 {
				currentProcess.GFXUsage = fmt.Sprintf("%.1f%%", float64(gfxTime)/1e9*100)
			} else {
				currentProcess.GFXUsage = "0.0%"
			}
		}
	}

	// Add the last process
	if currentProcess.Name != "" {
		processes = append(processes, currentProcess)
	}

	return processes, nil
}
