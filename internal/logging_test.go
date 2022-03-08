package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLogger(t *testing.T) {
	const layout = time.RFC3339Nano

	for _, tt := range []struct {
		name            string
		lvl             int
		expectedLogData []map[string]string
		expectCaller    bool
	}{
		{
			"debug level",
			2,
			[]map[string]string{
				{"level": "debug", "msg": "Configured logger", "effective_level": "debug"},
				{"level": "debug", "msg": "debug test"},
				{"level": "info", "msg": "info test"},
				{"level": "warn", "msg": "warn test"},
				{"level": "error", "msg": "error test"},
			},
			true,
		},
		{
			"info level",
			1,
			[]map[string]string{
				{"level": "info", "msg": "info test"},
				{"level": "warn", "msg": "warn test"},
				{"level": "error", "msg": "error test"},
			},
			false,
		},
		{
			"error level",
			0,
			[]map[string]string{
				{"level": "warn", "msg": "warn test"},
				{"level": "error", "msg": "error test"},
			},
			false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()
			buf := bytes.NewBuffer([]byte{})
			logger := ConfigureLogger(true, tt.lvl, buf)
			level.Debug(logger).Log("msg", "debug test")
			level.Info(logger).Log("msg", "info test")
			level.Warn(logger).Log("msg", "warn test")
			level.Error(logger).Log("msg", "error test")
			endTime := time.Now()

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			require.Len(t, lines, len(tt.expectedLogData),
				"Expected %d lines of log data but found %d", len(tt.expectedLogData), len(lines))

			for lineNo, line := range lines {
				t.Run(fmt.Sprintf("line:%d", lineNo), func(t *testing.T) {
					expectedData := tt.expectedLogData[lineNo]
					logData := map[string]string{}
					require.NoError(t, json.Unmarshal([]byte(line), &logData),
						"Error parsing log data as JSON on line %d: %q", lineNo, line)

					for key, expected := range expectedData {
						actual := logData[key]
						t.Run(key, func(t *testing.T) {
							assert.Equal(t, expected, actual,
								"Unexpected value for key %q in logged data on line %d", key, lineNo)
						})
					}

					t.Run("ts", func(t *testing.T) {
						ts, err := time.Parse(layout, logData["ts"])
						require.NoError(t, err,
							"error parsing timestamp %s with layout %s on line %d", logData["ts"], layout, lineNo)
						assert.False(t, ts.Before(startTime),
							"logged timestamp %s is earlier than expected (should be after %s)", ts, startTime)
						assert.False(t, ts.After(endTime),
							"logged timestamp %s is later than expected (should be before %s)", ts, endTime)
					})
					t.Run("caller", func(t *testing.T) {
						// Expect "caller" to be included only when lvl>=2 (DEBUG)
						if tt.expectCaller {
							assert.Contains(t, logData, "caller",
								"missing \"caller\" in logged data")
						} else {
							assert.NotContains(t, logData, "caller",
								"unexpected \"caller\" present in logged data")
						}
					})
				})
			}
		})
	}
}
