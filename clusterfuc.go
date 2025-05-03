package clusterfuc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/calamity-m/clusterfuc/pkg/agent"
	"github.com/calamity-m/clusterfuc/pkg/memoriser"
	"github.com/calamity-m/clusterfuc/pkg/model"
	"github.com/calamity-m/clusterfuc/pkg/tool"
)

const (
	OpenAIChatGPT4o     model.OpenAiModel = "gpt-4o"
	OpenAIChatGPT4oMini model.OpenAiModel = "gpt-4o-mini"

	Gemini2Flash     model.GeminiAiModel = "gemini-2.0-flash"
	Gemini2FlashLite model.GeminiAiModel = "gemini-2.0-flash-lite"
)

type AgentConfig struct {
	Client       *http.Client
	Model        model.AIModel
	SystemPrompt string
	Verbose      bool
	Auth         string
	URL          string
}

func NewAgent(cfg *AgentConfig) (*agent.Agent[model.AIModel], error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil agent config not allowed - %w", ErrAgentOptInvalid)
	}

	return &agent.Agent[model.AIModel]{
		Client:       cfg.Client,
		Model:        cfg.Model,
		Memoriser:    &memoriser.NoOpMemoriser{},
		SystemPrompt: cfg.SystemPrompt,
		Verbose:      cfg.Verbose,
		Auth:         cfg.Auth,
	}, nil
}

func RegisterTool[T any, S any](
	a *agent.Agent[model.AIModel],
	name string,
	t func(ctx context.Context, in T) (S, error),
) error {

	a.AddTool(tool.CreateTool(name, t))
	return nil
}
