package agent

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
)

var deepseek = NewDeepseekProvider(func(oa *DeepseekOptions) {
	oa.Stream = true
	oa.APIKey = zutil.Getenv("DEEPSEEK_API_KEY")
	oa.Debug = true
})

func TestNewDeepseekProvider(t *testing.T) {
	tt := zlsgo.NewTest(t)

	prompt := message.NewPrompt("帮我写一篇关于人工智能的文章", func(po *message.PromptOptions) {
		po.MaxLength = 20
		po.Rules = []string{"根据用户输入生成回答"}
	})

	messages, err := prompt.ConvertToMessages()
	tt.NoError(err, true)
	tt.Log(messages.String())

	data, err := deepseek.PrepareRequest(
		messages,
		ztype.Map{"temperature": 0.7},
	)
	tt.Log(string(data))
	tt.NoError(err, true)

	resp, err := deepseek.Generate(context.Background(), data)
	tt.Log(resp)
	tt.NoError(err, true)

	parseResponse, err := deepseek.ParseResponse(resp)
	tt.Log(parseResponse)
	tt.NoError(err, true)

	str, err := messages.ParseFormat(parseResponse.Content)
	tt.Log(str)
	tt.NoError(err, true)

	json := zjson.ParseBytes(str)
	tt.Log(json.Get("Assistant").String())
}
