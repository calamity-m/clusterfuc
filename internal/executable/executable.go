package executable

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type Execute[T any, S any] interface {
	Execute(context.Context, T) (S, error)
}

// Wrapper for a function that can be used as an executable
//
// T -> Input for function.
//
// S -> Output for function.
//
//	^-> it is important to note that S is not the "structured output"
//		an agent is confined by. S is purely some output from the function
//		that should be provided back to the agent as an answer to their
//		call.
type ExecuteFunc[T any, S any] func(ctx context.Context, in T) (S, error)

// Calls h(r, w) as a Executable
func (h ExecuteFunc[T, S]) Execute(ctx context.Context, in T) (S, error) {
	return h(ctx, in)
}

func ExecuteableFunction[T any, S any](name string, fn func(ctx context.Context, in T) (S, error)) Executable[any, any] {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
		ExpandedStruct:            true,
	}

	var val T
	schema := reflector.Reflect(val)

	return Executable[any, any]{
		Name: name,
		Executable: ExecuteFunc[any, any](func(ctx context.Context, in any) (any, error) {

			j, err := json.Marshal(in)
			if err != nil {
				return nil, err
			}

			var arg T
			err = json.Unmarshal(j, &arg)
			if err != nil {
				return nil, err
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

type JSONSchemaSubset struct {
	Properties any
	Required   []string
}

type Executable[T any, S any] struct {
	Executable  Execute[T, S]
	Definition  JSONSchemaSubset
	Output      JSONSchemaSubset
	Name        string
	Description string
}

func (t *Executable[T, S]) ValidDefinition() bool {
	return false
}
