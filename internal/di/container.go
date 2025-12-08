package di

import (
	"context"
	"fmt"

	tool "browser-agent/internal/adapter/tools"
	"browser-agent/internal/application/port/input"
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/application/service"
	"browser-agent/internal/infrastructure/browser/rod"
	"browser-agent/internal/infrastructure/llm/openrouter"
	"browser-agent/internal/infrastructure/logger"
	"browser-agent/internal/infrastructure/prompts"
	"browser-agent/internal/infrastructure/userinteraction"
	"browser-agent/internal/usecase/executor"
)

type Container struct {
	Browser         output.BrowserPort
	LLM             output.LLMPort
	Logger          output.LoggerPort
	UserInteraction output.UserInteractionPort
	Tools           output.ToolRegistry
	TaskExecutor    input.TaskExecutor
}

type Config struct {
	OpenRouterAPIKey   string
	OpenRouterModel    string
	BrowserHeadless    bool
	SystemPrompt       string
	ThinkingMode       bool
	ThinkingBudget     int
}

func NewContainer(ctx context.Context, cfg Config) (*Container, error) {
	log, err := logger.NewLoggerAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	browserCfg := rod.DefaultConfig()
	browserCfg.Headless = cfg.BrowserHeadless
	browser, err := rod.NewBrowserAdapter(ctx, browserCfg)
	if err != nil {
		log.Close()
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}

	llmCfg := openrouter.DefaultConfig(cfg.OpenRouterAPIKey, cfg.OpenRouterModel)
	llmCfg.Logger = log
	llmCfg.ThinkingMode = cfg.ThinkingMode
	if cfg.ThinkingBudget > 0 {
		llmCfg.ThinkingBudget = cfg.ThinkingBudget
	}
	llm := openrouter.NewOpenRouterAdapter(llmCfg)

	userInteraction := userinteraction.NewConsoleUserInteraction()

	tools := service.NewToolRegistry()
	registerBrowserTools(tools, browser, log)
	registerUserInteractionTools(tools, userInteraction, log)

	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = prompts.DefaultSystemPrompt
	}

	uc := executor.New(llm, tools, log, systemPrompt)

	return &Container{
		Browser:         browser,
		LLM:             llm,
		Logger:          log,
		UserInteraction: userInteraction,
		Tools:           tools,
		TaskExecutor:    uc,
	}, nil
}

func (c *Container) Close() {
	if c.Browser != nil {
		c.Browser.Close()
	}
	if c.Logger != nil {
		c.Logger.Close()
	}
}

func registerBrowserTools(registry *service.ToolRegistryImpl, browser output.BrowserPort, log output.LoggerPort) {
	registry.Register(tool.NewNavigateTool(browser, log))
	registry.Register(tool.NewClickTool(browser, log))
	registry.Register(tool.NewFillTool(browser, log))
	registry.Register(tool.NewScrollTool(browser, log))
	registry.Register(tool.NewScreenshotTool(browser, log))
	registry.Register(tool.NewExtractTool(browser, log))
	registry.Register(tool.NewPressEnterTool(browser, log))
	registry.Register(tool.NewObserveTool(browser, log))
}

func registerUserInteractionTools(registry *service.ToolRegistryImpl, userInteraction output.UserInteractionPort, log output.LoggerPort) {
	registry.Register(tool.NewAskQuestionTool(userInteraction, log))
	registry.Register(tool.NewWaitUserActionTool(userInteraction, log))
}
