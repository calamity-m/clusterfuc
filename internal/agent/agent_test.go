package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/calamity-m/clusterfuc/internal/executable"
	"github.com/calamity-m/clusterfuc/internal/model"
)

type Arg struct {
	Name string `json:"name" jsonschema:"description=this is aname"`
	Ok   bool   `json:"ok" jsonschema:"description=this is aname"`
}

func TestAgent(t *testing.T) {

	fn := func(ctx context.Context, in Arg) (Arg, error) {
		fmt.Printf("yahoooo %s\n", in.Name)
		in.Name = fmt.Sprintf("%s + %s", in.Name, "TEST EXECUTED")
		return in, nil
	}

	agent := Agent[model.GeminiAiModel]{
		Model:   model.GeminiAiModel("gemini-2.0-flash"),
		Verbose: true,
		Client:  http.DefaultClient,
		Auth:    os.Getenv("AUTH")}

	agent.Functions = append(agent.Functions, executable.ExecuteableFunction("test", fn))

	out, err := agent.Functions[0].Executable.Execute(context.Background(), Arg{Name: "testing", Ok: true})
	if err != nil {
		t.Error(err)
	}

	fmt.Println(out)

	argZ, ok := out.(Arg)
	if !ok {
		t.Fatalf("broke")
	}

	fmt.Println(argZ)

	gen := rand.Text()
	fmt.Println("rand")
	fmt.Println(gen)
	r := fmt.Sprintf("req id: %s - input: %s", gen, "I am testing function calling and flash 2.0's humour. If you don't have a tool to fulfil a request, do your best on your own. Please the test function, and also tell me a random joke.")
	o, err := agent.callGeminiSchema(
		context.Background(),
		"a",
		r,
		nil,
	)

	fmt.Println(err)
	fmt.Println(o)
}
