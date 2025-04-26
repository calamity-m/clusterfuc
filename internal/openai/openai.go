package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ──────────────────────────────────────────────────────────────────────────────
// Step 1: Struct definitions for request & response bodies
// ──────────────────────────────────────────────────────────────────────────────

// ResponseRequest models the POST /v1/responses payload.
type ResponseRequest struct {
	Model  string       `json:"model"`           // e.g. "gpt-4o-mini" :contentReference[oaicite:3]{index=3}
	Input  []InputItem  `json:"input"`           // text-only user/admin messages :contentReference[oaicite:4]{index=4}
	Format FormatConfig `json:"format"`          // choose text vs JSONSchema :contentReference[oaicite:5]{index=5}
	Tools  []ToolUnion  `json:"tools,omitempty"` // built-in or custom functions :contentReference[oaicite:6]{index=6}
}

// InputItem represents a single text item in the input array.
type InputItem struct {
	Type   string `json:"type"`             // must be "text" :contentReference[oaicite:7]{index=7}
	Role   string `json:"role"`             // e.g. "user", "assistant" :contentReference[oaicite:8]{index=8}
	Text   string `json:"text,omitempty"`   // the actual message content :contentReference[oaicite:9]{index=9}
	Output string `json:"output,omitempty"` // tool calll result
}

// FormatConfig lets you request text-only or JSON‑schema structured output.
type FormatConfig struct {
	OfTextConfig *TextConfig       `json:"of_text_config,omitempty"`
	OfJSONSchema *JSONSchemaConfig `json:"of_jsonschema,omitempty"`
}

// TextConfig for text‑only output.
type TextConfig struct {
	Type string `json:"type"` // must be "text" :contentReference[oaicite:10]{index=10}
}

// JSONSchemaConfig for structured JSON output via a provided schema.
type JSONSchemaConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"` // a JSON‑schema object :contentReference[oaicite:11]{index=11}
	Strict      bool            `json:"strict,omitempty"`
}

// ToolUnion allows either a built‑in tool or a custom function definition.
type ToolUnion struct {
	// For a built‑in tool:
	Name        string          `json:"name,omitempty"`        // e.g. "web_search" :contentReference[oaicite:12]{index=12}
	Description string          `json:"description,omitempty"` // human‑readable
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // tool-specific JSON schema

	// For a function call, you can supply "name", "description", and a full JSON schema
}

// ResponseResponse models the API’s response.
type ResponseResponse struct {
	ID           string         `json:"id"`                   // unique response ID :contentReference[oaicite:13]{index=13}
	CreatedAt    int64          `json:"created_at,omitempty"` // epoch seconds
	Output       ResponseOutput `json:"output"`
	FunctionCall *FunctionCall  `json:"function_call,omitempty"`
}

// ResponseOutput holds the text result.
type ResponseOutput struct {
	Text string `json:"text"` // generated text :contentReference[oaicite:14]{index=14}
}

// FunctionCall captures a model-initiated function invocation.
type FunctionCall struct {
	Name      string          `json:"name"`      // function name :contentReference[oaicite:15]{index=15}
	Arguments json.RawMessage `json:"arguments"` // JSON‑object of arguments :contentReference[oaicite:16]{index=16}
}

// ──────────────────────────────────────────────────────────────────────────────
// Step 2: HTTP helper functions using net/http + encoding/json
// ──────────────────────────────────────────────────────────────────────────────

const (
	baseURL = "https://api.openai.com/v1/responses"
)

// CreateResponse sends a POST to /v1/responses and returns the parsed response.
func CreateResponse(ctx context.Context, client *http.Client, apiKey string, reqBody ResponseRequest) (*ResponseResponse, error) {
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status %d: %s", resp.StatusCode, data)
	}

	var out ResponseResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Step 3: Example usage
// ──────────────────────────────────────────────────────────────────────────────

func ex() {
	ctx := context.Background()
	apiKey := "YOUR_OPENAI_KEY"

	// Build a text‑only request
	req := ResponseRequest{
		Model: "gpt-4o-mini",
		Input: []InputItem{
			{Type: "text", Role: "user", Text: "Hello, how are you?"},
		},
		Format: FormatConfig{
			OfTextConfig: &TextConfig{Type: "text"},
		},
		Tools: []ToolUnion{
			// Example: built-in web search
			{
				Name:        "web_search",
				Description: "Real‑time search over the web",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}

	// Create
	res, err := CreateResponse(ctx, http.DefaultClient, apiKey, req)
	if err != nil {
		panic(err)
	}
	fmt.Println("Response ID:", res.ID)
	fmt.Println("Text:", res.Output.Text)
}
