package ports

type EnvService interface {
	Get(key string) string
	MustGet(key string) string
}
