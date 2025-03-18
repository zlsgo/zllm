package agent_test

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

func TestNewOllamaProvider(t *testing.T) {
	ollama := "http://127.0.0.1:11434"
	tt := zlsgo.NewTest(t)

	if _, err := zhttp.Get(ollama + "/api/tags"); err != nil {
		tt.Log("跳过 Ollama 测试")
		return
	}

	llm := agent.NewOllamaProvider(func(oo *agent.OllamaOptions) {
		oo.Model = "qwen2.5:0.5b"
		oo.BaseURL = ollama
	})

	prompt := message.NewPrompt("帮我写一篇关于人工智能的文章，其他信息的你自行发挥。", func(po *message.PromptOptions) {
		po.MaxLength = 35
	})

	messages, err := prompt.ConvertToMessages()
	tt.NoError(err, true)

	data, err := llm.PrepareRequest(
		messages,
		ztype.Map{"temperature": 0.3},
	)
	tt.Log(string(data))
	tt.NoError(err, true)

	resp, err := llm.Generate(context.Background(), data)
	tt.Log(resp)
	tt.NoError(err, true)

	parseResponse, err := llm.ParseResponse(resp)
	tt.Log(string(parseResponse.Content))
	tt.NoError(err, true)

	str, err := messages.ParseFormat(parseResponse.Content)
	tt.NoError(err, true)
	tt.Log(string(str))

	json := zjson.ParseBytes(str)
	tt.Log(json.Get("Assistant").String())
}
