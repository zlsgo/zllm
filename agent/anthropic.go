package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

// Anthropic 特定配置选项
type AnthropicOptions struct {
	APIKey      string               // Anthropic API 密钥，支持逗号分隔的多密钥负载均衡
	Model       string               // 模型名称 (如 "claude-3-sonnet-20240229", "claude-3-opus-20240229")
	BaseURL     string               // API 基础 URL（可选）
	APIURL      string               // 完整 API 端点 URL（可选，覆盖 BaseURL）
	Version     string               // API 版本（默认 "2023-06-01"）
	Temperature float64              // 采样温度（0.0-1.0，控制随机性）
	Stream      bool                 // 启用流式响应
	MaxRetries  uint                 // 失败请求的最大重试次数
	MaxTokens   int                  // 响应中的最大 token 数（默认 4096）
	OnMessage   func(string, []byte) // 流式消息回调函数
}

// Anthropic Claude 模型的 LLM 代理实现
type AnthropicProvider struct {
	*baseProvider
	options  AnthropicOptions
	endpoint []string // 负载均衡的 API 端点 URL
	keys     []string // 负载均衡的 API 密钥
}

var _ LLM = &AnthropicProvider{}

//		o.MaxTokens = 4096
//		o.MaxRetries = 3
//	})
func NewAnthropic(opt ...func(*AnthropicOptions)) LLM {
	o := zutil.Optional(AnthropicOptions{
		APIKey:      zutil.Getenv("ANTHROPIC_API_KEY", ""),
		Model:       zutil.Getenv("ANTHROPIC_MODEL", "claude-3-5-sonnet-latest"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		APIURL:      zutil.Getenv("ANTHROPIC_API_URL", "/v1/messages"),
		Version:     zutil.Getenv("ANTHROPIC_VERSION", "2023-06-01"),
		MaxTokens:   1024,
	}, opt...)

	base := newBaseProvider(Config{
		Model:       o.Model,
		BaseURL:     o.BaseURL,
		APIKey:      o.APIKey,
		Temperature: o.Temperature,
		MaxRetries:  o.MaxRetries,
		Stream:      o.Stream,
	})

	return &AnthropicProvider{
		baseProvider: base,
		options:      o,
		endpoint:     parseKeys(o.BaseURL),
		keys:         parseKeys(o.APIKey),
	}
}

func (p *AnthropicProvider) getEndpoint() string {
	if len(p.endpoint) == 0 {
		return p.options.BaseURL + p.options.APIURL
	}
	return p.endpoint[0] + p.options.APIURL
}

func (p *AnthropicProvider) getAPIKey() string {
	if len(p.keys) == 0 {
		return p.options.APIKey
	}
	return p.keys[0]
}

// Generate 普通请求
func (p *AnthropicProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	if err := requireAPIKey(p.getAPIKey(), "anthropic"); err != nil {
		return nil, err
	}
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}

	// 如果调用方传入 stream=true，但未提供回调，则关闭流
	stream := zjson.GetBytes(body, "stream").Bool()
	if stream {
		body, _ = zjson.SetBytes(body, "stream", false)
	}

	logRequestBody(body)

	keys := newRand(p.keys)
	endpoints := newRand(p.endpoint)

	headers := zhttp.Header{
		"Content-Type":      "application/json",
		"x-api-key":         keys(),
		"anthropic-version": p.options.Version,
	}

	json, status, err := p.baseProvider.DoRequest(ctx, endpoints()+p.options.APIURL, headers, body)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, handleHTTPError("anthropic", status, "")
	}

	runtime.Log(json)
	return json, nil
}

// Stream 流式请求
func (p *AnthropicProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	if err := requireAPIKey(p.getAPIKey(), "anthropic"); err != nil {
		return nil, err
	}

	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}

	stream := zjson.GetBytes(body, "stream").Bool()
	if callback == nil && stream {
		stream = false
		body, _ = zjson.SetBytes(body, "stream", false)
	} else if callback != nil && !stream {
		stream = true
		body, _ = zjson.SetBytes(body, "stream", true)
	}

	logRequestBody(body)

	done := make(chan *zjson.Res, 1)

	keys := newRand(p.keys)
	endpoints := newRand(p.endpoint)

	headers := zhttp.Header{
		"Content-Type":      "application/json",
		"x-api-key":         keys(),
		"anthropic-version": p.options.Version,
	}

	if !stream {
		// 非流模式，直接请求
		go func() {
			defer close(done)
			json, status, err := p.baseProvider.DoRequest(ctx, endpoints()+p.options.APIURL, headers, body)
			if err != nil {
				runtime.Log("Anthropic request error:", err)
				return
			}
			if status >= 400 {
				runtime.Log("Anthropic http error:", status)
				return
			}
			done <- json
		}()
		return done, nil
	}

	// 流模式
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				runtime.Log("Anthropic stream goroutine panic:", r)
			}
		}()

		sse, err := p.baseProvider.DoSSE(ctx, endpoints()+p.options.APIURL, headers, body)
		if err != nil {
			runtime.Log("Anthropic SSE error:", err)
			return
		}

		json, err := processAnthropicStream(ctx, sse, newStreamConfig(func(chunk string, data []byte) {
			if p.options.OnMessage != nil {
				p.options.OnMessage(chunk, data)
			}
			if callback != nil {
				callback(chunk, data)
			}
		}), p.baseProvider.config.StreamTimeout)
		if err != nil || json == nil {
			if err != nil {
				runtime.Log("Anthropic stream processing error:", err)
			}
			return
		}

		// 返回带超时保护
		select {
		case done <- json:
		case <-ctx.Done():
			runtime.Log("Anthropic stream context cancelled")
		case <-time.After(30 * time.Second):
			runtime.Log("Anthropic stream response timeout after 30 seconds")
		}
	}()

	return done, nil
}

// PrepareRequest 将消息转换为 Anthropic 消息格式
func (p *AnthropicProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	req := ztype.Map{
		"model":       p.GetConfig().Model,
		"stream":      p.GetConfig().Stream,
		"temperature": p.GetConfig().Temperature,
	}

	// 收集 system 消息并转换对话
	history := messages.History(true)
	var sys []string
	arr := make([]ztype.Map, 0, len(history))
	for i := range history {
		role := history[i][0]
		content := history[i][1]
		switch role {
		case message.RoleSystem:
			sys = append(sys, content)
		default:
			// Anthropic 仅支持 user/assistant 角色
			if role != message.RoleUser && role != message.RoleAssistant {
				// 其他角色当作 user
				role = message.RoleUser
			}
			arr = append(arr, ztype.Map{
				"role": role,
				"content": []ztype.Map{
					{"type": "text", "text": content},
				},
			})
		}
	}

	if len(sys) > 0 {
		req["system"] = strings.Join(sys, "\n\n")
	}
	req["messages"] = arr

	// 默认 max_tokens
	if p.options.MaxTokens > 0 {
		req["max_tokens"] = p.options.MaxTokens
	} else {
		req["max_tokens"] = 1024
	}

	for _, v := range options {
		req = v(req)
	}

	if _, ok := req["max_tokens"]; !ok {
		req["max_tokens"] = 1024
	}

	return json.Marshal(req)
}

// ParseResponse 解析 Anthropic 返回
func (p *AnthropicProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	text := body.Get("content.0.text").String()
	if text == "" {
		// 兜底：有些情况 content 为空
		if body.Get("error").Exists() {
			return nil, errors.New(body.Get("error.message").String())
		}
		return &Response{Content: []byte{}}, nil
	}
	return &Response{Content: []byte(text)}, nil
}
