package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/calamity-m/clusterfuc/pkg/executable"
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
	// TODO - change to tools, not Functions
	Functions []executable.Executable[any, any]
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
	URL     string
}

type AgentInput struct {
	// An agent call should have some ID associated with it.
	// This may be a session ID, a user ID, or some kind of ID
	// related to the input.
	Id string
	// The latest user input. History of input should be maintained
	// via a Memoriser, rather than being passed every input
	// call.
	UserInput string
	Schema    json.RawMessage
}

func (a *Agent[T]) Call(ctx context.Context, input AgentInput) (string, error) {
	if _, ok := a.Model.(model.GeminiAiModel); ok {
		return a.callGemini(ctx, input.Id, input.UserInput, nil)
	}

	if _, ok := a.Model.(model.OpenAiModel); ok {
		return a.callOpenAI(ctx, input.Id, input.UserInput, nil)
	}

	return "", ErrModelUnmatched

}

func (a *Agent[T]) CallV2(ctx context.Context, input AgentInput) (string, error) {
	if a.Memoriser == nil {
		return "", fmt.Errorf("use NoOpMemoriser if no memory is wanted - %w", ErrNilMemoriser)
	}

	if input.Id == "" {
		return "", fmt.Errorf("empty id encountered - %w", ErrInvalidId)
	}

	if input.UserInput == "" {
		return "", fmt.Errorf("empty user input encountered - %w", ErrInvalidUserInput)
	}

	// Fetch our history
	state, err := a.Memoriser.Retrieve(input.Id)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve history - %w", err)
	}

	output := ""

	// body := make([]byte, 0)

	if _, ok := a.Model.(model.GeminiAiModel); ok {
		// body = gemini.Body(prompt, history, Schema)
		// output = gemini.Generate(ctx, body, client)
		// ^- output should be the final string. the
		// Generate method should take care of evertying
		// complex in that package, fuck it off from
		// this place
		return "", errors.ErrUnsupported
	}

	if _, ok := a.Model.(model.OpenAiModel); ok {
		oa, err := openai.NewOpenAIClient(a.Client, a.Auth)
		if err != nil {
			return "", err
		}

		body, err := oa.Body(a.Model.Model(), input.UserInput, a.SystemPrompt, state, input.Schema)
		if err != nil {
			return "", err
		}

		body, output, err = oa.Generate(ctx, body, a.tools)
		if err != nil {
			return output, err
		}

		// Update state
		state, err = json.Marshal(body)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse body into state", slog.Any("error", err), slog.Any("body", body))
		} else {
			if ok := a.Memoriser.Save(input.Id, state); !ok {
				slog.ErrorContext(ctx, "failed to save updated state", slog.Any("error", err))
			}
		}
	}

	return output, nil
}

func (a *Agent[T]) callGemini(ctx context.Context, id string, userInput string, schema *executable.JSONSchemaSubset) (string, error) {

	// Create our base body
	body, err := gemini.CreateRawRequestBody(a.SystemPrompt, schema, a.Functions)
	if err != nil {
		return "", fmt.Errorf("failed to create raw request body - %w", err)
	}

	// Turn history into a content slice and append our user input
	// atm we don't use history :')
	contents, err := gemini.VerifyContents([]any{})
	if err != nil {
		return "", fmt.Errorf("failed turning history into gemini contents - %w", err)
	}
	contents = append(
		contents,
		gemini.Content{
			Role: "user",
			Parts: []gemini.Part{{
				Text: userInput,
			}},
		},
	)
	body.Contents = contents

	// Make the request

	// TODO this url with the auth is based on gemini examples, but should test supplying
	// the token as an authorization bearer not putting it in the fucking url
	// why would google want that anyway?
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", a.URL, a.Model.Model(), a.Auth)
	resp, err := gemini.GenerateContent(ctx, a.Client, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to call gemini api - %w", err)
	}

	// Check if we have any function calls
	for _, candidate := range resp.Candidates {

		for _, part := range candidate.Content.Parts {
			// Add this response to our chain of contents

			fmt.Println(part)
			if part.FunctionCall.Name != "" {
				contents = append(contents, gemini.Content{Role: "model", Parts: []gemini.Part{part}})

				// We have some function call
				name := part.FunctionCall.Name
				for _, exec := range a.Functions {
					if exec.Name == name {
						out, err := exec.Executable.Execute(ctx, part.FunctionCall.Args)
						if err != nil {
							slog.ErrorContext(
								ctx,
								"failed to call function",
								slog.String("function", name),
								slog.Any("input", part.FunctionCall.Args),
							)
							contents = gemini.AppendContentsFailure(contents, name, err.Error())
							continue
						}

						// Append our response
						contents = gemini.AppendFunctionResponse(contents, name, out)
					}
				}
			}
		}
	}

	// Send response back
	body.Contents = contents
	resp, err = gemini.GenerateContent(ctx, a.Client, url, body)
	if err != nil {
		return "", fmt.Errorf("failed second trip request to gemini api - %w", err)
	}
	if resp.Candidates == nil || resp.Candidates[0].Content.Parts == nil {
		return "", fmt.Errorf("did not receive proper response - %#v", resp)
	}

	// Add response to contents and save them
	contents = append(contents, resp.Candidates[0].Content)
	body.Contents = contents

	// We have finished now, so save our contents
	s := make([]any, len(contents))
	for i, v := range contents {
		s[i] = v
	}
	if a.Memoriser != nil {
		// TODO a.Memoriser.Save(id, s)
	}

	// Get final response
	out := resp.Candidates[0].Content.Parts[0].Text

	if a.Verbose {
		if m, err := json.Marshal(body); err == nil {
			fmt.Println(string(m))
		}

		slog.DebugContext(ctx, "output", slog.Any("body", body))
	}

	return out, nil
}

func (a *Agent[T]) callOpenAI(ctx context.Context, id string, userInput string, schema *executable.JSONSchemaSubset) (string, error) {
	return "", nil
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
