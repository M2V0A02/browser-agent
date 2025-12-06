package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-agent/internal/domain/ports"
)

// LogEntry представляет одну структурированную запись в логе.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`     // INFO, ERROR, DEBUG
	Component string    `json:"component"` // "agent", "tool", "http", "system"
	ToolName  string    `json:"tool_name,omitempty"`
	Message   string    `json:"message"`
	Input     string    `json:"input,omitempty"`
	Output    string    `json:"output,omitempty"`
	Duration  int64     `json:"duration_ms,omitempty"`
	PageURL   string    `json:"page_url,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// JSONLogger реализует ports.Logger и пишет структурированные логи.
type JSONLogger struct {
	file   *os.File
	prefix string // префикс для сообщений (упрощает миграцию старого кода)
}

// NewJSONLogger создаёт новый логгер для задачи.
// Имя файла: timestamp_safeTaskName.log в папке ./log/
func NewJSONLogger(task string) (ports.Logger, error) {
	safeName := sanitize(task)
	filename := fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02_15-04-05"), safeName)

	if err := os.MkdirAll("log", 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	file, err := os.Create(filepath.Join("log", filename))
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	return &JSONLogger{
		file: file,
	}, nil
}

// logInternal записывает одну JSON-строку в файл.
func (l *JSONLogger) logInternal(entry *LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		// fallback: пишем текстовую строку в случае ошибки сериализации
		fmt.Fprintf(l.file, `{"timestamp":"%s","level":"ERROR","component":"logger","message":"failed to marshal log entry: %s"}`+"\n",
			time.Now().Format(time.RFC3339), strings.ReplaceAll(err.Error(), `"`, `\"`))
		return
	}
	l.file.Write(data)
	l.file.WriteString("\n")
}

// Logf реализует ports.Logger. Сохраняет обратную совместимость со старым кодом.
func (l *JSONLogger) Logf(format string, args ...any) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Component: "system",
		Message:   fmt.Sprintf(format, args...),
	}
	l.logInternal(entry)
}

// Close реализует ports.Logger.
func (l *JSONLogger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Helper-методы для структурированного логирования

// LogToolCall логирует вызов инструмента.
func (l *JSONLogger) LogToolCall(toolName, input string, start time.Time, pageURL string) {
	duration := time.Since(start).Milliseconds()
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Component: "tool",
		ToolName:  toolName,
		Message:   "tool called",
		Input:     input,
		Duration:  duration,
		PageURL:   pageURL,
	}
	l.logInternal(entry)
}

// LogToolResult логирует успешный результат работы инструмента.
func (l *JSONLogger) LogToolResult(toolName, output string, start time.Time, pageURL string) {
	duration := time.Since(start).Milliseconds()
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Component: "tool",
		ToolName:  toolName,
		Message:   "tool completed",
		Output:    output,
		Duration:  duration,
		PageURL:   pageURL,
	}
	l.logInternal(entry)
}

// LogToolError логирует ошибку инструмента.
func (l *JSONLogger) LogToolError(toolName, input string, err error, start time.Time, pageURL string) {
	duration := time.Since(start).Milliseconds()
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Component: "tool",
		ToolName:  toolName,
		Message:   "tool failed",
		Input:     input,
		Error:     err.Error(),
		Duration:  duration,
		PageURL:   pageURL,
	}
	l.logInternal(entry)
}

// LogAgentCall логирует начало выполнения задачи агентом.
func (l *JSONLogger) LogAgentCall(task string) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Component: "agent",
		Message:   "agent execution started",
		Input:     task,
	}
	l.logInternal(entry)
}

// LogAgentResult логирует успешное завершение работы агента.
func (l *JSONLogger) LogAgentResult(finalAnswer string, start time.Time) {
	duration := time.Since(start).Milliseconds()
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Component: "agent",
		Message:   "agent execution completed",
		Output:    finalAnswer,
		Duration:  duration,
	}
	l.logInternal(entry)
}

// LogAgentError логирует ошибку агента.
func (l *JSONLogger) LogAgentError(task string, err error, start time.Time) {
	duration := time.Since(start).Milliseconds()
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Component: "agent",
		Message:   "agent execution failed",
		Input:     task,
		Error:     err.Error(),
		Duration:  duration,
	}
	l.logInternal(entry)
}

// sanitize — делает имя задачи безопасным для файловой системы (копия из старого логгера).
func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
	s = strings.Trim(s, "_")
	if s == "" {
		return "task"
	}
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func NewLogger(task string) (ports.Logger, error) {
	return NewJSONLogger(task)
}
