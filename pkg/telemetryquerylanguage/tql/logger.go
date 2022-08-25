package tql

type Logger interface {
	WithFields(field map[string]interface{}) Logger
	Info(msg string)
	Error(msg string)
}

type NoOpLogger struct{}

func (nol NoOpLogger) WithFields(field map[string]interface{}) Logger {
	return nol
}

func (nol NoOpLogger) Info(msg string) {
	// no-op
}

func (nol NoOpLogger) Error(msg string) {
	// no-op
}
