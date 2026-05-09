package groxy

// Logger is used by Groxy to write log messages.
//
// It matches the Printf method provided by the standard library's log.Logger.
type Logger interface {
	Printf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Printf(format string, args ...any) {}

func resolveLogger(logger Logger) Logger {
	if logger == nil {
		return noopLogger{}
	}

	return logger
}
