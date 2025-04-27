package zllm

import (
	"context"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zpool"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
)

func BalancerCompleteLLM[T promptMsg](ctx context.Context, llms *zpool.Balancer[agent.LLMAgent], msg T, options ...func(ztype.Map) ztype.Map) (resp string, err error) {
	runErr := llms.Run(func(node agent.LLMAgent) (normal bool, err error) {
		resp, err = CompleteLLM(ctx, node, msg, options...)
		return err == nil, err
	})
	if runErr != nil {
		err = runErr
		return
	}
	return
}

func BalancerCompleteLLMJSON[T promptMsg](ctx context.Context, llms *zpool.Balancer[agent.LLMAgent], msg T, options ...func(ztype.Map) ztype.Map) (ztype.Map, error) {
	resp, err := BalancerCompleteLLM(ctx, llms, msg, options...)
	if err != nil {
		return nil, err
	}

	if len(resp) > 2 && resp[0] == '{' && resp[len(resp)-1] == '}' {
		return zjson.Parse(resp).Map(), nil
	}

	return ztype.ToMap(resp), nil
}
