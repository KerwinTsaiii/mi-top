// Copyright (c) 2024-2025 Carsen Klock under MIT License
// amdtop is a simple terminal based AMD GPU monitor written in Go Lang!
package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// Global variables
var (
	processList    *widgets.List
	selectedColumn int
	sortReverse    bool
	columns        = []string{"GPU", "Name", "PID", "Usage"}
)

// ProcessListItem for sorting
type ProcessListItem struct {
	gpu     int
	name    string
	pid     string
	usage   string
	display string
}

func updateProcessList(processes []ProcessInfo) {
	items := make([]ProcessListItem, 0)
	// Find the longest name length for alignment
	maxNameLen := 20 // Default minimum width
	maxPIDLen := 8   // PID width
	for _, proc := range processes {
		if len(proc.Name) > maxNameLen {
			maxNameLen = len(proc.Name)
		}
		item := ProcessListItem{
			gpu:   proc.GPU,
			name:  proc.Name,
			pid:   proc.PID,
			usage: proc.GFXUsage,
			// Update display format to include memory information
			display: fmt.Sprintf("[%2d] %-*s │ PID: %-*s │ MEM: %6.1f MB (VRAM: %6.1f MB, GTT: %6.1f MB, CPU: %6.1f MB) │ GFX: %6s",
				proc.GPU,
				maxNameLen, proc.Name,
				maxPIDLen, proc.PID,
				proc.TotalMem,
				proc.VRAMMem,
				proc.GTTMem,
				proc.CPUMem,
				proc.GFXUsage),
		}
		items = append(items, item)
	}
	// Update header format
	header := fmt.Sprintf("[GPU] %-*s │ %-*s │ %-58s │ %-10s",
		maxNameLen, "NAME",
		maxPIDLen+5, "PID",
		"MEMORY USAGE",
		"GPU USAGE")
	// Sort based on selected column
	sort.Slice(items, func(i, j int) bool {
		var result bool
		switch selectedColumn {
		case 0: // GPU
			result = items[i].gpu < items[j].gpu
		case 1: // Name
			result = items[i].name < items[j].name
		case 2: // PID
			result = items[i].pid < items[j].pid
		case 3: // Usage
			result = items[i].usage < items[j].usage
		}
		if sortReverse {
			return !result
		}
		return result
	})
	// Update list display
	processList.Rows = make([]string, len(items)+2) // +2 for header and separator
	processList.Rows[0] = header
	processList.Rows[1] = strings.Repeat("─", len(header))
	for i, item := range items {
		processList.Rows[i+2] = item.display
	}
}
func handleProcessListEvents(e ui.Event) {
	switch e.ID {
	case "<Up>":
		if processList.SelectedRow > 0 {
			processList.SelectedRow--
		}
	case "<Down>":
		if processList.SelectedRow < len(processList.Rows)-1 {
			processList.SelectedRow++
		}
	case "<Left>":
		if selectedColumn > 0 {
			selectedColumn--
			processList.Title = fmt.Sprintf("Process List (Sort: %s%s)",
				columns[selectedColumn],
				map[bool]string{true: " ↓", false: " ↑"}[sortReverse])
		}
	case "<Right>":
		if selectedColumn < len(columns)-1 {
			selectedColumn++
			processList.Title = fmt.Sprintf("Process List (Sort: %s%s)",
				columns[selectedColumn],
				map[bool]string{true: " ↓", false: " ↑"}[sortReverse])
		}
	case "<Enter>", "<Space>":
		sortReverse = !sortReverse
	}
}

// Store GPU utilization history
type GPUHistory struct {
	values []float64
	maxLen int
	index  int // Track current position
}

func newGPUHistory(maxLen int) *GPUHistory {
	return &GPUHistory{
		values: make([]float64, maxLen), // Create a fixed size array
		maxLen: maxLen,
		index:  0,
	}
}
func (gh *GPUHistory) add(value float64) {
	gh.values[gh.index] = value
	gh.index = (gh.index + 1) % gh.maxLen
}

// Get ordered data
func (gh *GPUHistory) getData() []float64 {
	if gh.index == 0 {
		return gh.values
	}
	result := make([]float64, gh.maxLen)
	copy(result, gh.values[gh.index:])
	copy(result[gh.maxLen-gh.index:], gh.values[:gh.index])
	return result
}

