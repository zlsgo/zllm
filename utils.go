package zllm

import (
	"context"
	"fmt"
	"time"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	runtime_errors "github.com/zlsgo/zllm/runtime/errors"
)

// ToolIterationState 工具迭代状态管理
type ToolIterationState struct {
	CurrentIteration int          // 当前迭代次数
	MaxIterations    int          // 最大迭代次数
	ToolResults      []ToolResult // 累积工具执行结果
	HasFinalResult   bool         // 是否获得最终结果
	FinalContent     string       // 最终内容
}

// llmInteractionProcessor LLM 交互处理器
type llmInteractionProcessor struct {
	ctx      context.Context             // 请求上下文
	llm      agent.LLM                   // LLM 代理
	messages *message.Messages           // 消息历史
	body     []byte                      // 请求体
	options  []func(ztype.Map) ztype.Map // 请求选项
}

// newLLMInteractionProcessor 创建 LLM 交互处理器
func newLLMInteractionProcessor(ctx context.Context, llm agent.LLM, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) *llmInteractionProcessor {
	return &llmInteractionProcessor{
		ctx:      ctx,
		llm:      llm,
		messages: messages,
		body:     body,
		options:  options,
	}
}

// Execute 运行 LLM 交互过程，处理工具调用和重试
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

// shouldContinueIteration 检查处理器是否应该继续下一次迭代
func (p *llmInteractionProcessor) shouldContinueIteration(state *ToolIterationState) bool {
	return state.CurrentIteration <= state.MaxIterations && !state.HasFinalResult
}

// processSingleIteration 处理单次 LLM 交互迭代，包含重试逻辑
func (p *llmInteractionProcessor) processSingleIteration(state *ToolIterationState) error {
	maxRetries := 2
	var consecutiveErrors int

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			interval := calculateRetryInterval(i, consecutiveErrors)
			time.Sleep(interval)
		}

		response, err := p.generateLLMResponse()
		if err != nil {
			consecutiveErrors++

			if !shouldRetryError(err, i, maxRetries, consecutiveErrors) {
				return err
			}
			continue
		}

		consecutiveErrors = 0

		if p.hasToolCalls(response) {
			if _, err := p.handleToolCalls(response, state); err != nil {
				return err
			}
			continue
		}

		if _, err := p.processFinalResponse(response, state); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("max retries (%d) reached", maxRetries)
}

// calculateRetryInterval 根据尝试次数和错误数量计算适当的重试间隔
// 参数 attempt 当前尝试次数
// 参数 consecutiveErrors 遇到的连续错误数量
// 返回 计算出的重试持续时间
func calculateRetryInterval(attempt, consecutiveErrors int) time.Duration {
	baseInterval := 100 * time.Millisecond

	if consecutiveErrors > 0 {
		if consecutiveErrors <= 3 {
			baseInterval = time.Duration(consecutiveErrors) * time.Second
			if baseInterval > 5*time.Second {
				baseInterval = 5 * time.Second
			}
		} else {
			baseInterval = time.Duration(attempt) * 200 * time.Millisecond
			if baseInterval > 2*time.Second {
				baseInterval = 2 * time.Second
			}
		}
	}

	return baseInterval
}

// shouldRetryError 根据错误类型和重试次数确定是否应该重试
func shouldRetryError(err error, currentAttempt, maxAttempts, consecutiveErrors int) bool {
	if llmErr, ok := err.(runtime_errors.LLMError); ok {
		if !llmErr.IsRetryable() {
			return false
		}

		switch llmErr.Code {
		case runtime_errors.ErrQuotaExceeded:
			return false
		case runtime_errors.ErrUnauthorized, runtime_errors.ErrInvalidRequest:
			return false
		}
	}

	if consecutiveErrors > 5 {
		return false
	}

	return currentAttempt < maxAttempts
}

// generateLLMResponse 从 LLM 生成响应
// 返回 解析后的 LLM 响应和任何错误
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

// hasToolCalls 检查 LLM 响应是否包含工具调用
// 参数 response 要检查的 LLM 响应
// 返回 如果存在工具调用返回 true，否则返回 false
func (p *llmInteractionProcessor) hasToolCalls(response *agent.Response) bool {
	return len(response.Tools) > 0
}

