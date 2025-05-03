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
	var resp CreateResponse
	if len(history) > 0 {
		err := json.Unmarshal(history, &resp)
		if err != nil {
			return nil, err
		}
	}

	// Set system instructions
	resp.Instructions = prompt

	// Set schema
	if schema != nil {
		resp.Text.Type = "json_schema"
		resp.Text.Strict = true
		resp.Text.Description = "schema for all responses to correspond to"
		resp.Text.Name = "schema"
		resp.Text.Schema = schema
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
	resp.Input = append(resp.Input, i)

	// Set model
	resp.Model = model

	return &resp, nil
}

func (oa *OpenAI) Generate(ctx context.Context, body *CreateResponse, tools []tool.Tool[any, any]) (*CreateResponse, string, error) {
	if body == nil {
		return nil, "", errors.New("nil body")
	}

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
		fmt.Println(body)

		// Send body and get resp
		resp, err := oa.createResponse(ctx, *body)
		if err != nil {
			return nil, "", err
		}

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
					return nil, "", fmt.Errorf("failed to decode function_call - %w", err)
				}

				for _, tool := range tools {
					if tool.Name == call.Name {
						result, err := tool.Executable.Execute(ctx, call.Arguments)
						if err != nil {
							return nil, reply, fmt.Errorf("encountered err while executing tool - %w", err)
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

	r := `{
  "id": "resp_67ccd2bed1ec8190b14f964abc0542670bb6a6b452d3795b",
  "object": "response",
  "created_at": 1741476542,
  "status": "completed",
  "error": null,
  "incomplete_details": null,
  "instructions": null,
  "max_output_tokens": null,
  "model": "gpt-4.1-2025-04-14",
  "output": [
    {
      "type": "message",
      "id": "msg_67ccd2bf17f0819081ff3bb2cf6508e60bb6a6b452d3795b",
      "status": "completed",
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "In a peaceful grove beneath a silver moon, a unicorn named Lumina discovered a hidden pool that reflected the stars. As she dipped her horn into the water, the pool began to shimmer, revealing a pathway to a magical realm of endless night skies. Filled with wonder, Lumina whispered a wish for all who dream to find their own hidden magic, and as she glanced back, her hoofprints sparkled like stardust.",
          "annotations": []
        }
      ]
    }
  ],
  "parallel_tool_calls": true,
  "previous_response_id": null,
  "reasoning": {
    "effort": null,
    "summary": null
  },
  "store": true,
  "temperature": 1.0,
  "text": {
    "format": {
      "type": "text"
    }
  },
  "tool_choice": "auto",
  "tools": [],
  "top_p": 1.0,
  "truncation": "disabled",
  "usage": {
    "input_tokens": 36,
    "input_tokens_details": {
      "cached_tokens": 0
    },
    "output_tokens": 87,
    "output_tokens_details": {
      "reasoning_tokens": 0
    },
    "total_tokens": 123
  },
  "user": null,
  "metadata": {}
}
`
	rb := []byte(r)
	slog.DebugContext(ctx, "ignore me", slog.Any("a", rb))

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
