package clusterfuc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/openai/openai-go"
)

type Aggs struct {
	Name string `json:"name" jsonschema:"description=this is aname"`
	Ok   bool   `json:"ok" jsonschema:"description=this is aname"`
}

func TestAgent(t *testing.T) {

	fn := func(ctx context.Context, in Aggs) (string, error) {
		fmt.Printf("yahoooo %s\n", in.Name)
		return in.Name, nil
	}

	opts := []OptsAgent[SingleAnswer]{
		WithTool[Aggs, string, SingleAnswer]("fn", fn),
		WithOpenAIClient[SingleAnswer](&openai.Client{}),
	}
	agent, err := NewOpenAIAgent(opts...)

	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	agent.Register("fn", Tool(fn))

	builder := strings.Builder{}

	err = json.NewEncoder(&builder).Encode(Aggs{Name: "yeeeheooo", Ok: true})
	if err != nil {
		t.Fatalf("failed to encode test input: %v", err)
	}

	buffie := bytes.NewBuffer([]byte{})

	err = agent.GetExecutables()["fn"].Executable.Execute(strings.NewReader(builder.String()), buffie)
	if err != nil {
		t.Fatalf("failed to execute tool: %v", err)
	}

	fmt.Printf("%#v\n", agent.GetExecutables()["fn"].Definition.Properties)

	w := bytes.Buffer{}
	if err := json.NewEncoder(&w).Encode(agent.GetExecutables()["fn"].Definition.Properties); err != nil {

	}

	fmt.Println(w.String())
}
