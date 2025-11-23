package zllm

import (
	"context"
	"fmt"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
)

// ToolIterationState 工具迭代状态
type ToolIterationState struct {
	CurrentIteration int
	MaxIterations    int
	ToolResults      []ToolResult
	HasFinalResult   bool
	FinalContent     string
}

// llmInteractionProcessor LLM交互处理器（私有）
type llmInteractionProcessor struct {
	ctx      context.Context
	llm      agent.LLMAgent
	messages *message.Messages
	body     []byte
	options  []func(ztype.Map) ztype.Map
}

// newLLMInteractionProcessor 创建LLM交互处理器（私有）
func newLLMInteractionProcessor(ctx context.Context, llm agent.LLMAgent, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) *llmInteractionProcessor {
	return &llmInteractionProcessor{
		ctx:      ctx,
		llm:      llm,
		messages: messages,
		body:     body,
		options:  options,
	}
}

// Execute 执行LLM交互（包括工具调用）
func (p *llmInteractionProcessor) Execute() (string, []byte, error) {
	state := &ToolIterationState{
		CurrentIteration: 0,
		MaxIterations:    getMaxToolIterations(p.ctx),
	}

	for p.shouldContinueIteration(state) {
		if err := p.processSingleIteration(state); err != nil {
			return "", nil, err
		}

		state.CurrentIteration++
	}

	if state.HasFinalResult {
		return state.FinalContent, p.getCurrentBody(), nil
	}

	return "", p.getCurrentBody(), fmt.Errorf("max tool iterations (%d) reached without final result", state.MaxIterations)
}

// shouldContinueIteration 判断是否继续下一次迭代
func (p *llmInteractionProcessor) shouldContinueIteration(state *ToolIterationState) bool {
	return state.CurrentIteration <= state.MaxIterations && !state.HasFinalResult
}

// processSingleIteration 处理单次LLM交互迭代
func (p *llmInteractionProcessor) processSingleIteration(state *ToolIterationState) error {
	err := agent.DoRetryFunc("processLLM", 2, func() (bool, error) {
		response, err := p.generateLLMResponse()
		if err != nil {
			return true, err
		}

		if p.hasToolCalls(response) {
			return p.handleToolCalls(response, state)
		}

		return p.processFinalResponse(response, state)
	})

	return err
}

// generateLLMResponse 生成LLM响应
func (p *llmInteractionProcessor) generateLLMResponse() (*agent.Response, error) {
	resp, err := p.llm.Generate(p.ctx, p.body)
	if err != nil {
		return nil, err
	}

	response, err := p.llm.ParseResponse(resp)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// hasToolCalls 检查响应是否包含工具调用
func (p *llmInteractionProcessor) hasToolCalls(response *agent.Response) bool {
	return len(response.Tools) > 0
}

// handleToolCalls 处理工具调用
func (p *llmInteractionProcessor) handleToolCalls(response *agent.Response, state *ToolIterationState) (bool, error) {
	if !isAllowTools(p.ctx) {
		return false, fmt.Errorf("tools not supported: %s", ztype.ToString(response.Tools))
	}

	runner := getToolRunner(p.ctx)
	if runner == nil {
		return false, fmt.Errorf("tool runner not configured: use WithToolRunner() to set up tool execution for %d tool(s)", len(response.Tools))
	}

	// 执行工具调用
	toolResults, err := p.executeToolCalls(response.Tools, runner, state)
	if err != nil {
		return false, err
	}

	// 更新消息和请求体
	if err := p.updateMessagesForNextIteration(toolResults); err != nil {
		return false, err
	}

	// 继续下一次迭代
	return false, nil
}

// executeToolCalls 执行工具调用
func (p *llmInteractionProcessor) executeToolCalls(tools []agent.Tool, runner ToolRunner, state *ToolIterationState) ([]ToolResult, error) {
	// 预分配或重用slice
	if cap(state.ToolResults) < len(tools) {
		state.ToolResults = make([]ToolResult, 0, len(tools))
	} else {
		state.ToolResults = state.ToolResults[:0]
	}

	// 执行每个工具调用
	for _, tool := range tools {
		result := p.executeSingleTool(tool, runner)
		state.ToolResults = append(state.ToolResults, result)
	}

	return state.ToolResults, nil
}

// executeSingleTool 执行单个工具
func (p *llmInteractionProcessor) executeSingleTool(tool agent.Tool, runner ToolRunner) ToolResult {
	result := ToolResult{
		Name: tool.Name,
		Args: tool.Args,
	}

	out, ferr := runner.Run(p.ctx, tool.Name, tool.Args)
	result.Result = out
	if ferr != nil {
		result.Err = ferr.Error()
	}

	return result
}

// updateMessagesForNextIteration 更新消息为下一次迭代
func (p *llmInteractionProcessor) updateMessagesForNextIteration(toolResults []ToolResult) error {
	content := getToolResultFormatter(p.ctx)(toolResults)
	p.messages.AppendUser(content)

	var err error
	p.body, err = p.llm.PrepareRequest(p.messages, p.options...)
	if err != nil {
		return fmt.Errorf("failed to prepare request after tool execution: %w", err)
	}

	return nil
}

// processFinalResponse 处理最终响应
func (p *llmInteractionProcessor) processFinalResponse(response *agent.Response, state *ToolIterationState) (bool, error) {
	formatted, err := p.formatResponseContent(response.Content)
	if err != nil {
		return false, err
	}

	state.FinalContent = formatted
	state.HasFinalResult = true
	return false, nil
}

// formatResponseContent 格式化响应内容
func (p *llmInteractionProcessor) formatResponseContent(content []byte) (string, error) {
	formatted, err := p.messages.ParseFormat(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse response format: %w", err)
	}

	return zstring.Bytes2String(formatted), nil
}

// getCurrentBody 获取当前请求体
func (p *llmInteractionProcessor) getCurrentBody() []byte {
	return p.body
}

// validateInteraction 验证交互参数
func (p *llmInteractionProcessor) validateInteraction() error {
	if p.ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if p.llm == nil {
		return fmt.Errorf("LLM agent cannot be nil")
	}
	if p.messages == nil {
		return fmt.Errorf("messages cannot be nil")
	}
	if len(p.body) == 0 {
		return fmt.Errorf("request body cannot be empty")
	}

	return nil
}

// ProcessLLMInteractionWithValidation 带验证的LLM交互处理
func ProcessLLMInteractionWithValidation(ctx context.Context, llm agent.LLMAgent, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) (string, []byte, error) {
	processor := newLLMInteractionProcessor(ctx, llm, messages, body, options...)

	if err := processor.validateInteraction(); err != nil {
		return "", nil, fmt.Errorf("invalid interaction parameters: %w", err)
	}

	return processor.Execute()
}

// parseJSONResponse 统一的 JSON 响应解析函数
func parseJSONResponse(resp string) (ztype.Map, error) {
	if len(resp) > MinJSONLength && resp[0] == JSONLeftBrace && resp[len(resp)-1] == JSONRightBrace {
		return zjson.Parse(resp).Map(), nil
	}
	return ztype.ToMap(resp), nil
}
