package zllm

import (
	"context"
	"errors"
	"fmt"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/utils"
)

func CompleteLLM[T *message.Prompt | *message.Messages](ctx context.Context, llm agent.LLMAgent, p T, options ...ztype.Map) (string, error) {
	var (
		messages *message.Messages
		err      error
	)

	switch v := any(p).(type) {
	case *message.Prompt:
		messages, err = v.ConvertToMessages()
		if err != nil {
			return "", err
		}
	case *message.Messages:
		messages = v
	default:
		return "", fmt.Errorf("invalid prompt type: %T", p)
	}

	content, err := llm.PrepareRequest(messages, options...)
	if err != nil {
		return "", err
	}

	parse, rawContext, err := processLLMInteraction(ctx, llm, messages, content)
	if err != nil {
		return zstring.Bytes2String(rawContext), err
	}

	err = messages.AppendAssistant(parse)

	return parse, err
}

func CompleteLLMJSON[T *message.Prompt | *message.Messages](ctx context.Context, llm agent.LLMAgent, p T, options ...ztype.Map) (ztype.Map, error) {
	resp, err := CompleteLLM(ctx, llm, p, options...)
	if err != nil {
		return nil, err
	}

	if len(resp) > 2 && resp[0] == '{' && resp[len(resp)-1] == '}' {
		return zjson.Parse(resp).Map(), nil
	}

	return ztype.ToMap(resp), nil
}

func processLLMInteraction(ctx context.Context, llm agent.LLMAgent, messages *message.Messages, body []byte) (parse string, rawContext []byte, err error) {
	i := -1
	_ = zutil.DoRetry(2, func() error {
		i++
		if i > 0 {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			utils.Warn("retrying:", i, errMsg)
		}

		var resp *zjson.Res
		resp, err = llm.Generate(ctx, body)
		if err != nil {
			return nil
		}

		var response *agent.Response
		response, err = llm.ParseResponse(resp)
		if err != nil {
			return err
		}

		if len(response.Tools) > 0 {
			err = errors.New("tools not supported: " + ztype.ToString(response.Tools))
			return nil
		}

		var b []byte
		b, err = messages.ParseFormat(response.Content)
		if err != nil {
			return err
		}

		parse = zstring.Bytes2String(b)
		return nil
	}, func(rc *zutil.RetryConf) {
		rc.Interval = 0
	})
	return
}
