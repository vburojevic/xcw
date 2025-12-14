package simulator

import "testing"

func FuzzParserParse(f *testing.F) {
	// Seeds: valid-ish JSON, valid fixture line, and junk.
	f.Add(`{"timestamp":"2025-12-08 22:11:55.808033+0100","messageType":"Info","eventType":"logEvent","eventMessage":"hi","processID":1,"processImagePath":"/Applications/MyApp.app/MyApp","threadID":1}`)
	f.Add(`{"timestamp":"2025-12-08T22:11:55Z","messageType":"Debug","eventType":"activityCreateEvent"}`)
	f.Add(`not json`)

	p := NewParser()
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = p.Parse([]byte(s))
	})
}
