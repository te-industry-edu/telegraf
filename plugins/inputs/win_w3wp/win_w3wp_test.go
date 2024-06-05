//go:build windows

package win_w3wp_test

import (
	"fmt"
	"testing"

	"github.com/influxdata/telegraf/plugins/inputs/win_w3wp"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

// MockWMI is a mock for the wmi package.
type MockWMI struct{}

// Query returns fake metrics for testing.
func (m *MockWMI) Query(query string, dst interface{}, connectServerArgs ...interface{}) error {
	// Cast the destination interface to the correct type.
	d, ok := dst.(*[]win_w3wp.Win32_PerfFormattedData_PerfProc_Process)
	if !ok {
		return fmt.Errorf("invalid type passed to Query")
	}

	// Create fake metrics.
	*d = []win_w3wp.Win32_PerfFormattedData_PerfProc_Process{
		{
			IDProcess:            1,
			Name:                 "test",
			PercentProcessorTime: 50,
			PrivateBytes:         1000,
			WorkingSet:           2000,
			HandleCount:          10,
			ThreadCount:          20,
			IOReadBytesPerSec:    3000,
			IOWriteBytesPerSec:   4000,
		},
	}

	return nil
}

func TestWinW3wp(t *testing.T) {
	// Create a new WinW3wp instance using the MockWMI client
	// to be able to use fake performance metrics when testing.
	w := win_w3wp.WinW3wp{
		WmiClient: &MockWMI{},
	}

	require.NoError(t, w.Init())
	acc := testutil.Accumulator{}
	require.NoError(t, w.Gather(&acc))

	// Check the gathered metrics against the expected values.
	require.Equal(t, 1, len(acc.Metrics))
	m := acc.Metrics[0]
	require.Equal(t, "w3wp", m.Measurement)
	require.Equal(t, map[string]interface{}{
		"Processor_Time": 50.0,
		"Private_Bytes":  1000,
		"Working_Set":    2000,
		"Handle_Count":   10,
		"Thread_Count":   20,
		"IO_Read":        3000,
		"IO_Write":       4000,
	}, m.Fields)
}
