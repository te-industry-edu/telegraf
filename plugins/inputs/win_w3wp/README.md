# Windows IIS App Pool Process Input Plugin

**Disclaimer:** This plugin only supports Windows.

This plugin collects metrics from Windows IIS App Pool processes. 

The following metrics are collected:

- percentProcessorTime - The percentage of processor time used by the application pool.
- privateBytes - The amount of private memory allocated for the application pool.
- workingSet - The amount of physical memory allocated for the application pool.
- handleCount - The number of handles currently open by the application pool.
- threadCount - The number of threads currently active in the application pool.
- ioReadBytesPerSec - The rate of input/output read operations per second for the application pool.
- ioWriteBytesPerSec - The rate of input/output write operations per second for the application pool.

The plugin uses the Windows Performance Counters to collect these metrics.

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

In addition to the plugin-specific configuration settings, plugins support
additional global and plugin configuration settings. These settings are used to
modify metrics, tags, and field or create aliases and configure ordering, etc.
See the [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Configuration

```toml @sample.conf
# Input plugin to collect Windows IIS App Pool metrics
# This plugin ONLY supports Windows
[[inputs.win_w3wp]]
  # The collection interval. Accepts a duration string (e.g., "500ms", "2.5s", "1m").
  interval = "30s"
```

## Metrics

The following metrics are collected:

- percentProcessorTime - The percentage of processor time used by the application pool.
- privateBytes - The amount of private memory allocated for the application pool.
- workingSet - The amount of physical memory allocated for the application pool.
- handleCount - The number of handles currently open by the application pool.
- threadCount - The number of threads currently active in the application pool.
- ioReadBytesPerSec - The rate of input/output read operations per second for the application pool.
- ioWriteBytesPerSec - The rate of input/output write operations per second for the application pool.

Each metric is tagged with the application pool name as `appPool`.

## Example Output

