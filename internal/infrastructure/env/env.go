package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type EnvService struct{}

func NewEnvService() *EnvService {
	_ = godotenv.Load()
	return &EnvService{}
}

func (e *EnvService) Get(key string) string {
	return os.Getenv(key)
}

func (e *EnvService) MustGet(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("ENV %s is missing", key)
	}
	return val
}
