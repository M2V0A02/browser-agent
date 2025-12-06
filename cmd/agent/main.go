// cmd/agent/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"browser-agent/internal/agents"
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"browser-agent/internal/infrastructure/env"
	"browser-agent/internal/infrastructure/logger"
)

func main() {
	envService := env.NewEnvService()

	taskLogger, err := logger.NewLogger("launch")
	if err != nil {
		log.Fatal("Не удалось создать логгер: ", err)
	}
	defer taskLogger.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	browser, err := rodwrapper.NewBrowser(ctx)
	if err != nil {
		taskLogger.Logf("Ошибка запуска браузера: %v", err)
		log.Fatal(err)
	}
	defer browser.Close()

	agent, err := agents.NewModernAgent(browser, envService, taskLogger)
	if err != nil {
		taskLogger.Logf("Ошибка создания агента: %v", err)
		log.Fatal(err)
	}

	fmt.Println("\nВведите задачу для агента:")
	reader := bufio.NewReader(os.Stdin)
	task, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Ошибка чтения ввода: ", err)
	}
	// Убираем символ перевода строки
	task = strings.TrimSpace(task)

	taskLogger.Logf("ЗАДАЧА: %s", task)
	fmt.Println("\nАгент начал работу...")

	result, err := agent.Execute(ctx, task)
	if err != nil {
		taskLogger.Logf("ОШИБКА: %v", err)
		fmt.Printf("\nОшибка выполнения: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nФИНАЛЬНЫЙ ОТВЕТ:")
	fmt.Println(result)
	taskLogger.Logf("УСПЕШНО ЗАВЕРШЕНО")
}
