package agent

import (
	"context"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
)

type DeepseekOptions struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	Stream      bool
	MaxRetries  uint
	OnMessage   func(string, []byte)
}

func (o *DeepseekOptions) getAPIKey() []string {
	return parseValue(o.APIKey)
}

func (o *DeepseekOptions) getEndpoints() []string {
	return parseValue(o.BaseURL)
}

func (o *DeepseekOptions) getAPIPath() string {
	return "/chat/completions"
}

func (o *DeepseekOptions) buildHeaders(apiKey string) zhttp.Header {
	return buildAuthHeaders(apiKey)
}

func (o *DeepseekOptions) getStreamProcessor() string {
	return "openai"
}

func (o *DeepseekOptions) getMaxRetries() uint {
	return o.MaxRetries
}

func (o *DeepseekOptions) getOnMessage() func(string, []byte) {
	return o.OnMessage
}

type DeepseekProvider struct {
	*baseProvider
	options  DeepseekOptions
	endpoint []string
	keys     []string
}

var _ LLM = &DeepseekProvider{}

func NewDeepseek(opt ...func(*DeepseekOptions)) LLM {
	o := zutil.Optional(DeepseekOptions{
		APIKey:      zutil.Getenv("DEEPSEEK_API_KEY", ""),
		Model:       zutil.Getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
	}, opt...)

	Config := Config{
		Model:       o.Model,
		BaseURL:     o.BaseURL,
		APIKey:      o.APIKey,
		Temperature: o.Temperature,
		MaxRetries:  o.MaxRetries,
		Stream:      o.Stream,
	}

	return &DeepseekProvider{
		baseProvider: newBaseProvider(Config),
		options:      o,
		endpoint:     parseValue(o.BaseURL),
		keys:         parseValue(o.APIKey),
	}
}

func (p *DeepseekProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.generateWithConfig(ctx, &p.options, body)
}

func (p *DeepseekProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.streamWithConfig(ctx, &p.options, body, callback)
}

func (p *DeepseekProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	return p.PrepareMessagesRequest(messages, options...)
}

func (p *DeepseekProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	return p.baseProvider.parseDefaultResponse(body)
}
