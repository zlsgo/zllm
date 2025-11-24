package agent

import (
	"context"
	"time"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

type OpenAIOptions struct {
	APIKey      string
	Model       string
	BaseURL     string
	APIURL      string
	Temperature float64
	Stream      bool
	MaxRetries  uint
	OnMessage   func(string, []byte)
}

// 实现 providerConfig 接口
func (o *OpenAIOptions) getAPIKey() string {
	return o.APIKey
}

func (o *OpenAIOptions) getEndpoints() []string {
	return parseKeys(o.BaseURL)
}

func (o *OpenAIOptions) getAPIPath() string {
	return o.APIURL
}

func (o *OpenAIOptions) buildHeaders(apiKey string) zhttp.Header {
	return buildAuthHeaders("Bearer " + apiKey)
}

func (o *OpenAIOptions) getStreamProcessor() string {
	return "openai"
}

func (o *OpenAIOptions) getMaxRetries() uint {
	return o.MaxRetries
}

func (o *OpenAIOptions) getOnMessage() func(string, []byte) {
	return o.OnMessage
}

type OpenAIProvider struct {
	*baseProvider
	options  OpenAIOptions
	endpoint []string
	keys     []string
}

func (p *OpenAIProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	return p.PrepareMessagesRequest(messages, options...)
}

var _ LLM = &OpenAIProvider{}

// 创建新的 OpenAI LLM 代理
//
//		o.APIKey = "sk-...your-api-key..."
//		o.Model = "gpt-4"
//		o.Temperature = 0.7
//		o.MaxRetries = 3
//	})
func NewOpenAI(opt ...func(*OpenAIOptions)) LLM {
	o := zutil.Optional(OpenAIOptions{
		APIKey:      zutil.Getenv("OPENAI_API_KEY", ""),
		Model:       zutil.Getenv("OPENAI_MODEL", "gpt-4.1"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		APIURL:      zutil.Getenv("OPENAI_API_URL", "/chat/completions"),
	}, opt...)

	if o.APIKey == "" {
		runtime.Log("Warning: OPENAI_API_KEY not set, provider will be non-functional")
	}

	// 使用新的配置系统
	config := DefaultConfig().
		WithAPIKey(o.APIKey).
		WithModel(o.Model).
		WithTemperature(o.Temperature).
		WithRetries(o.MaxRetries).
		WithTimeout(30*time.Second, 60*time.Second) // 默认超时时间

	baseProvider := newBaseProvider(config)

	return &OpenAIProvider{
		baseProvider: baseProvider,
		options:      o,
		endpoint:     parseKeys(o.BaseURL),
		keys:         parseKeys(o.APIKey),
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.generateWithConfig(ctx, &p.options, body)
}

func (p *OpenAIProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.streamWithConfig(ctx, &p.options, body, callback)
}

func (p *OpenAIProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	return p.baseProvider.parseDefaultResponse(body)
}
