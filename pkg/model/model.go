package model

const MAX_TOOLS_COUMT = 10

type OpenAiModel string
type GeminiAiModel string

// Type masturbation and overengineering in
// a very silly way
type AIModel interface {
	Model() string
}

func (m OpenAiModel) Model() string {
	return string(m)
}

func (m GeminiAiModel) Model() string {
	return string(m)
}
