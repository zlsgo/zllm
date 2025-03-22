package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/utils"
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

func (p *OpenAIProvider) Generate(ctx context.Context, body []byte) (json *zjson.Res, err error) {
	stream := zjson.GetBytes(body, "stream").Bool()

	utils.Log(zstring.Bytes2String(body))

	keys := newRand(p.keys)
	endpoints := newRand(p.endpoint)

	err = doRetry("openai", int(p.options.MaxRetries), func() (retry bool, err error) {
		url, header := endpoints()+p.options.APIURL, zhttp.Header{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + keys(),
		}
		if stream {
			json, err = p.streamable(ctx, url, header, body)
			return true, err
		}

		resp, err := utils.GetClient().Post(url, header, body, ctx)
		if err != nil {
			return false, err
		}

		status := resp.StatusCode()
		if status != http.StatusOK {
			errMsg := resp.JSON("error.message").String()
			if errMsg == "" {
				errMsg = fmt.Sprintf("status code: %d", status)
			}
			if zarray.Contains([]int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests}, status) {
				return true, errors.New(errMsg)
			}
			return false, errors.New(errMsg)
		}

		json = resp.JSONs()
		return false, nil
	})

	utils.Log(json)

	return
}

func (p *OpenAIProvider) streamable(ctx context.Context, url string, header zhttp.Header, body []byte) (*zjson.Res, error) {
	sse, err := zhttp.SSE(url, header, body, ctx)
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

	select {
	case <-c:
	case err := <-sse.Error():
		return nil, err
	}

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

func (p *OpenAIProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
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
