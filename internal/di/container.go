package di

import (
	"context"
	"fmt"

	"browser-agent/internal/adapter/tool"
	"browser-agent/internal/application/port/input"
	"browser-agent/internal/application/port/output"
	"browser-agent/internal/application/service"
	"browser-agent/internal/infrastructure/browser/rod"
	"browser-agent/internal/infrastructure/llm/openrouter"
	"browser-agent/internal/infrastructure/logger"
	"browser-agent/internal/infrastructure/prompts"
	"browser-agent/internal/usecase/executor"
)

type Container struct {
	Browser      output.BrowserPort
	LLM          output.LLMPort
	Logger       output.LoggerPort
	Tools        output.ToolRegistry
	TaskExecutor input.TaskExecutor
}

type Config struct {
	OpenRouterAPIKey string
	OpenRouterModel  string
	BrowserHeadless  bool
	SystemPrompt     string
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
	llm := openrouter.NewOpenRouterAdapter(llmCfg)

	tools := service.NewToolRegistry()
	registerBrowserTools(tools, browser, log)

	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = prompts.DefaultSystemPrompt
	}

	uc := executor.New(llm, tools, log, systemPrompt)

	return &Container{
		Browser:      browser,
		LLM:          llm,
		Logger:       log,
		Tools:        tools,
		TaskExecutor: uc,
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
	registry.Register(tool.NewUISummaryTool(browser, log))
	registry.Register(tool.NewPressEnterTool(browser, log))
}
