package zllm

import (
	"context"
	"fmt"

	"github.com/sohaha/zlsgo/zjson"
)

// MapToolHandler 工具处理函数类型
type MapToolHandler func(ctx context.Context, args *zjson.Res) (string, error)

// MapToolRunner 工具运行器
type MapToolRunner struct {
	handlers map[string]MapToolHandler
}

// NewMapToolRunner 创建工具运行器
func NewMapToolRunner(handlers map[string]MapToolHandler) *MapToolRunner {
	if handlers == nil {
		handlers = map[string]MapToolHandler{}
	}
	return &MapToolRunner{handlers: handlers}
}

// Run 执行工具
func (r *MapToolRunner) Run(ctx context.Context, name, args string) (string, error) {
	h, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	j := zjson.Parse(args)
	return h(ctx, j)
}
