package simulator

import "testing"

func BenchmarkParserParse(b *testing.B) {
	p := NewParser()
	line := []byte(`{"timestamp":"2025-12-08 22:11:55.808033+0100","messageType":"Error","eventType":"logEvent","eventMessage":"Connection failed","processID":1234,"processImagePath":"/Applications/MyApp.app/MyApp","processImageUUID":"UUID","subsystem":"com.example.myapp","category":"network","threadID":1}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Parse(line); err != nil {
			b.Fatal(err)
		}
	}
}