// Helper function to calculate appropriate number of data points
func calculateDataPoints(width int) int {
	// Consider borders and other UI elements for actual usable width
	usableWidth := width - 4 // Subtract borders and padding
	if usableWidth < 50 {
		return 50 // Minimum data points
	}
	if usableWidth > 500 {
		return 500 // Maximum data points
	}
	return usableWidth
}
func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("amdtop version %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build time: %s\n", BuildTime)
		return
	}
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()
	// Get terminal dimensions early
	termWidth, termHeight := ui.TerminalDimensions()
	dataPoints := calculateDataPoints(termWidth)
	// Get number of GPUs
	metrics, err := getGPUMetrics()
	if err != nil {
		log.Fatalf("failed to get GPU metrics: %v", err)
	}
	numGPUs := len(metrics)
	// Create GPU charts
	gpuCharts := make([]*widgets.SparklineGroup, numGPUs)
	gpuHistories := make([]*GPUHistory, numGPUs)
	for i := 0; i < numGPUs; i++ {
		sparkline := widgets.NewSparkline()
		sparkline.LineColor = ui.ColorGreen
		sparkline.TitleStyle = ui.NewStyle(ui.ColorWhite)
		sparkline.MaxVal = 100
		// Use calculated number of data points
		initialData := make([]float64, dataPoints)
		sparkline.Data = initialData
		spGroup := widgets.NewSparklineGroup()
		spGroup.Title = fmt.Sprintf("GPU %d", i)
		spGroup.Sparklines = []*widgets.Sparkline{sparkline}
		spGroup.BorderStyle = ui.NewStyle(ui.ColorWhite)
		spGroup.BorderLeft = true
		spGroup.BorderRight = true
		spGroup.BorderTop = true
		spGroup.BorderBottom = true
		// Set minimum height
		spGroup.SetRect(0, 0, termWidth, 10)
		gpuCharts[i] = spGroup
		gpuHistories[i] = newGPUHistory(dataPoints)
		// Initialize history data to 0
		for j := 0; j < dataPoints; j++ {
			gpuHistories[i].add(0)
		}
	}
	// Initialize process list
	processList = widgets.NewList()
	processList.Title = fmt.Sprintf("Process List (Sort: %s%s)",
		columns[selectedColumn],
		map[bool]string{true: " ↓", false: " ↑"}[sortReverse])
	processList.TextStyle = ui.NewStyle(ui.ColorWhite)
	processList.WrapText = false
	processList.SelectedRow = 0
	processList.BorderStyle = ui.NewStyle(ui.ColorWhite)
	// Set selected row color
	processList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorGreen)
	// Layout
	grid := ui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)
	// Adjust grid layout to use more space
	gridItems := make([]interface{}, 0)
	chartHeight := float64(0.8) / float64(numGPUs)
	for i := 0; i < numGPUs; i++ {
		gridItems = append(gridItems, ui.NewRow(chartHeight, ui.NewCol(1.0, gpuCharts[i])))
	}
	gridItems = append(gridItems, ui.NewRow(0.2, ui.NewCol(1.0, processList)))
	grid.Set(gridItems...)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				newDataPoints := calculateDataPoints(payload.Width)
				// Update number of data points for each chart
				for i := 0; i < numGPUs; i++ {
					newHistory := newGPUHistory(newDataPoints)
					// Copy existing data to new history
					oldData := gpuHistories[i].getData()
					for _, v := range oldData {
						newHistory.add(v)
					}
					gpuHistories[i] = newHistory
					gpuCharts[i].SetRect(0, 0, payload.Width, 10)
					gpuCharts[i].Sparklines[0].Data = make([]float64, newDataPoints)
				}
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			default:
				handleProcessListEvents(e)
			}
		case <-ticker.C:
			// Update metrics
			metrics, err := getGPUMetrics()
			if err == nil {
				for i, metric := range metrics {
					if i >= len(gpuCharts) {
						break
					}
					// Add new utilization data
					gpuHistories[i].add(metric.GFXUtil)
					// Update chart data using getData() to get correct order
					gpuCharts[i].Sparklines[0].Data = gpuHistories[i].getData()
					gpuCharts[i].Sparklines[0].MaxVal = 100
					// Update title, add current utilization
					gpuCharts[i].Title = fmt.Sprintf("GPU %d - %0.1fW, %0.1f°C, %0.1f%% Util, VRAM: %0.0f/%0.0f MB",
						metric.ID, metric.Power, metric.GPUTemp, metric.GFXUtil, metric.VRAMUsed, metric.VRAMTotal)
				}
			}
			// Update process list
			processes, err := getProcessInfo()
			if err == nil {
				updateProcessList(processes)
			}
			ui.Render(grid)
		}
	}
}
