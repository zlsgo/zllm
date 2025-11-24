package agent_test

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

var gemini = agent.NewGemini(func(oa *agent.GeminiOptions) {
	oa.Stream = false
})

func TestNewGemini(t *testing.T) {
	if zutil.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	runtime.SetDebug(true)
	tt := zlsgo.NewTest(t)

	tt.Run("base", func(tt *zlsgo.TestUtil) {
		resp, err := gemini.Generate(context.Background(), []byte("{}"))
		tt.Log(resp)
		tt.NoError(err)
	})

	tt.Run("prompt", func(tt *zlsgo.TestUtil) {
		prompt := message.NewPrompt("帮我写一篇关于人工智能的文章，其他信息的你自行发挥。", func(po *message.PromptOptions) {
			po.MaxLength = 20
			po.Rules = []string{"根据用户输入生成回答"}
		})

		messages, err := prompt.ConvertToMessages()
		tt.NoError(err, true)
		tt.Log(messages.String())

		data, err := gemini.PrepareRequest(
			messages,
			func(m ztype.Map) ztype.Map {
				m.Set("temperature", 0.7)
				return m
			},
		)
		tt.Log(string(data))
		tt.NoError(err, true)

		resp, err := gemini.Generate(context.Background(), data)
		tt.Log(resp)
		tt.NoError(err)

		parse, err := gemini.ParseResponse(resp)
		tt.NoError(err, true)
		tt.Log(string(parse.Content))
	})
}
