package userinteraction

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"browser-agent/internal/application/port/output"
	"github.com/fatih/color"
)

var _ output.UserInteractionPort = (*ConsoleUserInteraction)(nil)

type ConsoleUserInteraction struct {
	reader *bufio.Reader
}

func NewConsoleUserInteraction() *ConsoleUserInteraction {
	return &ConsoleUserInteraction{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (u *ConsoleUserInteraction) AskQuestion(ctx context.Context, question string) (string, error) {
	fmt.Printf("\n[USER INPUT REQUIRED] %s\n> ", question)

	answer, err := u.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	return strings.TrimSpace(answer), nil
}

func (u *ConsoleUserInteraction) WaitForUserAction(ctx context.Context, message string) error {
	fmt.Printf("\n[USER ACTION REQUIRED] %s\n", message)
	fmt.Print("Press Enter when done...")

	_, err := u.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to wait for user: %w", err)
	}

	return nil
}

func (u *ConsoleUserInteraction) ShowIteration(ctx context.Context, iteration, maxIterations int) {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Printf("\n━━━ Итерация %d/%d ━━━\n", iteration, maxIterations)
}

func (u *ConsoleUserInteraction) ShowThinking(ctx context.Context, content string) {
	if content == "" {
		return
	}

	blue := color.New(color.FgBlue)
	blue.Print("\n💭 Размышление: ")

	dim := color.New(color.Faint)
	truncated := truncate(content, 500)
	dim.Println(truncated)
}

func (u *ConsoleUserInteraction) ShowToolStart(ctx context.Context, toolName, arguments string) {
	icon, name := getToolDisplay(toolName)

	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Printf("\n%s %s\n", icon, name)

	summary := formatToolArguments(toolName, arguments)
	if summary != "" {
		dim := color.New(color.Faint)
		dim.Printf("   %s\n", summary)
	}
}

func (u *ConsoleUserInteraction) ShowToolResult(ctx context.Context, toolName, result string, isError bool) {
	if isError {
		red := color.New(color.FgRed)
		red.Print("❌ Ошибка: ")

		dim := color.New(color.Faint)
		dim.Println(truncate(result, 300))
		return
	}

	summary := formatToolResult(toolName, result)
	green := color.New(color.FgGreen)
	green.Printf("✓ %s\n", summary)
}

func getToolDisplay(toolName string) (string, string) {
	displays := map[string][2]string{
		"browser_navigate":       {"🌐", "Навигация"},
		"browser_click":          {"🖱️", "Клик"},
		"browser_fill":           {"✏️", "Заполнение"},
		"browser_scroll":         {"📜", "Прокрутка"},
		"browser_screenshot":     {"📸", "Скриншот"},
		"browser_press_enter":    {"⏎", "Enter"},
		"browser_observe":        {"👁️", "Наблюдение"},
		"browser_query_elements": {"🔍", "Извлечение данных"},
		"browser_search":         {"🔎", "Поиск"},
		"run_agent":              {"🤖", "Запуск агента"},
		"user_ask_question":      {"❓", "Вопрос пользователю"},
		"user_wait_action":       {"⏸️", "Ожидание действия"},
	}

	if display, ok := displays[toolName]; ok {
		return display[0], display[1]
	}
	return "🔧", toolName
}

func formatToolArguments(toolName, arguments string) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}

	switch toolName {
	case "browser_navigate":
		if url, ok := args["url"].(string); ok {
			return fmt.Sprintf("URL: %s", url)
		}

	case "browser_click":
		if selector, ok := args["selector"].(string); ok {
			observe := args["observe"]
			if observe == true {
				return fmt.Sprintf("Selector: %s (с наблюдением)", truncate(selector, 60))
			}
			return fmt.Sprintf("Selector: %s", truncate(selector, 60))
		}
		if selectors, ok := args["selectors"].([]interface{}); ok {
			return fmt.Sprintf("Batch: %d элементов", len(selectors))
		}

	case "browser_fill":
		if selector, ok := args["selector"].(string); ok {
			if text, ok := args["text"].(string); ok {
				return fmt.Sprintf("Поле: %s → %s", truncate(selector, 40), truncate(text, 30))
			}
		}
		if fields, ok := args["fields"].(map[string]interface{}); ok {
			return fmt.Sprintf("Batch: %d полей", len(fields))
		}

	case "browser_scroll":
		if direction, ok := args["direction"].(string); ok {
			directions := map[string]string{
				"up":     "⬆️ Вверх",
				"down":   "⬇️ Вниз",
				"top":    "⬆️ В начало",
				"bottom": "⬇️ В конец",
			}
			if display, ok := directions[direction]; ok {
				return display
			}
			return direction
		}

	case "browser_query_elements":
		if selector, ok := args["selector"].(string); ok {
			limit := 20
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			return fmt.Sprintf("Selector: %s (лимит: %d)", truncate(selector, 50), limit)
		}

	case "browser_search":
		searchType, _ := args["type"].(string)
		query, _ := args["query"].(string)
		types := map[string]string{
			"text":      "текст",
			"id":        "ID",
			"attribute": "атрибут",
		}
		if t, ok := types[searchType]; ok {
			return fmt.Sprintf("Тип: %s, Запрос: %s", t, truncate(query, 50))
		}

	case "run_agent":
		agentType, _ := args["agent_type"].(string)
		task, _ := args["task"].(string)
		agentNames := map[string]string{
			"navigation": "Навигация",
			"extraction": "Извлечение",
			"form":       "Формы",
			"analysis":   "Анализ",
		}
		if name, ok := agentNames[agentType]; ok {
			return fmt.Sprintf("Агент: %s | Задача: %s", name, truncate(task, 60))
		}

	case "user_ask_question":
		if question, ok := args["question"].(string); ok {
			return truncate(question, 80)
		}

	case "user_wait_action":
		if message, ok := args["message"].(string); ok {
			return truncate(message, 80)
		}
	}

	return ""
}

