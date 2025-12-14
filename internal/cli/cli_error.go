package cli

// CLIError is a structured error used for consistent NDJSON/text emission.
type CLIError struct {
	Code    string
	Message string
	Hint    string
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
