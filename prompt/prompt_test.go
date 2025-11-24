package prompt_test

import (
	"testing"

	"github.com/zlsgo/zllm/agent"
)

var llm agent.LLM

func TestMain(m *testing.M) {
	llm = agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
		oa.Stream = true
		oa.Temperature = 0
		oa.Model = "gpt-3.5-turbo"
	})

	m.Run()
}
