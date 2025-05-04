package clusterfuc

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/calamity-m/clusterfuc/pkg/agent"
	"github.com/calamity-m/clusterfuc/pkg/memoriser"
)

func TestAgentCreation(t *testing.T) {
	t.Run("nil agent config fails", func(t *testing.T) {
		_, err := NewAgent(nil)
		if err == nil {
			t.Errorf("expected err during openai creation but got nil")
		}

		_, err = NewAgent(nil)
		if err == nil {
			t.Errorf("expected err during openai creation but got nil")
		}
	})

	t.Run("agent config", func(t *testing.T) {
		agent, err := NewAgent(&AgentConfig{})
		if err != nil {
			t.Fatalf("did not expect err but got %v", err)
		}

		_, ok := agent.Memoriser.(*memoriser.NoOpMemoriser)
		if !ok {
			t.Errorf("expected NoOpMemoriser but got %#v instead", agent.Memoriser)
		}
	})
}

func TestExtendAgent(t *testing.T) {
	// TODO adding functions to the agent via
	// the extend function.
}

func TestAgentCall(t *testing.T) {
	// TODO test chain of parsing in/out
	// of an agent call.
}

func TestAgentAsFunction(t *testing.T) {
	// TODO test adding an agent itself
	// as a tool to another agent.
}

func TestAgentVerbosity(t *testing.T) {
	// TODO test verbosity of agent. Explicit
	// test due to the nature of this config
	// option. Displaying user input when
	// not wanted may be catastrophic.
}

func TestCrazy(t *testing.T) {
	type Arg struct {
		Name string `json:"name" jsonschema:"description=test name"`
		Ok   bool   `json:"ok" jsonschema:"description=test ok"`
	}

	fn := func(ctx context.Context, in Arg) (Arg, error) {
		fmt.Printf("yahoooo %s\n", in.Name)
		in.Name = fmt.Sprintf("%s + %s", in.Name, "TEST EXECUTED")
		return in, nil
	}

	oa, err := NewAgent(&AgentConfig{
		Model:   OpenAIChatGPT4oMini,
		Verbose: true,
		Client:  http.DefaultClient,
		Auth:    os.Getenv("AUTH"),
	})
	if err != nil {
		t.Fatalf("unexpected err - %#v", err)
	}

	RegisterTool(oa, "test", fn)

	gemini, err := NewAgent(&AgentConfig{
		Model:   Gemini2Flash,
		Verbose: true,
		Client:  http.DefaultClient,
		Auth:    os.Getenv("GEMINI_AUTH"),
	})
	if err != nil {
		t.Fatalf("unexpected err - %#v", err)
	}

	RegisterTool(gemini, "test", oa.Call)

	input := agent.AgentInput{
		Id: rand.Text(),
		UserInput: fmt.Sprintf(
			"req id: %s - input: %s",
			rand.Text(),
			"Please call the test function, and also tell me a random joke.",
		),
		Schema: nil,
	}

	o, err := gemini.Call(context.TODO(), input)

	fmt.Println(err)
	fmt.Println(o.Output)

}

func TestOpenAI(t *testing.T) {
	type Arg struct {
		Name string `json:"name" jsonschema:"description=test name"`
		Ok   bool   `json:"ok" jsonschema:"description=test ok"`
	}

	fn := func(ctx context.Context, in Arg) (Arg, error) {
		fmt.Printf("yahoooo %s\n", in.Name)
		in.Name = fmt.Sprintf("%s + %s", in.Name, "TEST EXECUTED")
		return in, nil
	}

	a, err := NewAgent(&AgentConfig{
		Model:   OpenAIChatGPT4oMini,
		Verbose: true,
		Client:  http.DefaultClient,
		Auth:    os.Getenv("AUTH"),
	})
	if err != nil {
		t.Fatalf("unexpected err - %#v", err)
	}

	err = RegisterTool(a, "test", fn)
	if err != nil {
		t.Fatalf("why fail - %s", err)
	}

	input := agent.AgentInput{
		Id: rand.Text(),
		UserInput: fmt.Sprintf(
			"req id: %s - input: %s",
			rand.Text(),
			"Please call the test function, and also tell me a random joke.",
		),
		Schema: nil,
	}

	o, err := a.Call(context.TODO(), input)

	fmt.Println(err)
	fmt.Println(o.Output)
}

func TestFreely(t *testing.T) {
	type Arg struct {
		Name string `json:"name" jsonschema:"description=this is aname"`
		Ok   bool   `json:"ok" jsonschema:"description=this is aname"`
	}

	fn := func(ctx context.Context, in Arg) (Arg, error) {
		fmt.Printf("yahoooo %s\n", in.Name)
		in.Name = fmt.Sprintf("%s + %s", in.Name, "TEST EXECUTED")
		return in, nil
	}

	a, err := NewAgent(&AgentConfig{
		Model:   Gemini2Flash,
		Verbose: true,
		Client:  http.DefaultClient,
		Auth:    os.Getenv("GEMINI_AUTH"),
	})
	if err != nil {
		t.Fatalf("unexpected err - %#v", err)
	}

	err = RegisterTool(a, "test", fn)
	if err != nil {
		t.Fatalf("why fail - %s", err)
	}

	input := agent.AgentInput{
		Id: rand.Text(),
		UserInput: fmt.Sprintf(
			"req id: %s - input: %s",
			rand.Text(),
			"I am testing function calling and flash 2.0's humour. If you don't have a tool to fulfil a request, do your best on your own. Please the test function, and also tell me a random joke.",
		),
		Schema: nil,
	}

	o, err := a.Call(context.TODO(), input)

	fmt.Println(err)
	fmt.Println(o.Output)
}
