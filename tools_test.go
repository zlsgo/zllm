package zllm

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

type mockToolRunner struct{}

func (mockToolRunner) Run(ctx context.Context, name, args string) (string, error) {
	j := zjson.Parse(args)
	if name == "echo" {
		return j.Get("text").String(), nil
	}
	return "", nil
}

type mockLLM struct{ step int }

var _ agent.LLM = (*mockLLM)(nil)

func (m *mockLLM) Generate(ctx context.Context, data []byte) (*zjson.Res, error) {
	if m.step == 0 {
		m.step++
		return zjson.ParseBytes([]byte(`{"choices":[{"message":{"tool_calls":[{"function":{"name":"echo","arguments":"{\"text\":\"hi\"}"}}]}}]}`)), nil
	}
	return zjson.ParseBytes([]byte(`{"choices":[{"message":{"content":"final: hi"}}]}`)), nil
}

func (m *mockLLM) Stream(ctx context.Context, data []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	ch := make(chan *zjson.Res, 1)
	res, _ := m.Generate(ctx, data)
	ch <- res
	close(ch)
	return ch, nil
}

func (m *mockLLM) PrepareRequest(msgs *message.Messages, opts ...func(ztype.Map) ztype.Map) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockLLM) ParseResponse(body *zjson.Res) (*agent.Response, error) {
	if body.Get("choices.0.message.tool_calls").Exists() {
		tools := []agent.Tool{{
			Name: body.Get("choices.0.message.tool_calls.0.function.name").String(),
			Args: body.Get("choices.0.message.tool_calls.0.function.arguments").String(),
		}}
		return &agent.Response{Tools: tools}, nil
	}
	content := body.Get("choices.0.message.content").Bytes()
	return &agent.Response{Content: content}, nil
}

func TestToolRunnerLoop(t *testing.T) {
	llm := &mockLLM{}
	p := message.NewPrompt("say hi via tool")

	ctx := WithToolRunner(context.Background(), mockToolRunner{})
	resp, err := CompleteLLM(ctx, llm, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "final: hi" {
		t.Fatalf("unexpected resp: %q", resp)
	}
}
