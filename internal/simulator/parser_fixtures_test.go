package simulator

import (
	"bufio"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParserFixtures(t *testing.T) {
	f, err := os.Open("testdata/parser_fixtures.ndjson")
	require.NoError(t, err)
	defer f.Close()

	type want struct {
		level     string
		process   string
		pid       int
		subsystem string
		category  string
		message   string
		ts        string
		nilEntry  bool
	}

	wants := []want{
		{
			level:     "Info",
			process:   "MyApp",
			pid:       123,
			subsystem: "com.example.app",
			category:  "ui",
			message:   "Hello (no-colon offset, fractional)",
			ts:        "2025-12-08T22:11:55.808033+01:00",
		},
		{
			level:     "Error",
			process:   "MyDaemon",
			pid:       456,
			subsystem: "com.example.daemon",
			category:  "net",
			message:   "Hello (colon offset, no fractional)",
			ts:        "2025-12-08T22:11:55+01:00",
		},
		{nilEntry: true},
		{
			level:     "Default",
			process:   "Other",
			pid:       321,
			subsystem: "com.example.other",
			category:  "misc",
			message:   "Fallback to formatString",
			ts:        "2025-12-08T22:11:55.000001Z",
		},
	}

	p := NewParser()
	sc := bufio.NewScanner(f)
	i := 0
	for sc.Scan() {
		line := sc.Bytes()
		entry, err := p.Parse(line)
		require.NoError(t, err)
		require.Less(t, i, len(wants), "fixture count mismatch")
		w := wants[i]
		i++

		if w.nilEntry {
			require.Nil(t, entry)
			continue
		}
		require.NotNil(t, entry)
		require.Equal(t, w.level, string(entry.Level))
		require.Equal(t, w.process, entry.Process)
		require.Equal(t, w.pid, entry.PID)
		require.Equal(t, w.subsystem, entry.Subsystem)
		require.Equal(t, w.category, entry.Category)
		require.Equal(t, w.message, entry.Message)
		require.Equal(t, w.ts, entry.Timestamp.Format(time.RFC3339Nano))
	}
	require.NoError(t, sc.Err())
	require.Equal(t, len(wants), i, "fixture count mismatch")
}
