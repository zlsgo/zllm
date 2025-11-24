package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

// Gemini 特定配置选项
type GeminiOptions struct {
	APIKey      string               // Gemini API 密钥，支持逗号分隔的多密钥负载均衡
	Model       string               // 模型名称 (如 "gemini-pro", "gemini-pro-vision")
	BaseURL     string               // API 基础 URL（可选）
	APIURL      string               // 完整 API 端点 URL（可选，覆盖 BaseURL）
	Temperature float64              // 采样温度（0.0-2.0，控制随机性）
	Stream      bool                 // 启用流式响应
	MaxRetries  uint                 // 失败请求的最大重试次数
	MaxTokens   int                  // 响应中的最大 token 数（可选）
	TopP        float64              // 核采样参数（0.0-1.0，控制多样性）
	TopK        int                  // 核采样参数，选择前 K 个候选词
	OnMessage   func(string, []byte) // 流式消息回调函数
}

// 实现 providerConfig 接口
func (o *GeminiOptions) getAPIKey() string {
	return o.APIKey
}

func (o *GeminiOptions) getEndpoints() []string {
	return parseKeys(o.BaseURL)
}

func (o *GeminiOptions) getAPIPath() string {
	return o.APIURL
}

func (o *GeminiOptions) buildHeaders(apiKey string) zhttp.Header {
	return zhttp.Header{
		"Content-Type":   "application/json",
		"x-goog-api-key": apiKey,
	}
}

func (o *GeminiOptions) getStreamProcessor() string {
	return "gemini"
}

func (o *GeminiOptions) getMaxRetries() uint {
	return o.MaxRetries
}

func (o *GeminiOptions) getOnMessage() func(string, []byte) {
	return o.OnMessage
}

// GeminiProvider Google Gemini 模型的 LLM 代理实现
type GeminiProvider struct {
	*baseProvider
	options  GeminiOptions
	endpoint []string // 负载均衡的 API 端点 URL
	keys     []string // 负载均衡的 API 密钥
}

var _ LLM = &GeminiProvider{}

// NewGemini 创建新的 Gemini LLM 代理
//
//	agent.NewGemini(func(o *agent.GeminiOptions) {
//		o.APIKey = "your-api-key"
//		o.Model = "gemini-pro"
//		o.Temperature = 0.7
//		o.MaxRetries = 3
//	})
func NewGemini(opt ...func(*GeminiOptions)) LLM {
	o := zutil.Optional(GeminiOptions{
		APIKey:      zutil.Getenv("GEMINI_API_KEY", ""),
		Model:       zutil.Getenv("GEMINI_MODEL", "gemini-2.0-flash"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
		APIURL:      zutil.Getenv("GEMINI_API_URL", "/v1beta/models/gemini-2.0-flash:generateContent"),
		TopP:        0.95,
		TopK:        32,
	}, opt...)

	if o.APIKey == "" {
		runtime.Log("Warning: GEMINI_API_KEY not set, provider will be non-functional")
	}

	if o.Temperature < 0 || o.Temperature > 2 {
		runtime.Log("Warning: Temperature should be between 0 and 2, clamping to valid range")
		if o.Temperature < 0 {
			o.Temperature = 0
		} else {
			o.Temperature = 2
		}
	}

	if o.APIURL == "" {
		modelPart := strings.ReplaceAll(o.Model, ":", "/")
		o.APIURL = "/v1beta/models/" + modelPart + ":generateContent"
	}

	baseProvider := newBaseProvider(Config{
		Model:       o.Model,
		BaseURL:     o.BaseURL,
		APIKey:      o.APIKey,
		Temperature: o.Temperature,
		MaxRetries:  o.MaxRetries,
		Stream:      o.Stream,
	})

	return &GeminiProvider{
		baseProvider: baseProvider,
		options:      o,
		endpoint:     parseKeys(o.BaseURL),
		keys:         parseKeys(o.APIKey),
	}
}

func (p *GeminiProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.generateWithConfig(ctx, &p.options, body)
}

func (p *GeminiProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.streamWithConfig(ctx, &p.options, body, callback)
}

// PrepareRequest 将消息转换为 Gemini API 格式
func (p *GeminiProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	generationConfig := ztype.Map{
		"temperature": p.GetConfig().Temperature,
	}

	if p.options.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = p.options.MaxTokens
	}
	if p.options.TopP > 0 {
		generationConfig["topP"] = p.options.TopP
	}
	if p.options.TopK > 0 {
		generationConfig["topK"] = p.options.TopK
	}

	history := messages.History(true)
	contents := make([]ztype.Map, 0, len(history))

	for i := range history {
		role := history[i][0]
		content := history[i][1]
		var geminiRole string
		switch role {
		case message.RoleUser:
			geminiRole = "user"
		case message.RoleAssistant:
			geminiRole = "model"
		case message.RoleSystem:
			if len(contents) == 0 {
				contents = append(contents, ztype.Map{
					"role":  "user",
					"parts": []ztype.Map{{"text": "System: " + content}},
				})
				continue
			}
			if len(contents) > 0 && contents[0]["role"] == "user" {
				if parts, ok := contents[0]["parts"].([]ztype.Map); ok {
					contents[0]["parts"] = append(parts, ztype.Map{"text": "\n\nSystem: " + content})
				}
			}
			continue
		default:
			geminiRole = "user"
		}

		contents = append(contents, ztype.Map{
			"role":  geminiRole,
			"parts": []ztype.Map{{"text": content}},
		})
	}

	request := ztype.Map{
		"contents":         contents,
		"generationConfig": generationConfig,
		"safetySettings": []ztype.Map{
			{
				"category":  "HARM_CATEGORY_HARASSMENT",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_HATE_SPEECH",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_DANGEROUS_CONTENT",
				"threshold": "BLOCK_NONE",
			},
		},
	}

	for _, v := range options {
		request = v(request)
	}

	return json.Marshal(request)
}

// ParseResponse 解析 Gemini 返回
func (p *GeminiProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	if body == nil {
		return nil, errors.New("empty response")
	}

	if body.Get("error").Exists() {
		msg := body.Get("error.message").String()
		if msg == "" {
			msg = "unknown error"
		}
		return nil, errors.New(msg)
	}

	candidates := body.Get("candidates")
	if !candidates.Exists() || len(candidates.Array()) == 0 {
		return nil, errors.New("no candidates in response")
	}

	content := candidates.Get("0.content.parts.0.text")
	if !content.Exists() {
		return &Response{Content: []byte{}}, nil
	}

	return &Response{Content: []byte(content.String())}, nil
}
