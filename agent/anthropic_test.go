package agent_test

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

var anthropic = agent.NewAnthropic(func(a *agent.AnthropicOptions) {
	a.Stream = false
})

func TestNewAnthropic(t *testing.T) {
	if zutil.Getenv("ANTHROPIC_BASE_URL") == "" {
		t.Skip("跳过测试 ANTHROPIC")
	}

	tt := zlsgo.NewTest(t)

	prompt := message.NewPrompt("帮我写一篇关于人工智能的文章", func(po *message.PromptOptions) {
		po.MaxLength = 20
		po.Rules = []string{"根据用户输入生成回答"}
	})

	messages, err := prompt.ConvertToMessages()
	tt.NoError(err, true)
	tt.Log(messages.String())

	data, err := anthropic.PrepareRequest(
		messages,
		func(m ztype.Map) ztype.Map {
			m.Set("temperature", 0.7)
			return m
		},
	)
	tt.NoError(err, true)
	tt.Log(string(data))

	resp, err := anthropic.Generate(context.Background(), data)
	tt.NoError(err, true)

	parse, err := anthropic.ParseResponse(resp)
	tt.NoError(err, true)
	tt.Log(string(parse.Content))
}
