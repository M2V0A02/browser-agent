.PHONY: build run test clean install help

BINARY_NAME=ai-agent
BUILD_DIR=build
MAIN_PATH=cmd/agent/main.go

help:
	@echo "Доступные команды:"
	@echo "  make build    - Собрать бинарный файл"
	@echo "  make run      - Запустить агента"
	@echo "  make test     - Запустить тесты"
	@echo "  make clean    - Очистить собранные файлы"
	@echo "  make install  - Установить в \$$GOPATH/bin"

build:
	@echo "Сборка $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Бинарник создан: $(BUILD_DIR)/$(BINARY_NAME)"

run:
	@go run $(MAIN_PATH)

test:
	@echo "Запуск тестов..."
	@go test ./... -v

test-coverage:
	@echo "Запуск тестов с покрытием..."
	@go test ./... -coverprofile=coverage.out
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
