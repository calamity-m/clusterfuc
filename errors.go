package clusterfuc

import (
	"errors"

	"github.com/calamity-m/clusterfuc/internal/agent"
	"github.com/calamity-m/clusterfuc/internal/gemini"
)

var (
	ErrExceededMaxToolCount = errors.New("exceeded max tool count")
	ErrAgentOptInvalid      = errors.New("invalid agent option was passed")
	ErrModelUnmatched       = agent.ErrModelUnmatched
	ErrInvalidGeminiContent = gemini.ErrInvalidGeminiContent
)
