package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/calamity-m/clusterfuc/pkg/gemini"
	"github.com/calamity-m/clusterfuc/pkg/memoriser"
	"github.com/calamity-m/clusterfuc/pkg/model"
	"github.com/calamity-m/clusterfuc/pkg/openai"
	"github.com/calamity-m/clusterfuc/pkg/tool"
)

var (
	ErrModelUnmatched   = errors.New("model could not be matched")
	ErrInvalidId        = errors.New("invalid id")
	ErrInvalidUserInput = errors.New("invalid user input")
	ErrNilMemoriser     = errors.New("nil memoriser")
)

// T model type, drives what agent this will be
type Agent[T model.AIModel] struct {
	// An internal list of tools. These tools must be abstracted
	// in terms of input/output. It is assumed that an agent
	// will serialize to and from json for tool call operations.
	//
	// The tool package provides a helper wrapper for turning any
	// function into this style
	tools        []tool.Tool[any, any]
	Memoriser    memoriser.Memoriser
	Client       *http.Client
	SystemPrompt string
	Model        model.AIModel
	Auth         string
	// Verbose will print user input, which may
	// be a cause for concern
	Verbose bool
}

type AgentInput struct {
	// An agent call should have some ID associated with it.
	// This may be a session ID, a user ID, or some kind of ID
	// related to the input.
	Id string `json:"id,omitempty" jsonschema:"description=ID of the user or conversation,required"`
	// The latest user input. History of input should be maintained
	// via a Memoriser, rather than being passed every input
	// call.
	UserInput string `json:"user_input,omitempty" jsonschema:"description=Input of the user for agent to use,required"`
	// Optional schema for agent to follow. The schema should follow some json encoded message, which is dependent on
	// the model provider being used. For example, the schema accepted by gemini may be different that the one
	// accepted by openai.
	Schema json.RawMessage `json:"-"`
}

type AgentOutput struct {
	Output string `json:"output,omitempty"`
}

func (a *Agent[T]) Call(ctx context.Context, input AgentInput) (AgentOutput, error) {
	if a.Memoriser == nil {
		return AgentOutput{}, fmt.Errorf("use NoOpMemoriser if no memory is wanted - %w", ErrNilMemoriser)
	}

	if input.Id == "" {
		return AgentOutput{}, fmt.Errorf("empty id encountered - %w", ErrInvalidId)
	}

	if input.UserInput == "" {
		return AgentOutput{}, fmt.Errorf("empty user input encountered - %w", ErrInvalidUserInput)
	}

	// Fetch our history
	state, err := a.Memoriser.Retrieve(input.Id)
	if err != nil {
		return AgentOutput{}, fmt.Errorf("failed to retrieve history - %w", err)
	}

	output := AgentOutput{}

	if _, ok := a.Model.(model.GeminiAiModel); ok {
		g, err := gemini.NewGeminiClient(a.Client, a.Auth, a.Model.Model())
		if err != nil {
			return AgentOutput{}, err
		}
		body, err := g.Body(input.UserInput, a.SystemPrompt, state, input.Schema)
		if err != nil {
			return AgentOutput{}, err
		}

		body, res, err := g.Generate(ctx, body, a.tools)
		if err != nil {
			return AgentOutput{}, err
		}
		output.Output = res

		// Update state
		state, err = json.Marshal(body)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse gemini body into state", slog.Any("error", err), slog.Any("body", body))
		} else {
			if ok := a.Memoriser.Save(input.Id, state); !ok {
				slog.ErrorContext(ctx, "failed to save updated gemini state", slog.Any("error", err))
			}
		}
	}

	if _, ok := a.Model.(model.OpenAiModel); ok {
		oa, err := openai.NewOpenAIClient(a.Client, a.Auth)
		if err != nil {
			return AgentOutput{}, err
		}

		body, err := oa.Body(a.Model.Model(), input.UserInput, a.SystemPrompt, state, input.Schema)
		if err != nil {
			return AgentOutput{}, err
		}

		body, res, err := oa.Generate(ctx, body, a.tools)
		if err != nil {
			return output, err
		}
		output.Output = res

		// Update state
		state, err = json.Marshal(body)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse openai body into state", slog.Any("error", err), slog.Any("body", body))
		} else {
			if ok := a.Memoriser.Save(input.Id, state); !ok {
				slog.ErrorContext(ctx, "failed to save updated openai state", slog.Any("error", err))
			}
		}
	}

	return output, nil
}

func (a *Agent[T]) AddTool(tool tool.Tool[any, any]) {
	a.tools = append(a.tools, tool)
}

func NewAgent(m model.AIModel) (*Agent[model.AIModel], error) {
	agent := &Agent[model.AIModel]{
		Model: m,
		tools: make([]tool.Tool[any, any], 0),
	}
	return agent, nil
}
