package openai

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

type CreateResponse struct {
	// Set of 16 key-value pairs that can be attached to an object.
	// This can be useful for storing additional information about the object in a structured format,
	// and querying for objects via API or the dashboard.
	Metadata map[string]string `json:"metadata,omitzero"`
	// What sampling temperature to use, between 0 and 2. Higher values like 0.8 will make the output more random,
	// while lower values like 0.2 will make it more focused and deterministic.
	// We generally recommend altering this or top_p but not both.
	Temperature float32 `json:"temperature,omitempty"`
	// An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens
	// with top_p probability mass. So 0.1 means only the tokens comprising the top 10% probability mass are considered.
	// We generally recommend altering this or temperature but not both.
	TopP float32 `json:"top_p,omitempty"`
	// A unique identifier representing your end-user, which can help OpenAI to monitor and detect abuse
	User string `json:"user,omitempty"`
	// Specifies the latency tier to use for processing the request. This parameter is relevant for customers subscribed to the scale tier service
	ServiceTier string `json:"service_tier,omitempty"`
	// Model to use, e.g. gpt-4o
	Model string `json:"model"`
	// o-series models only - reasoning configuration
	Reasoning Reasoning `json:"reasoning,omitzero"`
	// An upper bound for the number of tokens that can be generated for a response, including visible output tokens and reasoning tokens
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`
	// Inserts a system (or developer) message as the first item in the model's context
	Instructions string `json:"instructions,omitempty"`
	// Specifies the format that the model must output
	Text TextResponseFormatConfiguration `json:"text,omitzero"`
	// An array of tools the model may call while generating a response
	Tools []FunctionTool `json:"tools,omitzero"`
	// Tbh, idk. But let people do whatever the f they want here
	ToolChoice json.RawMessage `json:"tool_choice,omitzero"`
	// The truncation strategy to use for the model response
	Truncation string `json:"truncation,omitempty"`
	// A list of one or many input items to the model, containing different content types
	Input []json.RawMessage `json:"input"`
	// Specify additional output data to include in the model response
	Include []Includable `json:"include,omitzero"`
	// Whether to store the generated model response for later retrieval via API
	Store bool `json:"store,omitempty"`
	// If set to true, the model response data will be streamed to the client as it is generated using server-sent events
	Stream bool `json:"stream,omitempty"`
}

type Includable string

const (
	// Include the search results of the file search tool call
	IncludableFileSearchCallResults Includable = "file_search_call.results"
	// Include image urls from the input message
	IncludableInputImageImageUrl Includable = "message.input_image.image_url"
	// Include image urls from the computer call output
	IncludableComputerCallOutputImageUrl Includable = "computer_call_output.output.image_url"
)

type Reasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary,omitempty"`
}

type TextResponseFormatConfiguration struct {
	// response format. Used to generate responses. Either `text` or `json_schema`
	Type string `json:"type"`
	// A description of what the response format is for, used by the model to determine how to respond in the format.
	Description string `json:"description,omitempty"`
	// The name of the response format. Must be a-z, A-Z, 0-9, or contain underscores and dashes, with a maximum length of 64.
	Name string `json:"name,omitempty"`
	// The schema for the response format, described as a JSON Schema object. Learn how to build JSON schemas
	Schema json.RawMessage `json:"schema,omitzero"`
	// Whether to enable strict schema adherence when generating the
	Strict bool `json:"strict,omitempty"`
}

// Defines a function in your own code the model can choose to call.
type FunctionTool struct {
	//The type of the function tool. Always `function`.
	Type string `json:"type"`
	// The name of the function to call
	Name string `json:"name,omitempty"`
	// A description of the function. Used by the model to determine whether or not to call the function.
	Description string `json:"description,omitempty"`
	// A JSON schema object describing the parameters of the function
	Parameters FunctionToolParameters `json:"parameters,omitempty"`
	// Whether to enforce strict parameter validation. Default `true`
	Strict bool `json:"strict,omitempty"`
}

type FunctionToolParameters struct {
	Type                 string          `json:"type,omitempty"`
	Properties           json.RawMessage `json:"properties,omitzero"`
	Required             []string        `json:"required,omitempty"`
	AdditionalProperties bool            `json:"additionalProperties,omitempty"`
}

type Message struct {
	BaseItem
	// The unique ID of the output message.
	ID string `json:"id,omitempty"`
	// The role of the message input. One of `user`, `system`, or `developer`.
	Role string `json:"role,omitempty"`
	// The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.
	Status string `json:"status,omitempty"`
	// A list of one or many input items to the model, containing different content types.
	Content []MessageContent `json:"content,omitempty"`
}

// Currently MessageContent only supports text, not file or image
type MessageContent struct {
	// The type of the output text. Always either output_text or input_text.
	Type string `json:"type,omitempty"`
	// The text output from the model
	Text string `json:"text,omitempty"`
	// The annotations of the text output
	Annotations []json.RawMessage `json:"annotations,omitzero"`
	// The refusal explanation from the model.
	Refusal string `json:"refusal,omitempty"`
}

