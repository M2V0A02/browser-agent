.PHONY: build run test clean install help

BINARY_NAME=ai-agent
BUILD_DIR=build
MAIN_PATH=cmd/agent/main.go
APP_ENV ?= dev

help:
	@echo "Доступные команды:"
	@echo "  make build       - Собрать бинарный файл"
	@echo "  make run         - Запустить агента в dev режиме (APP_ENV=dev)"
	@echo "  make run-prod    - Запустить собранный бинарник в prod режиме (APP_ENV=prod)"
	@echo "  make test        - Запустить тесты (APP_ENV=test)"
	@echo "  make clean       - Очистить собранные файлы"
	@echo "  make install     - Установить в \$$GOPATH/bin"
	@echo ""
	@echo "Переменные окружения:"
	@echo "  APP_ENV=<env>    - Выбрать окружение (dev/test/prod)"

build:
	@echo "Сборка $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Бинарник создан: $(BUILD_DIR)/$(BINARY_NAME)"

run:
	@APP_ENV=dev go run $(MAIN_PATH)

run-prod:
	@echo "Запуск в production режиме..."
	@APP_ENV=prod $(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "Запуск тестов..."
	@APP_ENV=test go test ./... -v

test-coverage:
	@echo "Запуск тестов с покрытием..."
	@APP_ENV=test go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Отчет о покрытии: coverage.html"

clean:
	@echo "Очистка..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Очистка завершена"

install: build
	@echo "Установка в \$$GOPATH/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Установлено: $(GOPATH)/bin/$(BINARY_NAME)"

deps:
	@echo "Загрузка зависимостей..."
	@go mod download
	@go mod tidy
	@echo "✓ Зависимости загружены"