// handleToolCalls 处理来自 LLM 响应的工具调用
// 参数 response 包含工具调用的 LLM 响应
// 参数 state 要用工具结果更新的迭代状态
// 返回 如果应该继续处理返回 true，否则返回 false，以及任何错误
func (p *llmInteractionProcessor) handleToolCalls(response *agent.Response, state *ToolIterationState) (bool, error) {
	if !isAllowTools(p.ctx) {
		return false, fmt.Errorf("tools not supported: %s", ztype.ToString(response.Tools))
	}

	runner := getToolRunner(p.ctx)
	if runner == nil {
		return false, fmt.Errorf("tool runner not configured: use WithToolRunner() to set up tool execution for %d tool(s)", len(response.Tools))
	}

	toolResults, err := p.executeToolCalls(response.Tools, runner, state)
	if err != nil {
		return false, err
	}

	if err := p.updateMessagesForNextIteration(toolResults); err != nil {
		return false, err
	}

	return false, nil
}

// executeToolCalls 执行多个工具调用并收集它们的结果
// 参数 tools 要执行的工具列表
// 参数 runner 要使用的工具运行器实例
// 参数 state 要用结果更新的迭代状态
// 返回 工具结果和任何错误
func (p *llmInteractionProcessor) executeToolCalls(tools []agent.Tool, runner ToolRunner, state *ToolIterationState) ([]ToolResult, error) {
	if cap(state.ToolResults) < len(tools) {
		state.ToolResults = make([]ToolResult, 0, len(tools))
	} else {
		state.ToolResults = state.ToolResults[:0]
	}

	for _, tool := range tools {
		result := p.executeSingleTool(tool, runner)
		state.ToolResults = append(state.ToolResults, result)
	}

	return state.ToolResults, nil
}

// executeSingleTool 执行单个工具调用并返回其结果
// 参数 tool 要执行的工具
// 参数 runner 工具运行器实例
// 返回 工具执行结果
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

// updateMessagesForNextIteration 使用工具结果更新消息历史以进行下一次迭代
// 参数 toolResults 工具执行的结果
// 返回 如果更新消息失败则返回错误
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

// processFinalResponse 处理没有工具调用时的最终 LLM 响应
// 参数 response 要处理的 LLM 响应
// 参数 state 要用最终内容更新的迭代状态
// 返回 false 表示处理应该停止，以及遇到的任何错误
func (p *llmInteractionProcessor) processFinalResponse(response *agent.Response, state *ToolIterationState) (bool, error) {
	formatted, err := p.formatResponseContent(response.Content)
	if err != nil {
		return false, err
	}

	state.FinalContent = formatted
	state.HasFinalResult = true
	return false, nil
}

// formatResponseContent 根据消息格式要求格式化响应内容
// 参数 content 要格式化的原始内容
// 返回 格式化后的内容字符串和任何错误
func (p *llmInteractionProcessor) formatResponseContent(content []byte) (string, error) {
	formatted, err := p.messages.ParseFormat(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse response format: %w", err)
	}

	return zstring.Bytes2String(formatted), nil
}

// getCurrentBody 返回当前请求体
// 返回 当前请求体字节
func (p *llmInteractionProcessor) getCurrentBody() []byte {
	return p.body
}

// validateInteraction 验证处理器的配置和输入参数
// 返回 如果任何必需参数无效则返回错误
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

// processLLMInteractionWithValidation 处理带有参数验证的 LLM 交互
// 参数 ctx 请求上下文
// 参数 llm 要使用的 LLM 代理
// 参数 messages 消息历史
// 参数 body 请求体
// 参数 options 可选的请求修改器
// 返回 响应内容、原始体和任何错误
func processLLMInteractionWithValidation(ctx context.Context, llm agent.LLM, messages *message.Messages, body []byte, options ...func(ztype.Map) ztype.Map) (string, []byte, error) {
	processor := newLLMInteractionProcessor(ctx, llm, messages, body, options...)

	if err := processor.validateInteraction(); err != nil {
		return "", nil, fmt.Errorf("invalid interaction parameters: %w", err)
	}

	return processor.Execute()
}

// parseJSONResponse 将 JSON 响应字符串解析为映射
func parseJSONResponse(resp string) (ztype.Map, error) {
	if len(resp) > MinJSONLength && resp[0] == JSONLeftBrace && resp[len(resp)-1] == JSONRightBrace {
		return zjson.Parse(resp).Map(), nil
	}
	return ztype.ToMap(resp), nil
}
