package zllm

import (
	"context"

	"github.com/sohaha/zlsgo/zpool"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
)

// BalancerCompleteLLM 使用负载均衡器执行 LLM 请求
func BalancerCompleteLLM[T promptMsg](ctx context.Context, llms *zpool.Balancer[agent.LLM], msg T, options ...func(ztype.Map) ztype.Map) (resp string, err error) {
	runErr := llms.Run(func(node agent.LLM) (normal bool, err error) {
		resp, err = CompleteLLM(ctx, node, msg, options...)
		return err == nil, err
	})
	if runErr != nil {
		err = runErr
		return
	}
	return
}

// BalancerCompleteLLMJSON 使用负载均衡器执行 LLM 请求并返回 JSON 格式结果
func BalancerCompleteLLMJSON[T promptMsg](ctx context.Context, llms *zpool.Balancer[agent.LLM], msg T, options ...func(ztype.Map) ztype.Map) (ztype.Map, error) {
	resp, err := BalancerCompleteLLM(ctx, llms, msg, options...)
	if err != nil {
		return nil, err
	}

	return parseJSONResponse(resp)
}
