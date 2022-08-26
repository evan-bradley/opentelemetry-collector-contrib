package tql

// Logger allows printing logs inside TQL functions using a
// logging framework provided by the component using the TQL.
type Logger interface {
	WithFields(field map[string]any) Logger
	Info(msg string)
	Error(msg string)
}

type NoOpLogger struct{}

func (nol NoOpLogger) WithFields(field map[string]any) Logger {
	return nol
}

func (nol NoOpLogger) Info(msg string) {
	// no-op
}

func (nol NoOpLogger) Error(msg string) {
	// no-op
}
