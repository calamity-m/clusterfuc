package gemini

import "errors"

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
			return []Content{}, errors.New("input contains non gemini content")
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
