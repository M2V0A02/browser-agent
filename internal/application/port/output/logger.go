package output

type LoggerPort interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	WithField(key string, value any) LoggerPort
	WithFields(fields map[string]any) LoggerPort

	Close() error
}
