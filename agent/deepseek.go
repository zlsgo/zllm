package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/utils"
)

type DeepseekOptions struct {
	APIKey      string
	Model       string
	BaseURL     string
	Debug       bool
	Temperature float64
	Stream      bool
	MaxRetries  uint
	OnMessage   func(string, []byte)
}

type DeepseekProvider struct {
	options  DeepseekOptions
	endpoint string
	headers  zhttp.Header
}

var _ LLMAgent = &DeepseekProvider{}

func NewDeepseekProvider(opt ...func(*DeepseekOptions)) LLMAgent {
	o := zutil.Optional(DeepseekOptions{
		APIKey:      zutil.Getenv("DEEPSEEK_API_KEY", ""),
		Model:       zutil.Getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
	}, opt...)

	return &DeepseekProvider{
		options:  o,
		endpoint: o.BaseURL + "/chat/completions",
		headers: zhttp.Header{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + o.APIKey,
		},
	}
}

func (p *DeepseekProvider) Generate(ctx context.Context, body []byte) (json *zjson.Res, err error) {
	body, err = CompleteMessag(p, body)
	if err != nil {
		return nil, err
	}

	stream := zjson.GetBytes(body, "stream").Bool()

	utils.Log(zstring.Bytes2String(body))

	err = doRetry("deepseek", int(p.options.MaxRetries), func() (retry bool, err error) {
		if stream {
			json, err = p.streamable(ctx, body)
			return true, err
		}

		resp, err := utils.GetClient().Post(p.endpoint, p.headers, body, ctx)
		if err != nil {
			return true, err
		}

		json = resp.JSONs()
		return false, nil
	})

	utils.Log(json)

	return
}

func (p *DeepseekProvider) streamable(ctx context.Context, body []byte) (*zjson.Res, error) {
	sse, err := zhttp.SSE(p.endpoint, p.headers, body, ctx)
	if err != nil {
		return nil, err
	}

	result := zstring.Buffer()

	var rawMessage []byte
	c, err := sse.OnMessage(func(ev *zhttp.SSEEvent) {
		if bytes.Equal(ev.Data, []byte("[DONE]")) {
			sse.Close()
		}

		s := zjson.GetBytes(ev.Data, "choices.0.delta.content").String()

		if rawMessage == nil && s != "" {
			rawMessage = ev.Data
		}

		if p.options.OnMessage != nil {
			p.options.OnMessage(s, ev.Data)
		}

		result.WriteString(s)
	})
	if err != nil {
		return nil, err
	}

	<-c

	choice := zjson.GetBytes(rawMessage, "choices.0")
	_ = choice.Delete("delta")
	_ = choice.Set("message.content", result.String())
	_ = choice.Set("message.role", "assistant")
	_ = choice.Set("message.finish_reason", "stop")
	json, err := zjson.SetRawBytes(rawMessage, "choices.0", choice.Bytes())
	if err != nil {
		return nil, err
	}

	return zjson.ParseBytes(json), nil
}

func (p *DeepseekProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	requestBody := ztype.Map{
		"model":       p.options.Model,
		"stream":      p.options.Stream,
		"temperature": p.options.Temperature,
	}

	requestBody["messages"] = zarray.Map(messages.History(true), func(i int, v []string) map[string]string {
		return map[string]string{
			"role":    v[0],
			"content": v[1],
		}
	})

	for _, v := range options {
		v(requestBody)
	}

	return json.Marshal(requestBody)
}

func (p *DeepseekProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	data := body.Get("choices.0.message.content")

	if !data.Exists() {
		return nil, fmt.Errorf("error parsing response: %s", body.String())
	}

	content := data.Bytes()
	if len(content) == 0 {
		return nil, errors.New("empty response from API")
	}

	return &Response{
		Content: content,
	}, nil
}
