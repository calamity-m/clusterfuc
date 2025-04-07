package clusterfuc

import (
	"context"
	"errors"
	"log/slog"
)

func (a *Agent[Out]) callGeminiModel(ctx context.Context, input string) (string, error) {
	slog.DebugContext(ctx, "calling gemini model", slog.String("model", a.model))
	if a.verbose {
		slog.DebugContext(ctx, "input", slog.String("input", input))
	}

	return "", errors.ErrUnsupported
}

func FormatRequest(prompt string, input string, tools map[string]executableTool) string {
	return ""
}
