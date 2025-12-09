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
	"browser-agent/internal/usecase/agents/analysis"
	"browser-agent/internal/usecase/agents/extraction"
	"browser-agent/internal/usecase/agents/form"
	"browser-agent/internal/usecase/agents/navigation"
	"browser-agent/internal/usecase/orchestrator"
)

type Container struct {
	Browser         output.BrowserPort
	LLM             output.LLMPort
	Logger          output.LoggerPort
	UserInteraction output.UserInteractionPort
	Tools           output.ToolRegistry
	SimpleAgents    output.SimpleAgentRegistry
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

	simpleAgents := service.NewSimpleAgentRegistry()
	registerSimpleAgents(simpleAgents, llm, tools, log)

	agentTools := service.NewToolRegistry()
	registerAgentTools(agentTools, simpleAgents, log)

	orchestratorUC := orchestrator.New(llm, agentTools, log, prompts.OrchestratorPrompt)

	return &Container{
		Browser:         browser,
		LLM:             llm,
		Logger:          log,
		UserInteraction: userInteraction,
		Tools:           tools,
		SimpleAgents:    simpleAgents,
		TaskExecutor:    orchestratorUC,
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
	registry.Register(tool.NewPressEnterTool(browser, log))
	registry.Register(tool.NewObserveTool(browser, log))
	registry.Register(tool.NewQueryElementsTool(browser, log))
	registry.Register(tool.NewSearchTool(browser, log))
}

func registerUserInteractionTools(registry *service.ToolRegistryImpl, userInteraction output.UserInteractionPort, log output.LoggerPort) {
	registry.Register(tool.NewAskQuestionTool(userInteraction, log))
	registry.Register(tool.NewWaitUserActionTool(userInteraction, log))
}

func registerSimpleAgents(registry *service.SimpleAgentRegistryImpl, llm output.LLMPort, tools output.ToolRegistry, log output.LoggerPort) {
	registry.Register(navigation.New(llm, tools, log, prompts.NavigationPrompt))
	registry.Register(extraction.New(llm, tools, log, prompts.ExtractionPrompt))
	registry.Register(form.New(llm, tools, log, prompts.FormPrompt))
	registry.Register(analysis.New(llm, tools, log, prompts.AnalysisPrompt))
}

func registerAgentTools(registry *service.ToolRegistryImpl, agents output.SimpleAgentRegistry, log output.LoggerPort) {
	for _, agent := range agents.List() {
		registry.Register(tool.NewAgentTool(agent, log))
	}
}
