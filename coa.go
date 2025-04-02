package clusterfuc

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
)

type OpenAIAgent[Out any] struct {
	Agent[Out]
	Client      *openai.Client
	prompt      string
	name        string
	model       openai.ChatModel
	executables map[string]executableTool
}

func (a *OpenAIAgent[Out]) GetPrompt() string {
	return a.prompt
}

func (a *OpenAIAgent[Out]) SetPrompt(prompt string) {
	a.prompt = prompt
}

func (a *OpenAIAgent[Out]) GetName() string {
	return a.name
}

func (a *OpenAIAgent[Out]) SetName(name string) {
	a.name = name
}

func (a *OpenAIAgent[Out]) GetModel() string {
	return a.model
}

func (a *OpenAIAgent[Out]) SetModel(model string) {
	a.model = model
}

func (a *OpenAIAgent[Out]) GetExecutables() map[string]executableTool {
	return a.executables
}

func (a *OpenAIAgent[Out]) Call(ctx context.Context, input string) (string, error) {
	// ...existing logic for Call...
	return "", errors.ErrUnsupported
}

func (a *OpenAIAgent[Out]) Register(name string, tool executableTool) error {
	if len(a.executables) > MAX_TOOLS_COUMT {
		return ErrExceededMaxToolCount
	}

	if _, ok := a.executables[name]; ok {
		return ErrToolAlreadyExists
	}

	a.executables[name] = tool
	return nil
}

func WithOpenAIClient[Out any](client *openai.Client) OptsAgent[Out] {
	return func(a Agent[Out]) error {
		if client == nil {
			return fmt.Errorf("client cannot be nil - %w", ErrAgentClientInvalid)
		}

		agent, ok := a.(*OpenAIAgent[Out])
		if !ok {
			return fmt.Errorf("expected OpenAIAgent, got %T, - %w", a, ErrAgentClientInvalid)
		}
		agent.Client = client
		return nil
	}
}

func NewOpenAIAgent[Out any](opts ...OptsAgent[Out]) (*OpenAIAgent[Out], error) {
	opts = append(DefaultOpts[Out](), opts...)

	agent := &OpenAIAgent[Out]{executables: make(map[string]executableTool)}
	for _, opt := range opts {
		err := opt(agent)
		if err != nil {
			return nil, err
		}
	}

	return agent, nil
}
