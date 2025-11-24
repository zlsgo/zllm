package zllm

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zpool"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

func TestBalancerCompleteLLM(t *testing.T) {
	if zutil.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("跳过测试 OPENAI")
	}

	tt := zlsgo.NewTest(t)
	nodes := zpool.NewBalancer[agent.LLM]()

	nodes.Add("gpt", agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
		oa.Model = "gpt"
	}), func(opts *zpool.BalancerNodeOptions) {
		opts.Weight = 100
		opts.Cooldown = 10000
	})

	nodes.Add("gpt-4o-mini", agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
		oa.Model = "gpt-4o-mini"
	}))

	p := message.NewPrompt(
		"现在时间是 2025-02-10，地点是 北京，我现在需要知道明天的天气情况",
		func(p *message.PromptOptions) {
			p.Rules = []string{"提取时间地点"}
			p.OutputFormat = message.CustomOutputFormat(map[string]string{"时间": "{}", "地点": "{}"})
		},
	)

	resp, err := BalancerCompleteLLMJSON(context.Background(), nodes, p)
	tt.NoError(err, true)

	tt.Equal("2025-02-11", resp.Get("时间").String())
	tt.Equal("北京", resp.Get("地点").String())

	nodes.WalkNodes(func(node agent.LLM, available bool) (normal bool) {
		tt.Log(node, available)
		return true
	})
}
