package internal

import (
	"io"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// ConfigureLogger creates a new Logger for writing structured logs to w.
// When json is true, log output will be JSON-formatted; when false, logfmt format is used.
// lvl indicates the effective log level; numeric values correspond to log levels as-follows:
// 0 = WARN | 1 = INFO | 2 = DEBUG. Any value higher than 2 will be DEBUG.
// In addition to fields defined on a per-log basis, this function configures a "caller" field included
// on all logged output when lvl >= 2.
func ConfigureLogger(json bool, lvl int, w io.Writer) (l log.Logger) {
	if json {
		l = log.NewJSONLogger(w)
	} else {
		l = log.NewLogfmtLogger(w)
	}
	l = log.WithSuffix(l, "ts", log.DefaultTimestamp)

	var lvlValue level.Value
	if lvl >= 2 {
		l = log.WithSuffix(level.NewFilter(l, level.AllowDebug()), "caller", log.DefaultCaller)
		lvlValue = level.DebugValue()
	} else if lvl == 1 {
		l = level.NewFilter(l, level.AllowInfo())
		lvlValue = level.InfoValue()
	} else {
		l = level.NewFilter(l, level.AllowWarn())
		lvlValue = level.WarnValue()
	}

	l = log.NewSyncLogger(l)
	level.Debug(l).Log("msg", "Configured logger", "effective_level", lvlValue.String())
	return
}
