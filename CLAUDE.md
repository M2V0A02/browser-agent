# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

<<<<<<< HEAD
AI browser automation agent that uses LLM (via OpenRouter API) to autonomously control a browser through natural language tasks. Built in Go using Clean Architecture and Rod for browser automation.
=======
AI browser automation agent that uses LLM (via OpenRouter API) to autonomously control a browser through natural language tasks. Built in Go using Rod for browser automation.

**Note:** Current codebase needs refactoring towards Clean Architecture with proper interfaces, dependency injection, and comprehensive tests.
>>>>>>> 7e1ff8f (Рефакторинг агента: переход на go-openai SDK)

## Build and Run

```bash
# Run the agent
go run cmd/agent/main.go

# Run tests
go test ./...

# Run specific test
<<<<<<< HEAD
go test ./internal/infrastructure/browser/rod -run TestAdapter
=======
go test ./internal/infrastructure/browser/rodwrapper -run TestHTMLCleaner
>>>>>>> 7e1ff8f (Рефакторинг агента: переход на go-openai SDK)
```

## Required Environment Variables

Create `.env` file or export:
- `OPENROUTER_API_KEY` - API key for OpenRouter
- `OPENROUTER_MODEL_NAME` - Model to use (e.g., `anthropic/claude-3.5-sonnet`)

## Architecture

<<<<<<< HEAD
The project follows Clean Architecture with clear separation of concerns:

### Domain Layer (`internal/domain/`)
- `entity/` - Core business entities (`Task`, `Message`, `PageContent`, `UIElement`)
- `valueobject/` - Value objects

### Application Layer (`internal/application/`)
- `port/input/` - Input ports defining use case interfaces
  - `TaskExecutor` - Main interface for executing tasks
- `port/output/` - Output ports defining external service interfaces
  - `BrowserPort` - Browser automation operations
  - `LLMPort` - LLM chat completions
  - `ToolPort` & `ToolRegistry` - Tool system interfaces
  - `LoggerPort` - Logging interface
  - `ConfigPort` - Configuration interface
  - `UserInteractionPort` - User interaction interface
- `service/` - Application services (`ToolRegistry`)

### Use Cases (`internal/usecase/`)
- `executor/` - Task execution use case implementing ReAct pattern

### Adapters (`internal/adapter/`)
- `tools/` - Tool implementations (browser automation and user interaction tools)

### Infrastructure (`internal/infrastructure/`)
- `browser/rod/` - Rod browser implementation
- `llm/openrouter/` - OpenRouter LLM client
- `logger/` - Structured JSON logging
- `env/` - Environment variable service
- `prompts/` - System prompt loader
- `userinteraction/` - Console user interaction

### Dependency Injection (`internal/di/`)
- `container.go` - DI container wiring all components
=======
### Target: Clean Architecture

The project aims for Clean Architecture with clear separation:
- **Domain** - Business entities and interfaces (ports)
- **Use Cases** - Application business rules
- **Infrastructure** - External implementations (browser, LLM clients, logging)
- **Adapters** - Interface adapters between layers

### Current Structure

**Agent Layer** (`internal/agents/`)
- `agents.go` - Main `Agent` interface and factory
- `reactagent.go` - ReAct agent with OpenAI-compatible streaming
- `tools/` - Browser and user interaction tools implementing `Tool` interface

**Domain** (`internal/domain/`)
- `ports/` - Interfaces (`BrowserCore`, `Logger`, `EnvService`)
- `adapter/` - OpenAI tool format adapter

**Infrastructure** (`internal/infrastructure/`)
- `browser/rodwrapper/` - Rod browser wrapper
- `logger/` - JSON structured logging
- `env/` - Environment variable service
>>>>>>> 7e1ff8f (Рефакторинг агента: переход на go-openai SDK)

### Tool Interface

```go
<<<<<<< HEAD
type ToolPort interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, arguments string) (string, error)
}
```

Available tools: `navigate`, `click`, `fill`, `press_enter`, `scroll`, `extract`, `screenshot`, `observe`, `query_elements`, `search`, `ask_question`, `wait_user_action`

## Task Handling Guidelines

- **Clarify before complex tasks**: For complex or ambiguous tasks, always clarify requirements and approach with the user before proceeding:
  - Tasks with multiple valid implementation approaches
  - Tasks where requirements are unclear or incomplete
  - Tasks that affect critical parts of the system
  - Tasks that require architectural decisions
- For straightforward, well-defined tasks, proceed directly without asking unnecessary questions

## Code Style Guidelines

- **No comments**: Do not add comments to code. Code should be self-explanatory through clear naming and structure.
- Write clean, readable code without inline comments, block comments, or documentation comments

## Git Commit Guidelines

- Do not add Claude attribution to commits (no "Generated with Claude Code", no "Co-Authored-By: Claude")
- **IMPORTANT**: Only create git commits when explicitly requested by the user. Do not commit automatically after completing tasks.
- When creating commits, run tests first to ensure code quality
- Use clear, concise commit messages in English or Russian depending on the task context
=======
type Tool interface {
    Name() string
    Type() string
    Description() string
    Parameters() map[string]interface{}
    Call(ctx context.Context, input string) (string, error)
}
```

Available tools: `navigate`, `click`, `fill`, `press_enter`, `scroll`, `extract`, `screenshot`, `ui_summary`, `ask`, `wait_action`

### Refactoring Priorities

1. Decouple agents from concrete infrastructure (rodwrapper, OpenAI client)
2. Add proper interfaces for LLM client
3. Improve test coverage with mocks
4. Proper dependency injection via constructor
>>>>>>> 7e1ff8f (Рефакторинг агента: переход на go-openai SDK)
