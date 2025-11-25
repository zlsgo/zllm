// Package zllm 提供 LLM 底层实现，封装不同提供商的 API 差异
package zllm

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

// 默认配置和系统参数
const (
	DefaultTimeout       = 60 * time.Second       // 默认超时时间
	DefaultMaxToolIter   = 3                      // 默认最大工具迭代次数
	MaxRetryInterval     = 30 * time.Second       // 最大重试间隔
	BaseRetryInterval    = 100 * time.Millisecond // 基础重试间隔
	ShortRetryInterval   = 200 * time.Millisecond // 短重试间隔
	SecondRetryInterval  = time.Second            // 秒级重试间隔
	MaxConsecutiveErrors = 5                      // 最大连续错误次数
	MinJSONLength        = 2                      // JSON 最小长度
	JSONLeftBrace        = '{'                    // JSON 左括号
	JSONRightBrace       = '}'                    // JSON 右括号
)

// promptMsg 定义支持的消息类型约束，用于泛型函数
type promptMsg interface {
	*message.Prompt | *message.Messages
}

// 上下文键类型定义，用于 context.WithValue 的类型安全
type (
	allowToolsKey          struct{} // 工具使用权限键
	toolRunnerKey          struct{} // 工具执行器键
	toolResultFormatterKey struct{} // 工具结果格式化器键
	timeoutKey             struct{} // 超时时间键
	toolIterKey            struct{} // 工具迭代次数键
)

// WithAllowTools 在上下文中设置是否允许使用工具
func WithAllowTools(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, allowToolsKey{}, allow)
}

// isAllowTools 从上下文中获取工具使用权限设置
func isAllowTools(ctx context.Context) bool {
	if v, ok := ctx.Value(allowToolsKey{}).(bool); ok {
		return v
	}
	return true // 默认允许使用工具
}

// ToolRunner 定义工具执行接口，用于处理 LLM 工具调用
type ToolRunner interface {
	Run(ctx context.Context, name, args string) (string, error)
}

// WithToolRunner 在上下文中设置工具执行器
func WithToolRunner(ctx context.Context, r ToolRunner) context.Context {
	return context.WithValue(ctx, toolRunnerKey{}, r)
}

// getToolRunner 从上下文中获取工具执行器
func getToolRunner(ctx context.Context) ToolRunner {
	if v, ok := ctx.Value(toolRunnerKey{}).(ToolRunner); ok {
		return v
	}
	return nil
}

// ToolResult 定义工具执行的返回结果
type ToolResult struct {
	Name   string // 工具名称
	Args   string // 工具调用参数
	Result string // 工具执行结果
	Err    string // 错误信息，如果没有错误则为空
}

// ToolResultFormatter 定义工具结果格式化器函数类型
type ToolResultFormatter func(results []ToolResult) string

// WithTimeout 在上下文中设置请求超时时间
func WithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, timeoutKey{}, timeout)
}

// getTimeout 从上下文中获取超时时间设置
func getTimeout(ctx context.Context) time.Duration {
	if v, ok := ctx.Value(timeoutKey{}).(time.Duration); ok && v > 0 {
		return v
	}
	return DefaultTimeout // 使用默认超时
}

// WithToolResultFormatter 在上下文中设置工具结果格式化器
func WithToolResultFormatter(ctx context.Context, f ToolResultFormatter) context.Context {
	return context.WithValue(ctx, toolResultFormatterKey{}, f)
}

// getToolResultFormatter 从上下文中获取工具结果格式化器
func getToolResultFormatter(ctx context.Context) ToolResultFormatter {
	if v, ok := ctx.Value(toolResultFormatterKey{}).(ToolResultFormatter); ok && v != nil {
		return v
	}
	return defaultToolResultFormatter // 使用默认格式化器
}

// defaultToolResultFormatter 默认的工具结果格式化器
func defaultToolResultFormatter(results []ToolResult) string {
	arr := make([]ztype.Map, 0, len(results))
	for i := range results {
		item := ztype.Map{
			"tool":   results[i].Name,
			"args":   tryJSON(results[i].Args),
			"result": tryJSON(results[i].Result),
		}
		if results[i].Err != "" {
			item["error"] = results[i].Err
		}
		arr = append(arr, item)
	}
	return ztype.ToString(arr)
}

// tryJSON 尝试将字符串解析为 JSON，返回解析后的对象或原始字符串
func tryJSON(s string) any {
	if len(s) == 0 {
		return s
	}
	r := zjson.Parse(s)
	if r.Exists() {
		if r.IsObject() || r.IsArray() {
			return r.Value()
		}
		return r.Value()
	}
	return s
}

// CompleteLLM 向 LLM 发送提示并返回完整响应，支持工具执行、重试机制和超时处理
func CompleteLLM[T promptMsg](ctx context.Context, llm agent.LLM, msg T, options ...func(ztype.Map) ztype.Map) (string, error) {
	timeout := getTimeout(ctx)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		messages *message.Messages
		err      error
	)

	switch v := any(msg).(type) {
	case *message.Prompt:
		messages, err = v.ConvertToMessages()
		if err != nil {
			return "", err
		}
	case *message.Messages:
		messages = v
	default:
		return "", fmt.Errorf("invalid prompt type: %T", msg)
	}

	content, err := llm.PrepareRequest(messages, options...)
	if err != nil {
		return "", err
	}

	parse, _, err := processLLMInteraction(ctx, llm, messages, bytes.TrimSpace(content), options...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			runtime.Log("LLM request timeout after", timeout)
			return "", fmt.Errorf("LLM request timeout after %v", timeout)
		}
		return "", err
	}

	err = messages.AppendAssistant(parse)

	return parse, err
}

// CompleteLLMJSON 向 LLM 发送提示并返回解析后的 JSON 映射响应
func CompleteLLMJSON[T promptMsg](ctx context.Context, llm agent.LLM, msg T, options ...func(ztype.Map) ztype.Map) (ztype.Map, error) {
	resp, err := CompleteLLM(ctx, llm, msg, options...)
	if err != nil {
		return nil, err
	}

	return parseJSONResponse(resp)
}

// processLLMInteraction 处理与 LLM 的交互，处理工具调用和重试
func processLLMInteraction(ctx context.Context, llm agent.LLM, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) (parse string, rawContext []byte, err error) {
	return processLLMInteractionWithValidation(ctx, llm, messages, body, options...)
}

// WithMaxToolIterations 在上下文中设置工具迭代的最大次数
func WithMaxToolIterations(ctx context.Context, n int) context.Context {
	if n < 0 {
		n = 0
	}
	return context.WithValue(ctx, toolIterKey{}, n)
}

// getMaxToolIterations 从上下文中获取工具迭代的最大次数
func getMaxToolIterations(ctx context.Context) int {
	if v, ok := ctx.Value(toolIterKey{}).(int); ok && v >= 0 {
		return v
	}
	return DefaultMaxToolIter
}
