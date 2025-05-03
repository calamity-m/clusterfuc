package tool

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// An executable is something that can be executed by calling it with
// a context and some input argument T, producing some output S
// and an error.
type executable[T any, S any] interface {
	Execute(context.Context, T) (S, error)
}

// Helper for treating a function as a executable, copying the methodology
// of a http.HandlerFunc
type executableFunc[T any, S any] func(ctx context.Context, in T) (S, error)

// Calls h(r, w) as a Executable
func (h executableFunc[T, S]) Execute(ctx context.Context, in T) (S, error) {
	return h(ctx, in)
}

// Relevant subset of the json schema that providers care about for
// registering function-like tools
type JSONSchemaSubset struct {
	Properties any      `json:"properties,omitempty"`
	Required   []string `json:"required,omitempty"`
}

// A tool is a wrapper around some executable that a function can
// call out to.
type Tool[T any, S any] struct {
	Executable  executable[T, S]
	Definition  JSONSchemaSubset
	Output      JSONSchemaSubset
	Name        string
	Description string
}

// Creates a tool based on some provided function, where it's input/output types are abstracted,
// allowing it to be called by any agent.
//
// The input T and output S must be marshable to/from JSON, as that is how the
// abstraction is implemented.
func CreateTool[T any, S any](name string, fn func(ctx context.Context, in T) (S, error)) Tool[any, any] {
	// Might be worth removing dependency on this,
	// famous last words but inferring a schema
	// should be easy enough as we really just want
	// properties and required for most/all
	// model providers.
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
		ExpandedStruct:            true,
	}

	var val T
	schema := reflector.Reflect(val)

	return Tool[any, any]{
		Name: name,
		Executable: executableFunc[any, any](func(ctx context.Context, in any) (any, error) {
			// If our input is a string encoded json blob, we'll have to handle it
			// slightly differently
			var arg T

			if inStr, ok := in.(string); ok {
				err := json.Unmarshal([]byte(inStr), &arg)
				if err != nil {
					return nil, err
				}
			} else {
				j, err := json.Marshal(in)
				if err != nil {
					return nil, err
				}

				err = json.Unmarshal(j, &arg)
				if err != nil {
					return nil, err
				}
			}

			o, err := fn(ctx, arg)
			if err != nil {
				return nil, err
			}

			return o, nil
		}),
		Definition: JSONSchemaSubset{
			Properties: schema.Properties,
			Required:   schema.Required,
		},
	}
}

func (t *Tool[T, S]) ValidDefinition() bool {
	return false
}
