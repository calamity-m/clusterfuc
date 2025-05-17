package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/calamity-m/clusterfuc/pkg/tool"
)

var (
	ErrInvalidGeminiContent = errors.New("input contains non gemini content")
)

type FunctionCall struct {
	Name string `json:"name,omitempty"`
	Args any    `json:"args,omitempty"`
}

type FunctionResponse struct {
	Name     string `json:"name,omitempty"`
	Response any    `json:"response,omitempty"`
}

type Part struct {
	Text             string           `json:"text,omitempty"`
	FunctionCall     FunctionCall     `json:"functionCall,omitzero,omitempty"`
	FunctionResponse FunctionResponse `json:"functionResponse,omitzero,omitempty"`
	Thought          bool             `json:"thought,omitzero,omitempty"`
}

// Hacky way to verify union data type
func (p Part) Valid() bool {
	textSet := false

	if p.Text != "" {
		textSet = true
	}

	callSet := false
	if p.FunctionCall.Name != "" || p.FunctionCall.Args != nil {
		if textSet {
			return false
		} else {
			callSet = true
		}
	}

	if p.FunctionResponse.Name != "" || p.FunctionResponse.Response != nil {
		if textSet || callSet {
			return false
		}
	}

	return true
}

type Content struct {
	Role  string `json:"role,omitempty,omitzero"`
	Parts []Part `json:"parts,omitempty,omitzero"`
}

func VerifyContents(in []any) ([]Content, error) {
	contents := make([]Content, len(in))

	for ix, val := range in {
		content, ok := val.(Content)
		if !ok {
			return []Content{}, ErrInvalidGeminiContent
		}

		contents[ix] = content
	}

	return contents, nil
}

type GenerationConfig struct {
	ResponseSchema struct {
		Properties  any      `json:"properties,omitzero,omitempty"`
		Required    []string `json:"required,omitempty"`
		Title       string   `json:"title,omitempty"`
		Description string   `json:"description,omitempty"`
	} `json:"responseSchema,omitzero"`
}

type FunctionDeclaration struct {
	Name        string `json:"name,omitzero,omitempty"`
	Description string `json:"description,omitzero,omitempty"`
	Parameters  any    `json:"parameters,omitzero,omitempty"`
	Response    any    `json:"response,omitzero,omitempty"`
}

type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty,omitzero"`
	GoogleSearch         struct{}              `json:"google_search,omitzero,omitempty"`
}

type RequestBody struct {
	Contents          []Content        `json:"contents,omitempty,omitzero"`
	CachedContent     string           `json:"cachedContent,omitempty,omitzero"`
	Tools             []Tool           `json:"tools,omitempty,omitzero"`
	GenerationConfig  GenerationConfig `json:"generationConfig,omitzero,omitempty"`
	SystemInstruction Part             `json:"system_instruction,omitzero,omitempty"`
}

type Candidate struct {
	Content      Content `json:"content,omitzero,omitempty"`
	FinishReason string  `json:"finish_reason,omitempty,omitzero"`
	SafetyRating []struct {
		Category    string `json:"category,omitempty,omitzero"`
		Probability string `json:"probability,omitempty,omitzero"`
	} `json:"safety_rating,omitzero,omitempty"`
}

// Updated ResponseBody to replace map[string]any with specific fields
type ResponseBody struct {
	Candidates     []Candidate    `json:"candidates,omitzero,omitempty"`
	PromptFeedback string         `json:"promptFeedback,omitzero,omitempty"`
	UsageMetadata  UsageMetadata  `json:"usageMetadata,omitzero,omitempty"`
	SafetyRatings  []SafetyRating `json:"safetyRatings,omitzero,omitempty"`
}

// UsageMetadata represents metadata on token usage
type UsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount,omitzero"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitzero"`
	CandidatesTokenCount    int `json:"candidatesTokenCount,omitzero"`
	ToolUsePromptTokenCount int `json:"toolUsePromptTokenCount,omitzero"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount,omitzero"`
	TotalTokenCount         int `json:"totalTokenCount,omitzero"`
}

// SafetyRating represents safety ratings for the generated content
type SafetyRating struct {
	Category    string  `json:"category,omitzero,omitempty"`
	Probability float64 `json:"probability,omitzero,omitempty"`
	Blocked     bool    `json:"blocked,omitzero,omitempty"`
}

type Gemini struct {
	client *http.Client
	auth   string
	model  string
}

