package clusterfuc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
)

func (a *Agent[Out]) callGeminiModel(ctx context.Context, input []string) (string, error) {
	slog.DebugContext(ctx, "calling gemini model", slog.String("model", a.model))
	if a.verbose {
		slog.DebugContext(ctx, "input", slog.Any("input", input))
	}

	return "", errors.ErrUnsupported
}

func formatRequest(prompt string, input []string, tools map[string]executableTool) (string, error) {

	functionDeclarations := make([]map[string]any, 0, len(tools))
	for name, tool := range tools {
		functionDeclarations = append(functionDeclarations, map[string]any{
			"name":        name,
			"description": tool.Definition,
			"parameters": map[string]any{
				"type":       "object",
				"properties": tool.Definition.Properties,
				"required":   tool.Definition.Required,
			},
		})
	}

	request := map[string]any{
		"contents": map[string]any{
			"role":  "user",
			"parts": input,
		},
		"system_instruction": map[string]any{
			"parts": []string{prompt},
		},
		"tools": functionDeclarations,
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	return string(requestJSON), nil
}
