//go:build windows && (!custom || inputs || inputs.win_w3wp)

package all

import _ "github.com/influxdata/telegraf/plugins/inputs/win_w3wp" // register plugin
