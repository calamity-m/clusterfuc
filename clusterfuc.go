package clusterfuc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/calamity-m/clusterfuc/internal/agent"
	"github.com/calamity-m/clusterfuc/internal/executable"
	"github.com/calamity-m/clusterfuc/internal/model"
	"github.com/calamity-m/clusterfuc/pkg/memoriser"
)

const (
	OpenAIChatGPT4o     model.OpenAiModel = "gpt-4o"
	OpenAIChatGPT4oMini model.OpenAiModel = "gpt-4o-mini"

	Gemini2Flash     model.GeminiAiModel = "gemini-2.0-flash"
	Gemini2FlashLite model.GeminiAiModel = "gemini-2.0-flash-lite"
)

type GeminiAgentConfig struct {
	Client       *http.Client
	Model        model.GeminiAiModel
	SystemPrompt string
	Verbose      bool
	Auth         string
}

func NewGeminiAgent(cfg *GeminiAgentConfig) (*agent.Agent[model.GeminiAiModel], error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil agent config not allowed - %w", ErrAgentOptInvalid)
	}

	return &agent.Agent[model.GeminiAiModel]{
		Client:       cfg.Client,
		Functions:    []executable.Executable[any, any]{},
		Model:        cfg.Model,
		Memoriser:    &memoriser.NoOpMemoriser{},
		SystemPrompt: cfg.SystemPrompt,
		Verbose:      cfg.Verbose,
		Auth:         cfg.Auth,
	}, nil
}

func ExtendAgent[T any, S any](
	a *agent.Agent[model.OpenAiModel],
	fnName string,
	fn func(ctx context.Context, in T) (S, error),
) (*agent.Agent[model.OpenAiModel], error) {
	if len(a.Functions) >= model.MAX_TOOLS_COUMT {
		return a, ErrExceededMaxToolCount
	}

	a.Functions = append(a.Functions, executable.ExecuteableFunction(fnName, fn))
	return a, nil
}
