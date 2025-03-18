package agent

import (
	"context"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/message"
)

type LLMAgent interface {
	Generate(ctx context.Context, data []byte) (*zjson.Res, error)
	PrepareRequest(messages *message.Messages, options ...ztype.Map) (body []byte, err error)
	ParseResponse(*zjson.Res) (*Response, error)
}

type Tool struct {
	Name string `json:"name"`
	Args string `json:"args"`
}

type Response struct {
	Content []byte `json:"content"`
	Tools   []Tool `json:"tools"`
}
