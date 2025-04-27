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
	"github.com/zlsgo/zllm/inlay"
	"github.com/zlsgo/zllm/message"
)

type OllamaOptions struct {
	Model       string
	BaseURL     string
	Debug       bool
	Temperature float64
	Stream      bool
	MaxRetries  uint
	OnMessage   func(string, []byte)
}

type OllamaProvider struct {
	options  OllamaOptions
	endpoint string
	headers  zhttp.Header
}

var _ LLMAgent = &OllamaProvider{}

func NewOllamaProvider(opt ...func(*OllamaOptions)) LLMAgent {
	o := zutil.Optional(OllamaOptions{
		Model:       zutil.Getenv("OLLAMA_MODEL", "qwen2.5:3b"),
		Temperature: 0.48,
		MaxRetries:  3,
		BaseURL:     zutil.Getenv("OLLAMA_BASE_URL", "http://localhost:11434"),
	}, opt...)

	return &OllamaProvider{
		options:  o,
		endpoint: o.BaseURL + "/api/chat",
		// endpoint: o.BaseURL + "/api/generate",
		headers: zhttp.Header{
			"Content-Type": "application/json",
		},
	}
}

func (p *OllamaProvider) Generate(ctx context.Context, body []byte) (json *zjson.Res, err error) {
	body, err = CompleteMessag(p, body)
	if err != nil {
		return nil, err
	}

	inlay.Log(zstring.Bytes2String(body))
	stream := zjson.GetBytes(body, "stream").Bool()

	err = doRetry("ollama", int(p.options.MaxRetries), func() (retry bool, err error) {
		if stream {
			json, err = p.streamable(ctx, body)
			return true, err
		}

		var resp *zhttp.Res
		resp, err = inlay.GetClient().Post(p.endpoint, p.headers, body, ctx)
		if err != nil {
			return false, err
		}

		if resp.StatusCode() != 200 {
			err = fmt.Errorf("ollama status code: %d", resp.StatusCode())
			return false, err
		}

		json = resp.JSONs()
		return false, nil
	})

	inlay.Log(json)

	return
}

func (p *OllamaProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (done <-chan *zjson.Res, err error) {
	return nil, errors.New("ollama not support stream")
}

func (p *OllamaProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
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
		requestBody = v(requestBody)
	}

	return json.Marshal(requestBody)
}

func (p *OllamaProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	data := body.Get("message.content")

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

func (p *OllamaProvider) streamable(ctx context.Context, body []byte) (*zjson.Res, error) {
	sse, err := inlay.GetClient().SSE(p.endpoint, nil, p.headers, body, ctx)
	if err != nil {
		return nil, err
	}

	result := zstring.Buffer()

	var rawMessage []byte
	c, err := sse.OnMessage(func(ev *zhttp.SSEEvent) {
		values := bytes.Split(ev.Undefined, []byte("\n"))
		for i := range values {
			j := zjson.ParseBytes(values[i])

			if j.Get("done").Bool() {
				sse.Close()
			}

			s := j.Get("message").Get("content").String()

			if rawMessage == nil && s != "" {
				rawMessage = values[i]
			}

			if p.options.OnMessage != nil {
				p.options.OnMessage(s, values[i])
			}

			result.WriteString(s)
		}
	})
	if err != nil {
		return nil, err
	}

	<-c

	choice := zjson.ParseBytes(rawMessage)
	_ = choice.Set("done", true)
	_ = choice.Set("message.content", result.String())
	_ = choice.Set("message.role", "assistant")
	_ = choice.Set("done_reason", "stop")

	return choice, nil
}
