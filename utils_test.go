package zllm

import (
	"context"
	"testing"
	"time"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

func TestParseJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "ValidJSON",
			input: `{"name": "test", "value": 123}`,
		},
		{
			name:  "EmptyString",
			input: "",
		},
		{
			name:  "NonJSONString",
			input: "这是一个普通文本",
		},
		{
			name:  "ShortJSON",
			input: "{}",
		},
		{
			name:  "JSONArray",
			input: `[{"name": "test"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSONResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("parseJSONResponse() returned nil result")
			}
		})
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	t.Run("WithAllowTools", func(t *testing.T) {
		if got := isAllowTools(ctx); !got {
			t.Error("isAllowTools() default should be true")
		}

		ctxFalse := WithAllowTools(ctx, false)
		if got := isAllowTools(ctxFalse); got {
			t.Error("isAllowTools() should be false")
		}

		ctxTrue := WithAllowTools(ctx, true)
		if got := isAllowTools(ctxTrue); !got {
			t.Error("isAllowTools() should be true")
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		timeout := getTimeout(ctx)
		if timeout != DefaultTimeout {
			t.Errorf("getTimeout() = %v, want %v", timeout, DefaultTimeout)
		}

		customTimeout := 30 * time.Second
		ctxTimeout := WithTimeout(ctx, customTimeout)
		timeout = getTimeout(ctxTimeout)
		if timeout != customTimeout {
			t.Errorf("getTimeout() = %v, want %v", timeout, customTimeout)
		}
	})

	t.Run("WithMaxToolIterations", func(t *testing.T) {
		if got := getMaxToolIterations(ctx); got != DefaultMaxToolIter {
			t.Errorf("getMaxToolIterations() = %v, want %v", got, DefaultMaxToolIter)
		}

		customIter := 5
		ctxIter := WithMaxToolIterations(ctx, customIter)
		if got := getMaxToolIterations(ctxIter); got != customIter {
			t.Errorf("getMaxToolIterations() = %v, want %v", got, customIter)
		}

		ctxNegative := WithMaxToolIterations(ctx, -1)
		if got := getMaxToolIterations(ctxNegative); got != 0 {
			t.Errorf("getMaxToolIterations() with negative should be 0, got %v", got)
		}
	})
}

func TestConstants(t *testing.T) {
	if DefaultTimeout <= 0 {
		t.Error("DefaultTimeout should be positive")
	}

	if DefaultMaxToolIter <= 0 {
		t.Error("DefaultMaxToolIter should be positive")
	}

	if MinJSONLength != 2 {
		t.Errorf("MinJSONLength = %v, want %v", MinJSONLength, 2)
	}
}

func TestToolRunnerError(t *testing.T) {
	mockLLM := &mockAgentWithTools{
		tools: []agent.Tool{
			{Name: "test_tool", Args: `{"param": "value"}`},
		},
	}

	messages := message.NewMessages()
	messages.AppendUser("请调用测试工具")

	ctx := context.Background()

	_, err := CompleteLLM(ctx, mockLLM, messages)

	if err == nil {
		t.Error("Expected error when ToolRunner is not configured")
	}

	expectedError := "tool runner not configured: use WithToolRunner() to set up tool execution for 1 tool(s)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	ctxNoTools := WithAllowTools(context.Background(), false)
	_, err = CompleteLLM(ctxNoTools, mockLLM, messages)

	if err == nil {
		t.Error("Expected error when tools are not allowed")
	}

	if !containsString(err.Error(), "tools not supported") {
		t.Errorf("Expected error to contain 'tools not supported', got '%s'", err.Error())
	}
}

func TestToolRunnerMultipleTools(t *testing.T) {
	mockLLM := &mockAgentWithTools{
		tools: []agent.Tool{
			{Name: "tool1", Args: `{"param": "value1"}`},
			{Name: "tool2", Args: `{"param": "value2"}`},
			{Name: "tool3", Args: `{"param": "value3"}`},
		},
	}

	messages := message.NewMessages()
	messages.AppendUser("请调用多个测试工具")

	ctx := context.Background()

	_, err := CompleteLLM(ctx, mockLLM, messages)

	if err == nil {
		t.Error("Expected error when ToolRunner is not configured")
	}

	expectedError := "tool runner not configured: use WithToolRunner() to set up tool execution for 3 tool(s)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestToolRunnerNoTools(t *testing.T) {
	mockLLM := &mockAgentWithTools{
		tools: []agent.Tool{}, // 空工具列表
	}

	messages := message.NewMessages()
	messages.AppendUser("正常对话，不需要调用工具")

	ctx := context.Background()

	_, err := CompleteLLM(ctx, mockLLM, messages)

	if err != nil && containsString(err.Error(), "tool runner not configured") {
		t.Errorf("Unexpected tool runner error when no tools are returned: %s", err.Error())
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type mockAgentWithTools struct {
	agent.LLM
	tools []agent.Tool
}

func (m *mockAgentWithTools) Generate(ctx context.Context, data []byte) (*zjson.Res, error) {
	return &zjson.Res{}, nil
}

func (m *mockAgentWithTools) ParseResponse(resp *zjson.Res) (*agent.Response, error) {
	return &agent.Response{
		Content: []byte(""),
		Tools:   m.tools,
	}, nil
}

func (m *mockAgentWithTools) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockAgentWithTools) Stream(ctx context.Context, data []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	done := make(chan *zjson.Res)
	close(done)
	return done, nil
}
