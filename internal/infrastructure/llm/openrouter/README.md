# OpenRouter Adapter

Адаптер для работы с OpenRouter API через библиотеку go-openai.

## Особенности

### Reasoning Content Fix

OpenRouter API возвращает поле `"reasoning"` для моделей с reasoning capabilities (например, o1, o1-preview).
Однако библиотека go-openai ожидает поле `"reasoning_content"`.

Для совместимости адаптер автоматически заменяет `"reasoning":` на `"reasoning_content":` в ответах API.

Реализация:
- Чтение полного ответа в память
- Замена всех вхождений через `strings.ReplaceAll`
- Создание нового reader с исправленным содержимым

Это решает проблему с паникой `slice bounds out of range`, которая возникала при попытке замены "на лету" в буфере фиксированного размера.

## Конфигурация

```go
cfg := openrouter.DefaultConfig(apiKey, model)
cfg.ThinkingMode = true
cfg.ThinkingBudget = 10000  // max tokens for reasoning
cfg.Logger = logger

adapter := openrouter.NewOpenRouterAdapter(cfg)
```

## Логирование

Адаптер логирует:
- HTTP запросы (method, URL, body)
- Ошибки при выполнении запросов
- Предупреждения о пустых ответах

Логирование включается через параметр `Logger` в конфигурации.
