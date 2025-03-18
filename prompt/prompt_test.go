package prompt_test

import (
	"testing"

	"github.com/zlsgo/zllm/agent"
)

var llm agent.LLMAgent

func TestMain(m *testing.M) {
	// SetLogLevel(zlog.LogDebug)

	llm = agent.NewOpenAIProvider(func(oa *agent.OpenAIOptions) {
		oa.Stream = true
		oa.Temperature = 0
		oa.Model = "gpt-3.5-turbo"
	})

	m.Run()
}
