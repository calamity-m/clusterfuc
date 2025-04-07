package clusterfuc

const MAX_TOOLS_COUMT = 10

type OpenAIModel string
type GeminiModel string
type AIModel[T OpenAIModel | GeminiModel] interface {
	Model() string
}

func (m OpenAIModel) Model() string {
	return string(m)
}

func (m GeminiModel) Model() string {
	return string(m)
}

const (
	OpenAIChatGPT4o     OpenAIModel = "gpt-4o"
	OpenAIChatGPT4oMini OpenAIModel = "gpt-4o-mini"

	Gemini2Flash     GeminiModel = "gemini-2.0-flash"
	Gemini2FlashLite GeminiModel = "gemini-2.0-flash-lite"
)
