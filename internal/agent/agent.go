package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/calamity-m/clusterfuc/internal/executable"
	"github.com/calamity-m/clusterfuc/internal/gemini"
	"github.com/calamity-m/clusterfuc/internal/model"
	"github.com/calamity-m/clusterfuc/pkg/memoriser"
)

var (
	ErrModelUnmatched = errors.New("model could not be matched")
)

// T model type, drives what agent this will be
type Agent[T model.AIModel] struct {
	Functions    []executable.Executable[any, any]
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
	Id        string
	UserInput string
	Schema    *executable.JSONSchemaSubset
}

func (a *Agent[T]) Call(ctx context.Context, input AgentInput) (string, error) {
	if _, ok := a.Model.(model.GeminiAiModel); ok {
		return a.callGeminiSchema(ctx, input.Id, input.UserInput, input.Schema)
	}

	if _, ok := a.Model.(model.OpenAiModel); ok {
		return a.callOpenAISchema(ctx, input.Id, input.UserInput, input.Schema)
	}

	return "", ErrModelUnmatched

}

func (a *Agent[T]) callGeminiSchema(ctx context.Context, id string, userInput string, schema *executable.JSONSchemaSubset) (string, error) {

	// Create our base body
	body, err := gemini.CreateRawRequestBody(a.SystemPrompt, schema, a.Functions)
	if err != nil {
		return "", fmt.Errorf("failed to create raw request body - %w", err)
	}

	// Fetch our history
	var history []any
	if a.Memoriser != nil {
		history, err = a.Memoriser.Retrieve(id)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve history - %w", err)
		}
	} else {
		history = []any{}
	}

	// Turn history into a content slice and append our user input
	contents, err := gemini.VerifyContents(history)
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
		a.Memoriser.Save(id, s)
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

func (a *Agent[T]) callOpenAISchema(ctx context.Context, id string, userInput string, schema *executable.JSONSchemaSubset) (string, error) {
	return "", nil
}
