package clusterfuc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
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
		WithTool[Aggs, string, SingleAnswer]("fn", "description", fn),
	}
	agent, err := NewAgent(opts...)

	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	agent.RegisterTool("fn", Tool(fn, "a tool test"))

	builder := strings.Builder{}

	err = json.NewEncoder(&builder).Encode(Aggs{Name: "yeeeheooo", Ok: true})
	if err != nil {
		t.Fatalf("failed to encode test input: %v", err)
	}

	buffie := bytes.NewBuffer([]byte{})

	err = agent.ListTools()["fn"].Executable.Execute(strings.NewReader(builder.String()), buffie)
	if err != nil {
		t.Fatalf("failed to execute tool: %v", err)
	}

	fmt.Printf("%#v\n", agent.ListTools()["fn"].Definition.Properties)

	w := bytes.Buffer{}
	if err := json.NewEncoder(&w).Encode(agent.ListTools()["fn"].Definition.Properties); err != nil {

	}

	fmt.Println(w.String())
}