func (oa *Gemini) Body(userInput string, prompt string, history json.RawMessage, schema json.RawMessage) (*RequestBody, error) {
	// Validate user input
	if userInput == "" {
		return nil, errors.New("empty user input is weird")
	}

	// Form body from history
	var body RequestBody
	if len(history) > 0 {
		err := json.Unmarshal(history, &body)
		if err != nil {
			return nil, err
		}
	}

	// System prompt
	body.SystemInstruction.Text = prompt

	// User input
	body.Contents = append(body.Contents, Content{
		Role: "user",
		Parts: []Part{{
			Text: userInput,
		}},
	})

	// Schema
	if len(schema) > 0 {
		var jsonSchema tool.JSONSchemaSubset
		err := json.Unmarshal(schema, &jsonSchema)
		if err != nil {
			return nil, fmt.Errorf("invalid schema supplied, could not decode it - %w", err)
		}

		body.GenerationConfig.ResponseSchema.Properties = jsonSchema.Properties
		body.GenerationConfig.ResponseSchema.Required = jsonSchema.Required
	}

	return &body, nil
}

func (oa *Gemini) Generate(ctx context.Context, body *RequestBody, tools []tool.Tool[any, any]) (*RequestBody, string, error) {
	slog.DebugContext(ctx, "gemini agent called", slog.String("model", oa.model))

	if body == nil {
		return nil, "", errors.New("nil body")
	}

	// Set our tools on our body
	if len(body.Tools) == 0 {
		functionDecs := make([]FunctionDeclaration, len(tools))
		for i, tool := range tools {
			functionDecs[i] = FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Name,
				Parameters: map[string]any{
					"type":       "object",
					"properties": tool.Definition.Properties,
					"required":   tool.Definition.Required,
				},
			}
		}
		body.Tools = []Tool{{FunctionDeclarations: functionDecs}}
	}

	// In case we are returning, we need to record
	// our potential replies
	reply := ""

	// We might have function calls that require a resend
	calls := false

	// We might be calling a few times depending on the model, so
	// if we have a ctx done before we send a response we should
	// exit
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:

		// Send body and get resp
		resp, err := oa.generateContent(ctx, *body)
		if err != nil {
			return nil, "", err
		}

		if resp.Candidates == nil {
			return nil, "", errors.New("invalid output")
		}

		for _, candidate := range resp.Candidates {
			// Ensure our body retains this candidate for our history
			body.Contents = append(body.Contents, candidate.Content)

			for _, part := range candidate.Content.Parts {
				if part.FunctionCall.Name == "" {
					// We are on a message, rather than a function
					// call
					reply += part.Text
				} else {
					// Flip our tool call switch
					calls = true

					for _, tool := range tools {
						if tool.Name == part.FunctionCall.Name {
							out, err := tool.Executable.Execute(ctx, part.FunctionCall.Args)
							if err != nil {
								slog.ErrorContext(ctx, "failed to execute tool", slog.Any("tool", part.FunctionCall))

								// Add failed execution to the history
								body.Contents = append(body.Contents, Content{
									Role: "user",
									Parts: []Part{{
										FunctionResponse: FunctionResponse{
											Name: part.FunctionCall.Name,
											Response: map[string]any{
												"success":       false,
												"failureReason": err.Error(),
											},
										},
									}},
								})
							}

							// Add execution to the history
							body.Contents = append(body.Contents, Content{
								Role: "user",
								Parts: []Part{{
									FunctionResponse: FunctionResponse{
										Name:     part.FunctionCall.Name,
										Response: out,
									},
								}},
							})

						}
					}
				}

			}
		}

		if calls {
			return oa.Generate(ctx, body, tools)
		}

	}

	// if we've ended up here we have succeded
	return body, reply, nil
}

// createResponse sends a POST request to the OpenAI /v1/responses endpoint and parses the response
func (oa *Gemini) generateContent(ctx context.Context, body RequestBody) (*ResponseBody, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return &ResponseBody{}, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", "https://generativelanguage.googleapis.com/v1beta/models", oa.model, oa.auth)
	r, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return &ResponseBody{}, err
	}
	r.Header.Set("Content-Type", "application/json")

	resp, err := oa.client.Do(r)
	if err != nil {
		return &ResponseBody{}, err
	}

	if resp.StatusCode != 200 {
		failed, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.ErrorContext(ctx, "non 200 response parsing failed", slog.Any("error", err))
		}
		slog.ErrorContext(ctx, "non 200 response from gemini", slog.Any("body", failed))
		return &ResponseBody{}, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ResponseBody{}, err
	}

	var generated ResponseBody
	err = json.Unmarshal(respData, &generated)
	if err != nil {
		return &ResponseBody{}, err
	}

	return &generated, nil
}

func NewGeminiClient(client *http.Client, auth string, model string) (*Gemini, error) {
	return &Gemini{
		client: client,
		auth:   auth,
		model:  model,
	}, nil
}
