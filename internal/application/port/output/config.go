package output

type ConfigPort interface {
	Get(key string) string
	MustGet(key string) string
	GetWithDefault(key string, defaultValue string) string
}
