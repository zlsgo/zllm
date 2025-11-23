package zllm

import (
	"context"
	"fmt"
	"time"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

const (
	DefaultTimeout       = 60 * time.Second
	DefaultMaxToolIter   = 3
	MaxRetryInterval     = 30 * time.Second
	BaseRetryInterval    = 100 * time.Millisecond
	ShortRetryInterval   = 200 * time.Millisecond
	SecondRetryInterval  = time.Second
	MaxConsecutiveErrors = 5
	MinJSONLength        = 2
	JSONLeftBrace        = '{'
	JSONRightBrace       = '}'
)

type promptMsg interface {
	*message.Prompt | *message.Messages
}

type (
	allowToolsKey          struct{}
	toolRunnerKey          struct{}
	toolResultFormatterKey struct{}
	timeoutKey             struct{}
	toolIterKey            struct{}
)

// WithAllowTools 控制是否允许工具调用功能。默认为 true，允许执行工具调用；设为 false 时，如果 LLM 返回工具调用将返回错误。
func WithAllowTools(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, allowToolsKey{}, allow)
}

func isAllowTools(ctx context.Context) bool {
	if v, ok := ctx.Value(allowToolsKey{}).(bool); ok {
		return v
	}
	return true
}

type ToolRunner interface {
	Run(ctx context.Context, name, args string) (string, error)
}

func WithToolRunner(ctx context.Context, r ToolRunner) context.Context {
	return context.WithValue(ctx, toolRunnerKey{}, r)
}

func getToolRunner(ctx context.Context) ToolRunner {
	if v, ok := ctx.Value(toolRunnerKey{}).(ToolRunner); ok {
		return v
	}
	return nil
}

type ToolResult struct {
	Name   string
	Args   string
	Result string
	Err    string
}

type ToolResultFormatter func(results []ToolResult) string

// WithTimeout 设置请求超时时间
func WithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, timeoutKey{}, timeout)
}

func getTimeout(ctx context.Context) time.Duration {
	if v, ok := ctx.Value(timeoutKey{}).(time.Duration); ok && v > 0 {
		return v
	}
	return DefaultTimeout
}

func WithToolResultFormatter(ctx context.Context, f ToolResultFormatter) context.Context {
	return context.WithValue(ctx, toolResultFormatterKey{}, f)
}

func getToolResultFormatter(ctx context.Context) ToolResultFormatter {
	if v, ok := ctx.Value(toolResultFormatterKey{}).(ToolResultFormatter); ok && v != nil {
		return v
	}
	return defaultToolResultFormatter
}

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

func tryJSON(s string) any {
	if len(s) == 0 {
		return s
	}
	r := zjson.Parse(s)
	// Prefer object/array; otherwise fallback to scalar or raw string
	if r.Exists() {
		if r.IsObject() || r.IsArray() {
			return r.Value()
		}
		return r.Value()
	}
	return s
}

func CompleteLLM[T promptMsg](ctx context.Context, llm agent.LLMAgent, msg T, options ...func(ztype.Map) ztype.Map) (string, error) {
	// 添加超时保护
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

	parse, _, err := processLLMInteraction(ctx, llm, messages, content, options...)
	if err != nil {
		// 检查是否是超时错误
		if ctx.Err() == context.DeadlineExceeded {
			runtime.Log("LLM request timeout after", timeout)
			return "", fmt.Errorf("LLM request timeout after %v", timeout)
		}
		return "", err
	}

	err = messages.AppendAssistant(parse)

	return parse, err
}

func CompleteLLMJSON[T promptMsg](ctx context.Context, llm agent.LLMAgent, msg T, options ...func(ztype.Map) ztype.Map) (ztype.Map, error) {
	resp, err := CompleteLLM(ctx, llm, msg, options...)
	if err != nil {
		return nil, err
	}

	return parseJSONResponse(resp)
}

func processLLMInteraction(ctx context.Context, llm agent.LLMAgent, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) (parse string, rawContext []byte, err error) {
	// 使用重构后的LLM交互处理器
	return ProcessLLMInteractionWithValidation(ctx, llm, messages, body, options...)
}

// WithMaxToolIterations 配置工具调用-续写的最大迭代次数（默认 3，负数按 0 处理）。
func WithMaxToolIterations(ctx context.Context, n int) context.Context {
	if n < 0 {
		n = 0
	}
	return context.WithValue(ctx, toolIterKey{}, n)
}

func getMaxToolIterations(ctx context.Context) int {
	if v, ok := ctx.Value(toolIterKey{}).(int); ok && v >= 0 {
		return v
	}
	return DefaultMaxToolIter
}
