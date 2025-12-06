// internal/usecase/run_agent.go
package usecase

/*
import (
	"context"
	"fmt"
	"time"

	"browser-agent/internal/agent"
	"browser-agent/internal/ports"

	"github.com/go-rod/rod"
)

type RunAgentUseCase struct {
	page     *rodwrapper.Page
	env      ports.EnvService
	loggerFn func(task string) (ports.Logger, error)
}

func NewRunAgentUseCase(
	page *rodwrapper.Page,
	env ports.EnvService,
	loggerFn func(task string) (ports.Logger, error),
) ports.AIAgent {
	return &RunAgentUseCase{
		page:     page,
		env:      env,
		loggerFn: loggerFn,
	}
}

func (uc *RunAgentUseCase) Execute(ctx context.Context, task string) (string, error) {
	// 1. Создаём логгер для этой задачи
	taskLogger, err := uc.loggerFn(task)
	if err != nil {
		return "", fmt.Errorf("не удалось создать лог-файл: %w", err)
	}
	defer func() {
		_ = taskLogger.Close()
	}()

	// 2. Пишем заголовок в лог
	taskLogger.Logf("=== ЗАДАЧА ЗАПУЩЕНА ===\n")
	taskLogger.Logf("Время: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	taskLogger.Logf("Задача: %s\n\n", task)

	// 3. Создаём BrowserCore (обёртка над page + логгер)
	browserCore := agent.NewBrowserCore(uc.page, taskLogger)

	// 4. Создаём ModernAgent (всё ещё на langchaingo, но теперь с DI)
	modernAgent, err := agent.NewModernAgent(browserCore, uc.env, taskLogger)
	if err != nil {
		taskLogger.Logf("ОШИБКА создания агента: %v\n", err)
		return "", err
	}

	// 5. Запускаем выполнение
	taskLogger.Logf("Запуск ModernAgent.Execute...\n")
	finalAnswer, err := modernAgent.Execute(ctx, task)
	if err != nil {
		taskLogger.Logf("ОШИБКА выполнения задачи: %v\n", err)
		return "", err
	}

	// 6. Финальный ответ
	taskLogger.Logf("\n=== ЗАДАЧА УСПЕШНО ЗАВЕРШЕНА ===\n")
	taskLogger.Logf("Финальный ответ:\n%s\n", finalAnswer)

	return finalAnswer, nil
}
*/
