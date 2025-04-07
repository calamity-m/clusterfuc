package clusterfuc

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/invopop/jsonschema"
)

// executable defines some executable code that can be called
// agnostically of the underlying implementation
type executable interface {
	Execute(r io.Reader, w io.Writer) error
}

// Wrapper for a function that can be used as an executable
type executableFunc func(r io.Reader, w io.Writer) error

// Calls h(r, w) as a Executable
func (h executableFunc) Execute(r io.Reader, w io.Writer) error {
	return h(r, w)
}

// tool represets a executable function with input and output formats
// that an agent can utilise in a function/tool calling methodology
type tool[In any, Out any] func(context.Context, In) (Out, error)

// Definition determined to be associated with some Tool. It is intended that
// a definition will be generated at runtime rather than compile time.
type toolDef struct {
	Properties interface{}
	Required   []string
}

// Represents a fully formed "tool" that can be registered with an agent
// and called as part of the agent's functionality. An executableTool aims to
// abstract the management of parsing into and out of the tool's required
// formats and generally should not be directly instantiated by the user.
//
// Instead, it is recommended that either the AgentToolFunc wrapper or WithTool option
// are utilised to register tools with the agent.
type executableTool struct {
	Executable  executable
	Definition  toolDef
	Description string
}

// A tool itself is some function with a predetermined input and output that matches roughly,
// the following signature:
// func(context.Context, In) (Out, error).
//
// The function will be called with the context and input when the tool is executed.
// Output will be marshalled to JSON and provided back to the calling agent. It is assumed
// that the input type is considered to represent the tool's "schema" and will be geneerated
// at runtime.
func Tool[In any, Out any](f tool[In, Out], description string) executableTool {

	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
		ExpandedStruct:            true,
	}

	var val In
	schema := reflector.Reflect(val)

	return executableTool{
		Executable: executableFunc(func(r io.Reader, w io.Writer) error {
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
		Definition: toolDef{
			Properties: schema.Properties,
			Required:   schema.Required,
		}}
}

// Follows the same pattern as Tool but allows the user to specify a definition as a json
// encoded string that matches the internal toolDef struct.
func ToolDef[In any, Out any](f tool[In, Out], definition string, description string) (executableTool, error) {

	var def toolDef

	// Retrieve data from request
	err := json.NewDecoder(strings.NewReader(definition)).Decode(&def)
	if err != nil {
		return executableTool{}, err
	}

	return executableTool{
		Executable: executableFunc(func(r io.Reader, w io.Writer) error {
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
		Definition: def,
	}, nil
}
