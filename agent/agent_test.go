package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	runtime_errors "github.com/zlsgo/zllm/runtime/errors"
)

func TestLLMAgentInterface(t *testing.T) {
	tt := zlsgo.NewTest(t)

	providers := []struct {
		name     string
		provider agent.LLM
	}{
		{
			name:     "OpenAI",
			provider: agent.NewOpenAI(),
		},
		{
			name:     "Deepseek",
			provider: agent.NewDeepseek(),
		},
		{
			name:     "Ollama",
			provider: agent.NewOllama(),
		},
	}

	for _, p := range providers {
		tt.Run(p.name+" Interface Test", func(tt *zlsgo.TestUtil) {
			messages := message.NewMessages()
			messages.AppendUser("Hello, world!")

			data, err := p.provider.PrepareRequest(messages)
			tt.NoError(err, true)
			tt.Log("Prepared request length:", len(data))

			mockResponse := `{
				"choices": [{
					"message": {
						"content": "Hello! How can I help you today?",
						"role": "assistant"
					}
				}]
			}`

			resp, err := testParseResponse(p.provider, mockResponse)
			if err == nil {
				tt.Log("Parsed response content length:", len(resp.Content))
				tt.Equal(len(resp.Content) > 0, true)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tt := zlsgo.NewTest(t)

	testCases := []struct {
		name        string
		errorCode   runtime_errors.ErrorCode
		expectRetry bool
	}{
		{"Rate Limited", runtime_errors.ErrRateLimited, true},
		{"Server Error", runtime_errors.ErrServer, true},
		{"Unauthorized", runtime_errors.ErrUnauthorized, false},
		{"Quota Exceeded", runtime_errors.ErrQuotaExceeded, false},
		{"Model Not Found", runtime_errors.ErrModelNotFound, false},
		{"Token Limit", runtime_errors.ErrTokenLimit, false},
	}

	for _, tc := range testCases {
		tt.Run(tc.name, func(tt *zlsgo.TestUtil) {
			err := runtime_errors.NewLLMError(tc.errorCode, "test error")
			llmErr := err.(runtime_errors.LLMError)
			tt.Equal(llmErr.IsRetryable(), tc.expectRetry)
		})
	}
}

func TestStreamHandling(t *testing.T) {
	tt := zlsgo.NewTest(t)

	providers := []struct {
		name     string
		provider agent.LLM
	}{
		{
			name:     "OpenAI",
			provider: agent.NewOpenAI(func(o *agent.OpenAIOptions) { o.Stream = true }),
		},
		{
			name:     "Deepseek",
			provider: agent.NewDeepseek(func(o *agent.DeepseekOptions) { o.Stream = true }),
		},
		{
			name:     "Ollama",
			provider: agent.NewOllama(func(o *agent.OllamaOptions) { o.Stream = true }),
		},
	}

	for _, p := range providers {
		tt.Run(p.name+" Stream Test", func(tt *zlsgo.TestUtil) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			messages := message.NewMessages()
			messages.AppendUser("Say hello")

			data, err := p.provider.PrepareRequest(messages)
			tt.NoError(err)

			done, err := p.provider.Stream(ctx, data, func(chunk string, data []byte) {
				tt.Log("Stream chunk:", chunk)
			})

			if err == nil {
				tt.NotNil(done)
			}
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	tt := zlsgo.NewTest(t)

	openaiProvider := agent.NewOpenAI(func(o *agent.OpenAIOptions) {
		o.Model = "gpt-3.5-turbo"
		o.Temperature = 0.8
		o.MaxRetries = 5
	})

	messages := message.NewMessages()
	messages.AppendUser("Test configuration")

	data, err := openaiProvider.PrepareRequest(messages)
	tt.NoError(err, true)
	tt.Log("OpenAI request data:", string(data))

	deepseekProvider := agent.NewDeepseek(func(o *agent.DeepseekOptions) {
		o.Model = "deepseek-coder"
		o.Temperature = 0.1
	})

	data, err = deepseekProvider.PrepareRequest(messages)
	tt.NoError(err, true)
	tt.Log("Deepseek request data:", string(data))

	ollamaProvider := agent.NewOllama(func(o *agent.OllamaOptions) {
		o.Model = "llama2"
		o.Temperature = 0.5
	})

	data, err = ollamaProvider.PrepareRequest(messages)
	tt.NoError(err, true)
	tt.Log("Ollama request data:", string(data))
}

func testParseResponse(provider agent.LLM, jsonResponse string) (*agent.Response, error) {
	resp := zjson.ParseBytes([]byte(jsonResponse))
	return provider.ParseResponse(resp)
}
