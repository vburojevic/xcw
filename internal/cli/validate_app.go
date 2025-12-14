package cli

func validateAppPredicateAll(app, predicate string, all bool, hasSourceFilter bool) *CLIError {
	if app != "" {
		return nil
	}
	if predicate != "" {
		return nil
	}
	if hasSourceFilter {
		return nil
	}
	if all {
		return nil
	}
	return &CLIError{
		Code:    "FILTER_REQUIRED",
		Message: "--app is required unless --predicate or --all is provided",
		Hint:    "Add -a/--app, or provide --predicate, or add --all to intentionally stream/query all simulator logs",
	}
}
