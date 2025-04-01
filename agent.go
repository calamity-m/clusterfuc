package clusterfuc

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/invopop/jsonschema"
)

const MAX_TOOLS_COUMT = 10

// Definition of a tool, used for the agent
// provider llm to actually do stuff
type AgentToolDefinition struct {
	Properties interface{}
	Required   []string
}

// Encapsulates a tool the agent has at it's disposal
type AgentTool struct {
	Executable Executable
	Definition AgentToolDefinition
}

// The agent itself, agnostic of model or provider or llm.
// Model value should drive this.
type Agent struct {
	Prompt      string
	Name        string
	Model       string
	Executables map[string]AgentTool
}

// Building ontop of an agent, this provides structured output
// and input
type StructuredAgent[In any, Out any] struct {
	Agent
}

// Shorthand function for opts pattern
type OptsAgent func(*Agent) error

// Defaults opts most likely to be used
func defaultOpts() []OptsAgent {
	opts := make([]OptsAgent, 0)

	return opts
}

// Create new agent using opts pattern
func NewAgent(opts ...OptsAgent) (*Agent, error) {
	opts = append(defaultOpts(), opts...)

	agent := &Agent{}
	for _, opt := range opts {
		err := opt(agent)
		if err != nil {
			return nil, err
		}
	}

	return agent, nil
}

// Set prompt
func WithPrompt(prompt string) OptsAgent {
	return func(a *Agent) error {
		a.Prompt = prompt
		return nil
	}
}

// Set name
func WithName(name string) OptsAgent {
	return func(a *Agent) error {
		a.Name = name
		return nil
	}
}

// Set model, tbd how this works in the future
func WithModel(model string) OptsAgent {

	return func(a *Agent) error {
		a.Model = model
		return nil
	}
}

// Apply a function as a "tool" to the agent
func WithTool[In any, Out any](toolName string, tool Tool[In, Out]) OptsAgent {
	return func(a *Agent) error {
		err := a.Register(toolName, ExecutableTool(tool))
		if err != nil {
			return err
		}

		return nil
	}
}

// Shorthand option for adding an agent itself as a tool.
func WithAgentAsTool[In any, Out any](toolName string, agent Agent) OptsAgent {
	return func(a *Agent) error {
		err := a.Register(toolName, ExecutableTool(agent.Call))
		if err != nil {
			return err
		}

		return nil
	}
}

// Shorthand option for adding a structured output agent as a tool.
func WithStructuredAgentAsTool[In any, Out any](toolName string, agent StructuredAgent[In, Out]) OptsAgent {
	return func(a *Agent) error {
		err := a.Register(toolName, ExecutableTool(agent.StructuredCall))
		if err != nil {
			return err
		}

		return nil
	}
}

// Call the agent
func (a *Agent) Call(ctx context.Context, input string) (string, error) {
	return "", errors.ErrUnsupported
}

// Call the agent using structured output
func (a *StructuredAgent[In, Out]) StructuredCall(ctx context.Context, input In) (Out, error) {
	var out Out
	return out, errors.ErrUnsupported
}

// Registers a tool with the agent
func (a *Agent) Register(name string, tool AgentTool) error {
	if len(a.Executables) > MAX_TOOLS_COUMT {
		return ErrExceededMaxToolCount
	}

	if _, ok := a.Executables[name]; ok {
		return ErrToolAlreadyExists
	}

	a.Executables[name] = tool
	return nil
}

// Base executable
type Executable interface {
	Execute(r io.Reader, w io.Writer) error
}

// Handler
type ExecutableHandler func(r io.Reader, w io.Writer) error

func (h ExecutableHandler) Execute(r io.Reader, w io.Writer) error {
	return h(r, w)
}

// Function type that can be used to wrap your functions as tools
// through ExecutableTool func
type Tool[In any, Out any] func(context.Context, In) (Out, error)

// Wraps a function as a tool
func ExecutableTool[In any, Out any](f Tool[In, Out]) AgentTool {

	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
		ExpandedStruct:            true,
	}

	var val In
	schema := reflector.Reflect(val)

	return AgentTool{
		Executable: ExecutableHandler(func(r io.Reader, w io.Writer) error {
			var in In

			// Retrieve data from request
			err := json.NewDecoder(r).Decode(&in)
			if err != nil {
				return err
			}

			out, err := f(context.TODO(), in)
			if err != nil {
				return err
			}

			err = json.NewEncoder(w).Encode(out)
			if err != nil {
				return err
			}

			return nil
		}),
		Definition: AgentToolDefinition{
			Properties: schema.Properties,
			Required:   schema.Required,
		}}
}
