// Package agent 统一LLM提供商接口
package agent

import (
	"context"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/message"
)

// LLM LLM代理接口
type LLM interface {
	Generate(ctx context.Context, data []byte) (resp *zjson.Res, err error)
	Stream(ctx context.Context, data []byte, callback func(string, []byte)) (done <-chan *zjson.Res, err error)
	PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) (body []byte, err error)
	ParseResponse(*zjson.Res) (*Response, error)
}

// Response LLM响应格式
type Response struct {
	Content []byte `json:"content"`
	Tools   []Tool `json:"tools"`
}