type FunctionToolCall struct {
	BaseItem
	// The unique ID of the function tool call.
	ID string `json:"id,omitempty"`
	// The unique ID of the function tool call generated by the model.
	CallID string `json:"call_id"`
	// The name of the function to run.
	Name string `json:"name"`
	// A JSON string of the arguments to pass to the function.
	Arguments any `json:"arguments,omitzero"`
	//  status of the item. One of in_progress, completed, or incomplete. Populated when items are returned via API.
	Status string `json:"status,omitempty"`
}

type FunctionToolCallOutput struct {
	BaseItem
	// The unique ID of the function tool call output. Populated when this item is returned via API.
	ID string `json:"id,omitempty"`
	// The unique ID of the function tool call generated by the model.
	CallID string `json:"call_id"`
	// A JSON string of the output of the function tool call.
	Output string `json:"output,omitzero"`
	//The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.
	Status string `json:"status,omitempty"`
}

type Response struct {
	// Set of 16 key-value pairs that can be attached to an object.
	// This can be useful for storing additional information about the object in a structured format,
	// and querying for objects via API or the dashboard.
	Metadata map[string]string `json:"metadata,omitempty"`
	// A unique identifier representing your end-user, which can help OpenAI to monitor and detect abuse
	User string `json:"user,omitempty"`
	// Unique identifier for this Response
	ID string `json:"id,omitempty"`
	// The status of the response generation. One of completed, failed, in_progress, or incomplete
	Status string `json:"status,omitempty"`
	// Unix timestamp (in seconds) of when this Response was created
	CreatedAt int `json:"created_at,omitempty"`
	// An error object returned when the model fails to generate a Response
	Error ResponseError `json:"error,omitempty"`
	// Details about why the response is incomplete
	IncompleteDetails IncompleteDetails `json:"incomplete_details,omitempty"`
	// An array of content items generated by the model
	Output []json.RawMessage `json:"output,omitempty"`
	// Represents token usage details including input tokens, output tokens, a breakdown of output tokens, and the total tokens used
	Usage ResponseUsage `json:"usage,omitempty"`
}

type ResponseUsage struct {
	// The number of input tokens
	InputTokens int `json:"input_tokens,omitempty"`
	// A detailed breakdown of the input tokens
	InputTokensDetails InputTokenDetails `json:"input_tokens_details,omitempty"`
	// The number of output tokens
	OutputTokens int `json:"output_tokens,omitempty"`
	// A detailed breakdown of the output tokens
	OutputTokensDetails OutputTokenDetails `json:"output_tokens_details,omitempty"`
	// The total number of tokens used
	TotalTokens int `json:"total_tokens,omitempty"`
}

