package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zerror"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/inlay"
	"github.com/zlsgo/zllm/message"
)

type OpenAIOptions struct {
	// Api Key，多个 Api Key 用逗号分隔
	APIKey string
	// 模型
	Model string
	// 基础 URL
	BaseURL string
	// Api URL
	APIURL string
	// 调试
	Debug bool
	// 温度
	Temperature float64
	// 流式
	Stream bool
	// 最大重试次数
	MaxRetries uint
	// 消息回调
	OnMessage func(string, []byte)
}

type OpenAIProvider struct {
	options  OpenAIOptions
	endpoint []string
	keys     []string
	headers  zhttp.Header
}

var _ LLMAgent = &OpenAIProvider{}

func NewOpenAIProvider(opt ...func(*OpenAIOptions)) LLMAgent {
	o := zutil.Optional(OpenAIOptions{
		APIKey:      zutil.Getenv("OPENAI_API_KEY", ""),
		Model:       zutil.Getenv("OPENAI_MODEL", "gpt-3.5-turbo"),
		Temperature: 0.5,
		MaxRetries:  3,
		Stream:      false,
		BaseURL:     zutil.Getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		APIURL:      zutil.Getenv("OPENAI_API_URL", "/chat/completions"),
	}, opt...)

	return &OpenAIProvider{
		options:  o,
		endpoint: zarray.Slice[string](o.BaseURL, ","),
		keys:     zarray.Slice[string](o.APIKey, ","),
		headers: zhttp.Header{
			"Content-Type": "application/json",
		},
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, body []byte) (*zjson.Res, error) {
	done, err := p.Stream(ctx, body, nil)
	if err != nil {
		return &zjson.Res{}, err
	}
	json := <-done
	return json, err
}

func (p *OpenAIProvider) Stream(ctx context.Context, body []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	body, err := CompleteMessag(p, body)
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

	inlay.Log(zstring.Bytes2String(body))

	keys := newRand(p.keys)
	endpoints := newRand(p.endpoint)

	done := make(chan *zjson.Res, 1)
	var json *zjson.Res
	err = doRetry("openai", int(p.options.MaxRetries), func() (retry bool, err error) {
		url, header := endpoints()+p.options.APIURL, zhttp.Header{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + keys(),
		}

		if stream {
			err = p.streamable(ctx, url, header, body, done, callback)
			if err != nil {
				status := zerror.UnwrapFirstCode(err)
				if status == 0 {
					return false, errors.New("ai provider api request failed")
				}

				if msg, ok := zerror.Unwrap(err, status); ok {
					canRetry, err := isRetry(int(status), msg.Error())
					if err != nil {
						return canRetry, err
					}
				}

				return true, errors.New("ai provider api request failed")
			}
			return false, nil
		}

		resp, err := inlay.GetClient().Post(url, header, body, ctx)
		if err != nil {
			return false, err
		}

		json = resp.JSONs()
		canRetry, err := isRetry(resp.StatusCode(), json.Get("error.message").String())
		if err != nil {
			return canRetry, err
		}

		done <- json
		inlay.Log(json)
		return false, nil
	})

	return done, err
}

func (p *OpenAIProvider) streamable(ctx context.Context, url string, header zhttp.Header, body []byte, done chan<- *zjson.Res, callback func(string, []byte)) error {
	sse, err := inlay.GetClient().SSE(url, nil, header, body, ctx)
	if err != nil {
		return err
	}

	go func() {
		var (
			rawMessage []byte
			result     = zstring.Buffer()
		)

		defer func() {
			if rawMessage != nil {
				choice := zjson.GetBytes(rawMessage, "choices.0")
				_ = choice.Delete("delta")
				_ = choice.Set("message.content", result.String())
				_ = choice.Set("message.role", "assistant")
				_ = choice.Set("message.finish_reason", "stop")
				json, _ := zjson.SetRawBytes(rawMessage, "choices.0", choice.Bytes())
				done <- zjson.ParseBytes(json)
			}
		}()

		c, err := sse.OnMessage(func(ev *zhttp.SSEEvent) {
			if bytes.Equal(ev.Data, []byte("[DONE]")) {
				sse.Close()
			}

			s := zjson.GetBytes(ev.Data, "choices.0.delta.content").String()

			if rawMessage == nil && s != "" {
				rawMessage = ev.Data
			}

			if callback != nil {
				callback(s, ev.Data)
			}

			if p.options.OnMessage != nil {
				p.options.OnMessage(s, ev.Data)
			}

			result.WriteString(s)
		})
		if err != nil {
			return
		}

		select {
		case <-c:
			return
		case <-sse.Error():
			return
		}
	}()

	return nil
}

func (p *OpenAIProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	requestBody := ztype.Map{
		"model":  p.options.Model,
		"stream": p.options.Stream,
	}
	if p.options.Temperature >= 0 {
		requestBody["temperature"] = p.options.Temperature
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

func (p *OpenAIProvider) ParseResponse(body *zjson.Res) (*Response, error) {
	data := body.Get("choices.0.message.content")

	if !data.Exists() {
		return nil, fmt.Errorf("error parsing response: %s", body.String())
	}

	content := data.Bytes()
	if len(content) == 0 {
		tool := body.Get("choices.0.message.tool_calls")
		if tool.Exists() {
			var tools []Tool
			for _, v := range tool.Array() {
				tools = append(tools, Tool{
					Name: v.Get("function.name").String(),
					Args: v.Get("function.arguments").String(),
				})
			}

			return &Response{
				Tools: tools,
			}, nil
		}
		return nil, errors.New("empty response from API")
	}

	return &Response{
		Content: content,
	}, nil
}
