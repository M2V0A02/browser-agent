package env

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type EnvService struct{}

func NewEnvService() *EnvService {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "dev"
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Info: no .env file with secrets found (this is OK for CI/CD)")
	}

	envFile := fmt.Sprintf(".env.%s", appEnv)
	if err := godotenv.Overload(envFile); err != nil {
		log.Printf("Warning: could not load %s: %v", envFile, err)
	}

	log.Printf("Environment loaded: APP_ENV=%s", appEnv)

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