func formatToolResult(toolName, result string) string {
	switch toolName {
	case "browser_navigate":
		return result

	case "browser_click":
		if strings.Contains(result, "Successfully clicked") {
			parts := strings.Split(result, " ")
			if len(parts) >= 3 {
				return fmt.Sprintf("Кликнуто элементов: %s", parts[2])
			}
		}
		if strings.Contains(result, "Click successful") {
			lines := strings.Split(result, "\n")
			details := []string{}
			for _, line := range lines {
				if strings.HasPrefix(line, "✓") {
					detail := strings.TrimPrefix(line, "✓ ")
					details = append(details, strings.TrimSpace(detail))
				}
			}
			if len(details) > 0 {
				return fmt.Sprintf("Успешно | %s", strings.Join(details, ", "))
			}
			return "Успешно"
		}
		return result

	case "browser_fill":
		if strings.Contains(result, "Successfully filled") {
			parts := strings.Split(result, " ")
			if len(parts) >= 3 {
				return fmt.Sprintf("Заполнено полей: %s", parts[2])
			}
		}
		return result

	case "browser_scroll":
		return result

	case "browser_screenshot":
		return "Скриншот сделан"

	case "browser_press_enter":
		return "Enter нажат"

	case "browser_observe":
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Visible Elements:") {
				return line
			}
		}
		return "Наблюдение завершено"

	case "browser_query_elements":
		if strings.HasPrefix(result, "Found") {
			lines := strings.Split(result, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}
		return result

	case "browser_search":
		if strings.HasPrefix(result, "Found") {
			lines := strings.Split(result, "\n")
			if len(lines) > 0 {
				firstLine := lines[0]
				if strings.Contains(firstLine, "element(s)") {
					return firstLine
				}
				return truncate(firstLine, 100)
			}
		}
		return truncate(result, 100)

	case "run_agent":
		return truncate(result, 150)

	case "user_ask_question":
		return fmt.Sprintf("Ответ: %s", truncate(result, 80))

	case "user_wait_action":
		return result
	}

	return truncate(result, 100)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
