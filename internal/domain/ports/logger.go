package ports

type Logger interface {
	Logf(format string, args ...any)
	Close() error
}
