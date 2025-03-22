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

var openai = NewOpenAIProvider(func(oa *OpenAIOptions) {
	oa.Stream = true
	oa.BaseURL = zutil.Getenv("OPENAI_BASE_URL")
	oa.APIKey = zutil.Getenv("OPENAI_API_KEY")
})

func TestNewOpenAIProvider(t *testing.T) {
	tt := zlsgo.NewTest(t)

	prompt := message.NewPrompt("帮我写一篇关于人工智能的文章，其他信息的你自行发挥。", func(po *message.PromptOptions) {
		po.MaxLength = 20
		po.Rules = []string{"根据用户输入生成回答"}
	})

	messages, err := prompt.ConvertToMessages()
	tt.NoError(err, true)
	tt.Log(messages.String())

	data, err := openai.PrepareRequest(
		messages,
		func(m ztype.Map) ztype.Map {
			m.Set("temperature", 0.7)
			return m
		},
	)
	tt.Log(string(data))
	tt.NoError(err, true)

	resp, err := openai.Generate(context.Background(), data)
	tt.Log(resp)
	tt.NoError(err, true)

	parseResponse, err := openai.ParseResponse(resp)
	tt.Log(parseResponse)
	tt.NoError(err, true)

	str, err := messages.ParseFormat(parseResponse.Content)
	tt.Log(str)
	tt.NoError(err, true)

	json := zjson.ParseBytes(str)
	tt.Log(json.Get("Assistant").String())
}