type InputTokenDetails struct {
	// The number of tokens that were retrieved from the cache
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type OutputTokenDetails struct {
	// The number of reasoning tokens
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type ResponseError struct {
	// The error code for the response
	Code string `json:"code,omitempty"`
	// A human-readable description of the error
	Message string `json:"message,omitempty"`
}

type IncompleteDetails struct {
	Reason string
}

// Potential input items that create and response can trade back and forth with
type BaseItem struct {
	Type string `json:"type"`
}

type OpenAI struct {
	client *http.Client
	auth   string
}

func (oa *OpenAI) Body(model string, userInput string, prompt string, history json.RawMessage, schema json.RawMessage) (*CreateResponse, error) {
	// Validate user input
	if userInput == "" {
		return nil, errors.New("empty user input is weird")
	}

	// Form body from history
	var body CreateResponse
	if len(history) > 0 {
		err := json.Unmarshal(history, &body)
		if err != nil {
			return nil, err
		}
	}

	// Set system instructions
	body.Instructions = prompt

	// Set schema
	if schema != nil {
		body.Text.Type = "json_schema"
		body.Text.Strict = true
		body.Text.Description = "schema for all responses to correspond to"
		body.Text.Name = "schema"
		body.Text.Schema = schema
	}

	// Set user input
	i, err := json.Marshal(Message{
		BaseItem: BaseItem{
			Type: "message",
		},
		Role: "user",
		Content: []MessageContent{
			{
				Type: "input_text",
				Text: userInput,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode user input - %w", err)
	}
	body.Input = append(body.Input, i)

	// Set model
	body.Model = model

	return &body, nil
}

func (oa *OpenAI) Generate(ctx context.Context, body *CreateResponse, tools []tool.Tool[any, any]) (*CreateResponse, string, error) {
	if body == nil {
		return nil, "", errors.New("nil body")
	}

	slog.DebugContext(ctx, "openai agent called", slog.String("model", body.Model))

	// Set our tools on our body
	if len(body.Tools) == 0 {
		for _, tool := range tools {
			params, err := json.Marshal(tool.Definition.Properties)
			if err != nil {
				return nil, "", fmt.Errorf("failed to encode tool for request - %w", err)
			}
			body.Tools = append(body.Tools, FunctionTool{
				Type:        "function",
				Name:        tool.Name,
				Description: tool.Description,
				Strict:      false,
				Parameters: FunctionToolParameters{
					Type:                 "object",
					Properties:           params,
					Required:             tool.Definition.Required,
					AdditionalProperties: false,
				},
			})
		}
	}

	slog.DebugContext(ctx, "openai agent tools registered", slog.Any("tools", body.Tools))

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
		resp, err := oa.createResponse(ctx, *body)
		if err != nil {
			return nil, "", err
		}

		slog.DebugContext(ctx, "received response from openai", slog.Any("resp", resp))

		if resp.Output == nil {
			return nil, "", errors.New("invalid output")
		}

		// loop through response output
		for _, output := range resp.Output {
			var base BaseItem
			err := json.Unmarshal(output, &base)
			if err != nil {
				return nil, "", fmt.Errorf("failed decoding input type - %w", err)
			}

			switch base.Type {
			case "message":
				// Ensure our body retains this for our history
				body.Input = append(body.Input, output)

				var message Message
				err := json.Unmarshal(output, &message)
				if err != nil {
					return nil, "", fmt.Errorf("failed to decode output_text - %w", err)
				}

				for _, content := range message.Content {
					if content.Type != "output_text" {
						slog.ErrorContext(ctx, "received non output_text message from model", slog.Any("type", content.Type))
						return nil, "", fmt.Errorf("received non output_text message from model")
					}

					if content.Refusal != "" {
						slog.ErrorContext(ctx, "encountered refusal", slog.Any("reply", reply), slog.Any("refusal", content.Refusal))
						return nil, "", fmt.Errorf("refusal encountered: %s", content.Refusal)
					} else {
						reply += content.Text
					}
				}

			case "function_call":
				// Ensure our body retains this for our history
				body.Input = append(body.Input, output)

				var call FunctionToolCall
				err := json.Unmarshal(output, &call)
				if err != nil {
					slog.ErrorContext(ctx, "encountered err while parsing tool call", slog.Any("error", err))
					return nil, "", fmt.Errorf("failed to decode function_call - %w", err)
				}

				for _, tool := range tools {
					if tool.Name == call.Name {
						result, err := tool.Executable.Execute(ctx, call.Arguments)
						if err != nil {
							// Tool failures might be expected, so we'll append it to input and move on
							// rather than failing outright
							slog.ErrorContext(ctx, "encountered err while executing tool", slog.Any("error", err))
							output, err := json.Marshal(FunctionToolCallOutput{
								BaseItem: BaseItem{Type: "function_call_output"},
								CallID:   call.CallID,
								Output:   errorResponse(err.Error()),
							})
							if err != nil {
								return nil, reply, fmt.Errorf("failed encoding tool call failure - %w", err)
							}
							body.Input = append(body.Input, output)
							continue
						}

						str, err := json.Marshal(result)
						if err != nil {
							return nil, reply, fmt.Errorf("failed to encode results into json - %w", err)
						}
						output, err := json.Marshal(FunctionToolCallOutput{
							BaseItem: BaseItem{Type: "function_call_output"},
							CallID:   call.CallID,
							Output:   string(str),
						})
						if err != nil {
							return nil, reply, fmt.Errorf("failed encoding tool call result - %w", err)
						}

						body.Input = append(body.Input, output)
					}
				}

				calls = true
			default:
				slog.ErrorContext(ctx, "failed to match output type", slog.Any("type", base.Type), slog.Any("raw", output))
				return nil, "", errors.New("unmatched idk")
			}
		}

		// Send response through again if we are not marked as completed
		if calls || resp.Status != "completed" {
			return oa.Generate(ctx, body, tools)
		}

	}

	// if we've ended up here we have succeded
	return body, reply, nil
}

// createResponse sends a POST request to the OpenAI /v1/responses endpoint and parses the response
func (oa *OpenAI) createResponse(ctx context.Context, body CreateResponse) (*Response, error) {
	// Marshal the request body into JSON
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+oa.auth)

	// Send the HTTP request
	resp, err := oa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal the response body into the Response struct
	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func NewOpenAIClient(client *http.Client, auth string) (*OpenAI, error) {
	return &OpenAI{
		client: client,
		auth:   auth,
	}, nil
}

func errorResponse(message string) string {
	r, err := json.Marshal(struct {
		Success bool   `json:"success"`
		Reason  string `json:"reason"`
	}{
		Success: false,
		Reason:  message,
	})

	if err != nil {
		slog.Error("Encountered improbable json marshling error", slog.String("err", err.Error()))
		return message
	}

	return string(r)
}
