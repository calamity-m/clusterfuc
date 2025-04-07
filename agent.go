package clusterfuc

import (
	"context"
	"errors"
	"net/http"
)

// Basic binding for agent output
type SingleAnswer struct {
	Answer string `json:"answer"`
}

// Basic input and output struct for
// agents that append their messages to a chain
type Flow struct {
	Messages []string `json:"messages"`
}

// Agent interface providing the same functionality as the original struct
type Agent[Out any] struct {
	tools      map[string]executableTool
	httpClient *http.Client
	prompt     string
	name       string
	model      string
	verbose    bool
}

func (a *Agent[Out]) ListTools() map[string]executableTool {
	return a.tools
}

func (a *Agent[Out]) RegisterTool(name string, tool executableTool) error {
	if len(a.tools) > MAX_TOOLS_COUMT {
		return ErrExceededMaxToolCount
	}

	if _, ok := a.tools[name]; ok {
		return ErrToolAlreadyExists
	}

	a.tools[name] = tool
	return nil
}

func (a *Agent[Out]) Call(ctx context.Context, input string) (string, error) {

	// Haha switch statement go brrrr
	// there is definitely a 100% better way to do this
	// but i am lazy and this is easy to read
	switch a.model {
	case OpenAIChatGPT4o.Model(), OpenAIChatGPT4oMini.Model():
		return a.callOpenAIModel(ctx, input)
	case Gemini2Flash.Model(), Gemini2FlashLite.Model():
		return a.callGeminiModel(ctx, input)
	}

	// Unmatched model so let's just go away and cry
	return "", errors.ErrUnsupported
}

// Shorthand function for opts pattern
type OptsAgent[Out any] func(*Agent[Out]) error

// Defaults opts most likely to be used
func DefaultOpts[Out any]() []OptsAgent[Out] {
	opts := make([]OptsAgent[Out], 0)

	return opts
}

// Set prompt
func WithPrompt[Out any](prompt string) OptsAgent[Out] {
	return func(a *Agent[Out]) error {
		a.prompt = prompt
		return nil
	}
}

// Set name
func WithName[Out any](name string) OptsAgent[Out] {
	return func(a *Agent[Out]) error {
		a.name = name
		return nil
	}
}

// Set model, tbd how this works in the future
func WithModel[Out any, Model OpenAIModel | GeminiModel](model AIModel[Model]) OptsAgent[Out] {
	return func(a *Agent[Out]) error {
		a.model = model.Model()
		return nil
	}
}

// Set HTTP client
func WithHTTPClient[Out any](client *http.Client) OptsAgent[Out] {
	return func(a *Agent[Out]) error {
		a.httpClient = client
		return nil
	}
}

// Apply a function as a "tool" to the agent
func WithTool[In any, Out any, AgentOut any](toolName string, tool tool[In, Out]) OptsAgent[AgentOut] {
	return func(a *Agent[AgentOut]) error {
		err := a.RegisterTool(toolName, Tool(tool))
		if err != nil {
			return err
		}

		return nil
	}
}

// Shorthand option for adding an agent itself as a tool.
func WithAgentAsTool[In any, Out any, AgentOut any](toolName string, agent Agent[AgentOut]) OptsAgent[AgentOut] {
	return func(a *Agent[AgentOut]) error {
		err := a.RegisterTool(toolName, Tool(agent.Call))
		if err != nil {
			return err
		}

		return nil
	}
}

// Create a new agent with the given options
func NewAgent[Out any](opts ...OptsAgent[Out]) (*Agent[Out], error) {
	opts = append(DefaultOpts[Out](), opts...)

	agent := &Agent[Out]{tools: make(map[string]executableTool)}
	for _, opt := range opts {
		err := opt(agent)
		if err != nil {
			return nil, err
		}
	}

	return agent, nil
}
