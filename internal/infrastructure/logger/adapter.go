package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"browser-agent/internal/application/port/output"
)

var _ output.LoggerPort = (*LoggerAdapter)(nil)

type LoggerAdapter struct {
	file   *os.File
	fields map[string]any
}

func NewLoggerAdapter(taskName string) (*LoggerAdapter, error) {
	safeName := sanitize(taskName)
	filename := fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02_15-04-05"), safeName)

	if err := os.MkdirAll("log", 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	file, err := os.Create(filepath.Join("log", filename))
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	return &LoggerAdapter{
		file:   file,
		fields: make(map[string]any),
	}, nil
}

func (l *LoggerAdapter) log(level, msg string, args ...any) {
	entry := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	for k, v := range l.fields {
		entry[k] = v
	}

	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			entry[key] = args[i+1]
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.file, `{"timestamp":"%s","level":"ERROR","message":"marshal error: %v"}`+"\n",
			time.Now().Format(time.RFC3339), err)
		return
	}

	l.file.Write(data)
	l.file.WriteString("\n")
}

func (l *LoggerAdapter) Debug(msg string, args ...any) {
	l.log("DEBUG", msg, args...)
}

func (l *LoggerAdapter) Info(msg string, args ...any) {
	l.log("INFO", msg, args...)
}

func (l *LoggerAdapter) Warn(msg string, args ...any) {
	l.log("WARN", msg, args...)
}

func (l *LoggerAdapter) Error(msg string, args ...any) {
	l.log("ERROR", msg, args...)
}

func (l *LoggerAdapter) WithField(key string, value any) output.LoggerPort {
	newFields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &LoggerAdapter{
		file:   l.file,
		fields: newFields,
	}
}

func (l *LoggerAdapter) WithFields(fields map[string]any) output.LoggerPort {
	newFields := make(map[string]any, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &LoggerAdapter{
		file:   l.file,
		fields: newFields,
	}
}

func (l *LoggerAdapter) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

func sanitize(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	s = string(result)
	if s == "" {
		return "task"
	}
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
