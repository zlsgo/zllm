package zllm

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

func TestToolRunnerError(t *testing.T) {
	// 创建一个模拟的 LLM 代理，返回工具调用
	mockLLM := &mockAgentWithTools{
		tools: []agent.Tool{
			{Name: "test_tool", Args: `{"param": "value"}`},
		},
	}

	// 测试没有配置 ToolRunner 的情况
	messages := message.NewMessages()
	messages.AppendUser("请调用测试工具")

	// 不设置 ToolRunner
	ctx := context.Background()

	_, err := CompleteLLM(ctx, mockLLM, messages)
	
	// 验证返回了合适的错误信息
	if err == nil {
		t.Error("Expected error when ToolRunner is not configured")
	}
	
	expectedError := "tool runner not configured: use WithToolRunner() to set up tool execution for 1 tool(s)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// 测试禁用工具调用的情况
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
	// 测试多个工具的情况
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
	// 测试没有工具调用的正常情况
	mockLLM := &mockAgentWithTools{
		tools: []agent.Tool{}, // 空工具列表
	}

	messages := message.NewMessages()
	messages.AppendUser("正常对话，不需要调用工具")

	ctx := context.Background()

	// 这种情况应该正常处理，不返回工具错误
	_, err := CompleteLLM(ctx, mockLLM, messages)
	
	// 由于我们的模拟代理简化处理，这里不会产生工具错误
	// 在实际场景中，如果LLM返回没有工具的响应，应该正常处理
	// 这里我们主要验证不会因为工具配置问题而失败
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

// 模拟代理，用于测试工具调用错误处理
type mockAgentWithTools struct {
	agent.LLMAgent
	tools []agent.Tool
}

func (m *mockAgentWithTools) Generate(ctx context.Context, data []byte) (*zjson.Res, error) {
	// 简化处理 - 直接返回空结果
	// 在实际使用中，这里应该返回包含工具调用的JSON数据
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
