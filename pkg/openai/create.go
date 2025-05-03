package openai

import "encoding/json"

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
