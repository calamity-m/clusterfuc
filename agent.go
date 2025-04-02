package clusterfuc

import (
	"context"
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
type Agent[Out any] interface {
	GetPrompt() string
	SetPrompt(prompt string)
	GetName() string
	SetName(name string)
	GetModel() string
	SetModel(model string)
	Call(ctx context.Context, input string) (string, error)
	Register(name string, tool executableTool) error
}

// Shorthand function for opts pattern
type OptsAgent[Out any] func(Agent[Out]) error

// Defaults opts most likely to be used
func DefaultOpts[Out any]() []OptsAgent[Out] {
	opts := make([]OptsAgent[Out], 0)

	return opts
}

// Set prompt
func WithPrompt[Out any](prompt string) OptsAgent[Out] {
	return func(a Agent[Out]) error {
		a.SetPrompt(prompt)
		return nil
	}
}

// Set name
func WithName[Out any](name string) OptsAgent[Out] {
	return func(a Agent[Out]) error {
		a.SetName(name)
		return nil
	}
}

// Set model, tbd how this works in the future
func WithModel[Out any](model string) OptsAgent[Out] {

	return func(a Agent[Out]) error {
		a.SetModel(model)
		return nil
	}
}

// Apply a function as a "tool" to the agent
func WithTool[In any, Out any, AgentOut any](toolName string, tool tool[In, Out]) OptsAgent[AgentOut] {
	return func(a Agent[AgentOut]) error {
		err := a.Register(toolName, Tool(tool))
		if err != nil {
			return err
		}

		return nil
	}
}

// Shorthand option for adding an agent itself as a tool.
func WithAgentAsTool[In any, Out any, AgentOut any](toolName string, agent Agent[AgentOut]) OptsAgent[AgentOut] {
	return func(a Agent[AgentOut]) error {
		err := a.Register(toolName, Tool(agent.Call))
		if err != nil {
			return err
		}

		return nil
	}
}
