package openai

import (
	"context"
	"testing"
)

func TestGenerate(t *testing.T) {
	oa := OpenAI{}

	boz, out, err := oa.Generate(context.TODO(), &CreateResponse{}, nil)
	if err != nil {
		t.Errorf("%s?", err)
	}

	if out != "" {
		t.Errorf("%s?", out)
	}

	if boz == nil {
		t.Errorf("aaa?")
	}
}
