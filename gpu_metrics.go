package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
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
	cmd := exec.Command("amd-smi", "process", "--csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute amd-smi: %v", err)
	}

	// Create CSV reader
	reader := csv.NewReader(strings.NewReader(string(output)))

	// Read header line
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %v", err)
	}

	var processes []ProcessInfo
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %v", err)
		}

		// Skip if no process detected
		if strings.Contains(record[1], "No running processes detected") {
			continue
		}

		// Parse GPU ID
		gpuID, err := strconv.Atoi(record[0])
		if err != nil {
			continue
		}

		// Convert memory values from bytes to MB
		vramMem, _ := strconv.ParseFloat(record[2], 64)
		cpuMem, _ := strconv.ParseFloat(record[5], 64)
		gttMem, _ := strconv.ParseFloat(record[7], 64)
		totalMem, _ := strconv.ParseFloat(record[8], 64)

		process := ProcessInfo{
			GPU:      gpuID,
			Name:     record[3],
			PID:      record[4],
			GFXUsage: record[6] + "%",
			VRAMMem:  vramMem / 1024 / 1024, // Convert bytes to MB
			CPUMem:   cpuMem / 1024 / 1024,
			GTTMem:   gttMem / 1024 / 1024,
			TotalMem: totalMem / 1024 / 1024,
		}
		processes = append(processes, process)
	}

	return processes, nil
}
