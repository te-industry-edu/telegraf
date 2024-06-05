//go:generate ../../../tools/readme_config_includer/generator
//go:build !windows

package win_w3wp

import (
	_ "embed"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

type WinW3wp struct {
	Log telegraf.Logger `toml:"-"`
}

func (w *WinW3wp) Init() error {
	w.Log.Warn("current platform is not supported")
	return nil
}
func (w *WinW3wp) SampleConfig() string                { return sampleConfig }
func (w *WinW3wp) Gather(_ telegraf.Accumulator) error { return nil }

func init() {
	inputs.Add("win_w3wp", func() telegraf.Input {
		return &WinW3wp{}
	})
}
