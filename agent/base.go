package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

// AuthProvider 认证配置接口
type AuthProvider interface {
	getAPIKey() []string
	buildHeaders(apiKey string) zhttp.Header
}

// EndpointProvider 端点配置接口
type EndpointProvider interface {
	getEndpoints() []string
	getAPIPath() string
}

// StreamConfigProvider 流配置接口
type StreamConfigProvider interface {
	getStreamProcessor() string
	getOnMessage() func(string, []byte)
}

// RetryConfigProvider 重试配置接口
type RetryConfigProvider interface {
	getMaxRetries() uint
}

// providerConfig Provider配置接口 - 组合接口
type providerConfig interface {
	AuthProvider
	EndpointProvider
	StreamConfigProvider
	RetryConfigProvider
}

// Config 提供商配置
type Config struct {
	// 基础配置
	Model       string
	BaseURL     string
	APIKey      string
	Temperature float64

	// 高级配置
	MaxRetries     uint
	Stream         bool
	RequestTimeout time.Duration
	StreamTimeout  time.Duration

	// 调试配置
	DebugMode bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Temperature:    0.5,
		MaxRetries:     3,
		Stream:         false,
		RequestTimeout: 30 * time.Second,
		StreamTimeout:  60 * time.Second,
		DebugMode:      false,
	}
}

// WithAPIKey 设置API密钥
func (c Config) WithAPIKey(key string) Config {
	c.APIKey = key
	return c
}

// WithModel 设置模型
func (c Config) WithModel(model string) Config {
	c.Model = model
	return c
}

// WithTemperature 设置温度参数
func (c Config) WithTemperature(temp float64) Config {
	c.Temperature = temp
	return c
}

// WithTimeout 设置超时时间
func (c Config) WithTimeout(request, stream time.Duration) Config {
	c.RequestTimeout = request
	c.StreamTimeout = stream
	return c
}

// WithRetries 设置重试次数
func (c Config) WithRetries(retries uint) Config {
	c.MaxRetries = retries
	return c
}

// WithDebug 设置调试模式
func (c Config) WithDebug(debug bool) Config {
	c.DebugMode = debug
	return c
}

type baseProvider struct {
	config Config
}

func newBaseProvider(config Config) *baseProvider {
	if config.Temperature < 0 || config.Temperature > 2 {
		runtime.Log("Warning: Temperature should be between 0 and 2, clamping to valid range")
		if config.Temperature < 0 {
			config.Temperature = 0
		} else {
			config.Temperature = 2
		}
	}
	return &baseProvider{
		config: config,
	}
}

func (bp *baseProvider) GetConfig() Config {
	return bp.config
}

func (bp *baseProvider) PrepareMessagesRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	requestBody := ztype.Map{
		"model":  bp.config.Model,
		"stream": bp.config.Stream,
	}

	requestBody["temperature"] = bp.config.Temperature

	requestBody["messages"] = zarray.Map(messages.History(true), func(i int, v []string) map[string]string {
		return map[string]string{
			"role":    v[0],
			"content": v[1],
		}
	})

	for _, v := range options {
		requestBody = v(requestBody)
	}

	return json.Marshal(requestBody)
}

func (bp *baseProvider) DoRequest(ctx context.Context, url string, headers zhttp.Header, body []byte) (*zjson.Res, int, error) {
	resp, err := runtime.GetClient().Post(url, headers, body, ctx)
	if err != nil {
		return nil, 0, err
	}

	return resp.JSONs(), resp.StatusCode(), nil
}

func (bp *baseProvider) DoSSE(ctx context.Context, url string, headers zhttp.Header, body []byte) (*zhttp.SSEEngine, error) {
	return runtime.GetClient().SSE(url, nil, headers, body, ctx)
}

// generateWithConfig 通用生成方法
func (bp *baseProvider) generateWithConfig(ctx context.Context, config providerConfig, body []byte) (*zjson.Res, error) {
	// 强制关闭流式模式进行 Generate
	stream := zjson.GetBytes(body, "stream").Bool()
	if stream {
		body, _ = zjson.SetBytes(body, "stream", false)
	}

	logRequestBody(body)

	keys := newRand(config.getAPIKey())
	endpoints := newRand(config.getEndpoints())

	url := endpoints() + config.getAPIPath()
	headers := config.buildHeaders(keys())

	json, status, err := bp.DoRequest(ctx, url, headers, body)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, handleHTTPError("provider", status, "")
	}

	runtime.Log(json)
	return json, nil
}

// streamWithConfig 通用流处理方法
func (bp *baseProvider) streamWithConfig(ctx context.Context, config providerConfig, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
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

	if !stream {
		go func() {
			defer close(done)
			defer func() {
				if r := recover(); r != nil {
					runtime.Log("Generate goroutine panic:", r)
				}
			}()

			json, err := bp.generateWithConfig(ctx, config, body)
			if err != nil {
				runtime.Log("Generate error:", err)
				return
			}

			select {
			case done <- json:
			case <-ctx.Done():
				runtime.Log("Generate context cancelled")
				return
			case <-time.After(bp.config.RequestTimeout):
				runtime.Log("Generate response timeout after", bp.config.RequestTimeout)
				return
			}
		}()
		return done, nil
	}

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				runtime.Log("Stream goroutine panic:", r)
			}
		}()

		keys := newRand(config.getAPIKey())
		endpoints := newRand(config.getEndpoints())

		url := endpoints() + config.getAPIPath()
		headers := config.buildHeaders(keys())

		sse, err := bp.DoSSE(ctx, url, headers, body)
		if err != nil {
			runtime.Log("SSE error:", err)
			return
		}

		streamConfig := newStreamConfig(func(chunk string, data []byte) {
			if config.getOnMessage() != nil {
				config.getOnMessage()(chunk, data)
			}
			if callback != nil {
				callback(chunk, data)
			}
		})

		var json *zjson.Res
		switch config.getStreamProcessor() {
		case "openai":
			json, err = processOpenAIStream(ctx, sse, streamConfig, bp.config.StreamTimeout)
		case "anthropic":
			json, err = processAnthropicStream(ctx, sse, streamConfig, bp.config.StreamTimeout)
		case "ollama":
			json, err = processOllamaStream(ctx, sse, streamConfig, bp.config.StreamTimeout)
		case "gemini":
			json, err = processGeminiStream(ctx, sse, streamConfig, bp.config.StreamTimeout)
		default:
			runtime.Log("Unknown stream processor:", config.getStreamProcessor())
			return
		}

		if err != nil || json == nil {
			if err != nil {
				runtime.Log("Stream processing error:", err)
			}
			return
		}

		runtime.Log(json)

		select {
		case done <- json:
		case <-ctx.Done():
			runtime.Log("Stream context cancelled")
			return
		case <-time.After(bp.config.StreamTimeout):
			runtime.Log("Stream response timeout after", bp.config.StreamTimeout)
			return
		}
	}()

	return done, nil
}

// parseDefaultResponse 通用响应解析
func (bp *baseProvider) parseDefaultResponse(body *zjson.Res) (*Response, error) {
	tools, _, hasTools := preferToolCallsInResponse(body)
	if hasTools {
		// 转换私有 tool 为公开 Tool
		publicTools := make([]Tool, len(tools))
		for i, t := range tools {
			publicTools[i] = Tool{
				Name: t.Name,
				Args: t.Args,
			}
		}
		return &Response{Tools: publicTools}, nil
	}
	content, err := extractContentOrError(body)
	if err != nil {
		return nil, err
	}
	return &Response{Content: content}, nil
}
