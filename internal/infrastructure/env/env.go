package env

import (
	"log"
	"os"
	"strconv"

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

func (e *EnvService) GetBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func (e *EnvService) GetInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}
