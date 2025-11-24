package agent

import (
	"context"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
)

type OllamaOptions struct {
	Model       string
	BaseURL     string
	APIKey      string // 支持 Ollama 远程部署的 token 认证
	Temperature float64
	Stream      bool
	MaxRetries  uint
	OnMessage   func(string, []byte)
}

func (o *OllamaOptions) getAPIKey() string {
	return o.APIKey
}

func (o *OllamaOptions) getEndpoints() []string {
	return []string{o.BaseURL}
}

func (o *OllamaOptions) getAPIPath() string {
	return "/api/chat"
}

func (o *OllamaOptions) buildHeaders(apiKey string) zhttp.Header {
	if apiKey != "" {
		return buildAuthHeaders("Bearer " + apiKey)
	}
	return buildJSONHeaders()
}

func (o *OllamaOptions) getStreamProcessor() string {
	return "ollama"
}

func (o *OllamaOptions) getMaxRetries() uint {
	return o.MaxRetries
}

func (o *OllamaOptions) getOnMessage() func(string, []byte) {
	return o.OnMessage
}

type OllamaProvider struct {
	*baseProvider
	options  OllamaOptions
	endpoint string
}

var _ LLM = &OllamaProvider{}

func NewOllama(opt ...func(*OllamaOptions)) LLM {
	o := zutil.Optional(OllamaOptions{
		Model:       zutil.Getenv("OLLAMA_MODEL", "qwen2.5:3b"),
		Temperature: 0.48,
		MaxRetries:  3,
		BaseURL:     zutil.Getenv("OLLAMA_BASE_URL", "http://localhost:11434"),
		APIKey:      zutil.Getenv("OLLAMA_API_KEY", ""),
	}, opt...)

	Config := Config{
		Model:       o.Model,
		BaseURL:     o.BaseURL,
		APIKey:      o.APIKey,
		Temperature: o.Temperature,
		MaxRetries:  o.MaxRetries,
		Stream:      o.Stream,
	}

	return &OllamaProvider{
		baseProvider: newBaseProvider(Config),
		options:      o,
		endpoint:     o.BaseURL + "/api/chat",
	}
}

func (p *OllamaProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.generateWithConfig(ctx, &p.options, body)
}

func (p *OllamaProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	var err error
	body, err = completeMessage(p, body)
	if err != nil {
		return nil, err
	}
	return p.baseProvider.streamWithConfig(ctx, &p.options, body, callback)
}

func (p *OllamaProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	return p.PrepareMessagesRequest(messages, options...)
}

func (p *OllamaProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	return p.baseProvider.parseDefaultResponse(body)
}
