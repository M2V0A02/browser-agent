package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"browser-agent/internal/di"
	"browser-agent/internal/infrastructure/env"
)

func main() {
	envService := env.NewEnvService()

	fmt.Println("\nВведите задачу для агента:")
	reader := bufio.NewReader(os.Stdin)
	task, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Ошибка чтения ввода: ", err)
	}
	task = strings.TrimSpace(task)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	thinkingMode := envService.GetBool("THINKING_MODE", true)
	thinkingBudget := envService.GetInt("THINKING_BUDGET", 10000)

	container, err := di.NewContainer(ctx, di.Config{
		OpenRouterAPIKey: envService.MustGet("OPENROUTER_API_KEY"),
		OpenRouterModel:  envService.MustGet("OPENROUTER_MODEL_NAME"),
		BrowserHeadless:  false,
		ThinkingMode:     thinkingMode,
		ThinkingBudget:   thinkingBudget,
	})
	if err != nil {
		log.Fatalf("Ошибка инициализации: %v", err)
	}
	defer container.Close()

	container.Logger.Info("Task started", "task", task)
	fmt.Println("\nАгент начал работу...")

	result, err := container.TaskExecutor.Execute(ctx, task)
	if err != nil {
		container.Logger.Error("Task failed", "error", err)
		fmt.Printf("\nОшибка выполнения: %v\n", err)
		os.Exit(1)
	}

	container.Logger.Info("Task completed", "iterations", result.Iterations)
	fmt.Println("\nФИНАЛЬНЫЙ ОТВЕТ:")
	fmt.Println(result.FinalAnswer)
}
