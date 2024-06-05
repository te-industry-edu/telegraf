//go:generate ../../../tools/readme_config_includer/generator
//go:build windows

package win_w3wp

import (
	_ "embed"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

// AppPoolMetric represents the metrics for an application pool.
type AppPoolMetric struct {
	percentProcessorTime float64 // The percentage of processor time used by the application pool.
	privateBytes         uint64  // The amount of private memory allocated for the application pool.
	workingSet           uint64  // The amount of physical memory allocated for the application pool.
	handleCount          uint32  // The number of handles currently open by the application pool.
	threadCount          uint32  // The number of threads currently active in the application pool.
	ioReadBytesPerSec    uint64  // The rate of input/output read operations per second for the application pool.
	ioWriteBytesPerSec   uint64  // The rate of input/output write operations per second for the application pool.
}

// Win32_PerfFormattedData_PerfProc_Process represents performance data for a process on a Windows system.
type Win32_PerfFormattedData_PerfProc_Process struct {
	IDProcess            uint32 // The unique identifier of the process.
	Name                 string // The name of the process.
	PercentProcessorTime uint64 // The percentage of processor time used by the process.
	PrivateBytes         uint64 // The amount of private memory allocated to the process.
	WorkingSet           uint64 // The amount of physical memory allocated to the process.
	HandleCount          uint32 // The number of handles opened by the process.
	ThreadCount          uint32 // The number of threads created by the process.
	IOReadBytesPerSec    uint64 // The rate at which the process reads data from I/O devices.
	IOWriteBytesPerSec   uint64 // The rate at which the process writes data to I/O devices.
}

// Win32_Process represents a Windows process.
type Win32_Process struct {
	ProcessId   uint32 // ProcessId is the unique identifier of the process.
	Name        string // Name is the name of the process.
	CommandLine string // CommandLine is the command line used to start the process.
}

// WmiClient is an interface for querying WMI (Windows Management Instrumentation) data.
// This interface can be used to mock the WMI client for testing purposes.
type WmiClient interface {
	Query(query string, dst interface{}, connectServerArgs ...interface{}) error
}

// WinW3wp represents a plugin for collecting metrics from Windows IIS worker processes (w3wp).
type WinW3wp struct {
	Log         telegraf.Logger `toml:"-"`
	IntervalStr string          `toml:"interval"`
	// AppPoolMetrics is a map that stores the metrics for each application pool.
	// The keys of the map are the names of the application pools, and the values
	// are pointers to AppPoolMetric structs that contain the metrics for each pool.
	AppPoolMetrics map[string]*AppPoolMetric
	// WmiClient represents a client for interacting with Windows Management Instrumentation (WMI).
	// If not set, the default WMI client will be used.
	WmiClient WmiClient

	interval    time.Duration
	processMap  map[uint32]string
	metricsChan chan []Win32_PerfFormattedData_PerfProc_Process
}

// SampleConfig returns the sample configuration for the WinW3wp plugin.
func (*WinW3wp) SampleConfig() string {
	return sampleConfig
}

// Init is for setup, and validating config.
func (w *WinW3wp) Init() error {
	if w.IntervalStr == "" {
		w.IntervalStr = "10s"
	}

	var err error
	w.interval, err = time.ParseDuration(w.IntervalStr)
	if err != nil {
		return err
	}

	if w.WmiClient == nil {
		w.WmiClient = wmi.DefaultClient
	}

	w.AppPoolMetrics = make(map[string]*AppPoolMetric)
	w.processMap = make(map[uint32]string)
	w.metricsChan = make(chan []Win32_PerfFormattedData_PerfProc_Process, 1)

	go w.updateProcessMap(w.processMap, w.interval)
	go w.updateMetrics(w.processMap, w.interval, w.metricsChan)
	go w.updateAppPoolMetrics(w.processMap, w.metricsChan)

	return nil
}

// updateProcessMap updates the process map with the app pool names of the running w3wp.exe processes.
// It queries the WMI for processes with the name 'w3wp.exe' and extracts the app pool name from the command line.
// The process map is a map where the key is the process ID and the value is the app pool name.
// The function runs indefinitely with the specified interval between each query.
func (w *WinW3wp) updateProcessMap(processMap map[uint32]string, interval time.Duration) {
	var processes []Win32_Process
	pq := wmi.CreateQuery(&processes, "WHERE Name = 'w3wp.exe'")

	for {
		err := w.WmiClient.Query(pq, &processes)
		if err != nil {
			w.Log.Error("Error querying WMI: ", err)
		} else {
			for _, p := range processes {
				// Check if the command line contains "-ap"
				if strings.Contains(p.CommandLine, "-ap") {
					// Extract the app pool name from the command line
					appPoolName := getAppPoolNameFromCommandLine(p.CommandLine)
					processMap[p.ProcessId] = appPoolName
				}
			}
		}

		time.Sleep(interval)
	}
}

// getAppPoolNameFromCommandLine extracts the application pool name from the given command line string.
// It searches for the "-ap" flag and retrieves the value enclosed in double quotes following the flag.
// If the flag or the value is not found, it returns an empty string.
func getAppPoolNameFromCommandLine(commandLine string) string {
	// Find the index of "-ap"
	apIndex := strings.Index(commandLine, "-ap")
	if apIndex == -1 {
		return ""
	}

	// Find the start and end indices of the app pool name
	startIndex := apIndex + len("-ap") + 2 // Add 2 to skip the space and the opening quote
	endIndex := strings.Index(commandLine[startIndex:], "\"") + startIndex

	// Extract the app pool name
	appPoolName := commandLine[startIndex:endIndex]

	return appPoolName
}

// updateMetrics updates the metrics for the WinW3wp plugin.
// It queries the WMI for performance data of processes and filters the processes based on the processMap.
// The filtered processes are then sent to the metricsChan for further processing.
//
// Parameters:
//   - processMap: A map containing the process IDs and their corresponding names.
//   - interval: The time interval between each query to the WMI.
//   - metricsChan: A channel to send the filtered processes to.
//
// This function runs in a loop and periodically queries the WMI for performance data.
// If there is an error during the query, it logs the error.
// The function sleeps for the specified interval before making the next query.
func (w *WinW3wp) updateMetrics(processMap map[uint32]string, interval time.Duration, metricsChan chan []Win32_PerfFormattedData_PerfProc_Process) {
	var dst []Win32_PerfFormattedData_PerfProc_Process
	q := wmi.CreateQuery(&dst, "")

	for {
		err := w.WmiClient.Query(q, &dst)
		if err != nil {
			w.Log.Error("Error querying WMI: ", err)
		} else {
			// Filter the processes to only include ones that are in the processMap
			filteredProcesses := make([]Win32_PerfFormattedData_PerfProc_Process, 0, len(dst))
			for _, process := range dst {
				if _, ok := processMap[process.IDProcess]; ok {
					filteredProcesses = append(filteredProcesses, process)
				}
			}

			// Send the filtered processes to the metricsChan
			metricsChan <- filteredProcesses
		}

		time.Sleep(interval)
	}
}

// updateAppPoolMetrics updates the metrics for the WinW3wp plugin.
// It receives a map of process IDs to app pool names and a channel of metrics.
// For each metric received, it looks up the app pool name in the processMap using the process ID.
// If the process ID is not found in the processMap, the metric is skipped.
// Otherwise, it updates the app pool metric with the received values.
func (w *WinW3wp) updateAppPoolMetrics(processMap map[uint32]string, metricsChan chan []Win32_PerfFormattedData_PerfProc_Process) {
	for metrics := range metricsChan {
		for _, metric := range metrics {
			// Look up the app pool name in the processMap using the process ID
			appPoolName, ok := processMap[metric.IDProcess]
			if !ok {
				// If the process ID isn't in the processMap, skip this metric
				continue
			}

			if _, ok := w.AppPoolMetrics[appPoolName]; !ok {
				w.AppPoolMetrics[appPoolName] = &AppPoolMetric{}
			}

			// Update the app pool metric
			w.AppPoolMetrics[appPoolName].percentProcessorTime = float64(metric.PercentProcessorTime)
			w.AppPoolMetrics[appPoolName].privateBytes = metric.PrivateBytes
			w.AppPoolMetrics[appPoolName].workingSet = metric.WorkingSet
			w.AppPoolMetrics[appPoolName].handleCount = metric.HandleCount
			w.AppPoolMetrics[appPoolName].threadCount = metric.ThreadCount
			w.AppPoolMetrics[appPoolName].ioReadBytesPerSec = metric.IOReadBytesPerSec
			w.AppPoolMetrics[appPoolName].ioWriteBytesPerSec = metric.IOWriteBytesPerSec
		}
	}
}

// Gather collects metrics from the Windows IIS worker process (w3wp) for each configured application pool.
// It adds the collected metrics to the provided Accumulator.
//
// The Gather method implements the telegraf.Input interface and is called by Telegraf to collect metrics.
// For each application pool, it creates tags with the appPool name and adds fields with the corresponding metrics.
// The metrics collected include Processor Time, Private Bytes, Working Set, Handle Count, Thread Count,
// IO Read Bytes per Second, and IO Write Bytes per Second.
//
// The gathered metrics are added to the Accumulator using the "w3wp" measurement name.
//
// This method returns nil to indicate that the gathering process completed successfully.
func (w *WinW3wp) Gather(acc telegraf.Accumulator) error {
	for appPoolName, metrics := range w.AppPoolMetrics {
		tags := map[string]string{
			"appPool": appPoolName,
		}
		fields := map[string]interface{}{
			"Processor_Time": metrics.percentProcessorTime,
			"Private_Bytes":  metrics.privateBytes,
			"Working_Set":    metrics.workingSet,
			"Handle_Count":   metrics.handleCount,
			"Thread_Count":   metrics.threadCount,
			"IO_Read":        metrics.ioReadBytesPerSec,
			"IO_Write":       metrics.ioWriteBytesPerSec,
		}
		acc.AddFields("w3wp", fields, tags)
	}
	return nil
}

// init registers the "win_w3wp" input plugin with the Telegraf inputs registry.
func init() {
	inputs.Add("win_w3wp", func() telegraf.Input { return &WinW3wp{} })
}
