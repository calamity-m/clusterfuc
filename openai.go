package clusterfuc

import (
	"context"
	"errors"
	"log/slog"
)

func (a *Agent[Out]) callOpenAIModel(ctx context.Context, input string) (string, error) {
	slog.DebugContext(ctx, "calling openai model", slog.String("model", a.model))
	if a.verbose {
		slog.DebugContext(ctx, "input", slog.String("input", input))
	}

	return "", errors.ErrUnsupported
}
