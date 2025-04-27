package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/calamity-m/clusterfuc/pkg/executable"
)

// CreateRawRequestBody constructs the request body for the Gemini API. Raw is considered
// because it has no input added to it
func CreateRawRequestBody(systemPrompt string, schema *executable.JSONSchemaSubset, executables []executable.Executable[any, any]) (*RequestBody, error) {
	body := &RequestBody{}

	body.SystemInstruction.Text = systemPrompt

	if schema != nil {
		body.GenerationConfig.ResponseSchema.Properties = schema.Properties
		body.GenerationConfig.ResponseSchema.Properties = schema.Required
	}

	functions := make([]FunctionDeclaration, len(executables))
	for i, exec := range executables {
		functions[i] = FunctionDeclaration{
			Name:        exec.Name,
			Description: exec.Name,
			Parameters: map[string]any{
				"type":       "object",
				"properties": exec.Definition.Properties,
				"required":   exec.Definition.Required,
			},
		}
	}
	body.Tools = []Tool{{FunctionDeclarations: functions}}

	return body, nil
}

func GenerateContent(ctx context.Context, client *http.Client, url string, body *RequestBody) (ResponseBody, error) {

	data, err := json.Marshal(body)
	if err != nil {
		return ResponseBody{}, err
	}

	r, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return ResponseBody{}, err
	}

	resp, err := client.Do(r)
	if err != nil {
		return ResponseBody{}, err
	}

	if resp.StatusCode != 200 {
		failed, _ := io.ReadAll(resp.Body)
		fmt.Println(string(failed))
		fmt.Printf("%#v\n", resp)
		return ResponseBody{}, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseBody{}, err
	}

	var generated ResponseBody
	err = json.Unmarshal(respData, &generated)
	if err != nil {
		return ResponseBody{}, err
	}

	return generated, nil
}

func AppendContentsFailure(contents []Content, functionName string, failure string) []Content {
	failed := Content{
		Role: "user",
		Parts: []Part{{
			FunctionResponse: FunctionResponse{
				Name: functionName,
				Response: map[string]any{
					"success":       false,
					"failureReason": failure,
				},
			},
		},
		}}

	contents = append(contents, failed)

	return contents
}

func AppendFunctionResponse(contents []Content, functionName string, response any) []Content {
	new := Content{
		Role: "user",
		Parts: []Part{{
			FunctionResponse: FunctionResponse{
				Name:     functionName,
				Response: response,
			},
		},
		}}

	contents = append(contents, new)

	return contents
}
