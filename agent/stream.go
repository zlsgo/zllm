package agent

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/zlsgo/zllm/runtime"
)

type streamProcessor interface {
	ProcessMessage(ev *zhttp.SSEEvent, config *streamConfig) (isDone bool, content string)
	BuildResponse(rawMessage []byte, result string) *zjson.Res
}

type streamConfig struct {
	OnMessage func(string, []byte)
}

func newStreamConfig(onMessage func(string, []byte)) *streamConfig {
	return &streamConfig{
		OnMessage: onMessage,
	}
}

func processOpenAIStream(ctx context.Context, sse *zhttp.SSEEngine, config *streamConfig, timeout time.Duration) (*zjson.Res, error) {
	processor := &openAIStreamProcessor{}
	return processStreamGeneric(ctx, sse, config, processor, timeout)
}

type openAIStreamProcessor struct{}

func (p *openAIStreamProcessor) ProcessMessage(ev *zhttp.SSEEvent, config *streamConfig) (bool, string) {
	if bytes.Equal(ev.Data, []byte("[DONE]")) {
		return true, ""
	}

	content := zjson.GetBytes(ev.Data, "choices.0.delta.content").String()
	return false, content
}

func (p *openAIStreamProcessor) BuildResponse(rawMessage []byte, result string) *zjson.Res {
	choice := zjson.GetBytes(rawMessage, "choices.0")
	_ = choice.Delete("delta")
	_ = choice.Set("message.content", result)
	_ = choice.Set("message.role", "assistant")
	_ = choice.Set("message.finish_reason", "stop")
	json, _ := zjson.SetRawBytes(rawMessage, "choices.0", choice.Bytes())
	return zjson.ParseBytes(json)
}

func processOllamaStream(ctx context.Context, sse *zhttp.SSEEngine, config *streamConfig, timeout time.Duration) (*zjson.Res, error) {
	processor := &ollamaStreamProcessor{}
	return processStreamGeneric(ctx, sse, config, processor, timeout)
}

// processAnthropicStream 处理Anthropic流
func processAnthropicStream(ctx context.Context, sse *zhttp.SSEEngine, config *streamConfig, timeout time.Duration) (*zjson.Res, error) {
	processor := &anthropicStreamProcessor{}
	return processStreamGeneric(ctx, sse, config, processor, timeout)
}

// processGeminiStream 处理Gemini流
func processGeminiStream(ctx context.Context, sse *zhttp.SSEEngine, config *streamConfig, timeout time.Duration) (*zjson.Res, error) {
	processor := &geminiStreamProcessor{}
	return processStreamGeneric(ctx, sse, config, processor, timeout)
}

func processStreamGeneric(ctx context.Context, sse *zhttp.SSEEngine, config *streamConfig, processor streamProcessor, timeout time.Duration) (*zjson.Res, error) {
	var (
		rawMessage []byte
		result     = zstring.Buffer()
	)

	defer func() {
		if r := recover(); r != nil {
			runtime.Log("Stream processing panic:", r)
		}
		if sse != nil {
			sse.Close()
		}
	}()

	done, processErr := sse.OnMessage(func(ev *zhttp.SSEEvent) {
		defer func() {
			if r := recover(); r != nil {
				runtime.Log("SSE event processing panic:", r)
			}
		}()

		isDone, content := processor.ProcessMessage(ev, config)

		if isDone {
			sse.Close()
			return
		}

		if content != "" {
			if rawMessage == nil {
				rawMessage = make([]byte, len(ev.Data))
				copy(rawMessage, ev.Data) // 避免数据竞争
			}

			if config.OnMessage != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							runtime.Log("Stream callback panic:", r)
						}
					}()
					config.OnMessage(content, ev.Data)
				}()
			}

			result.WriteString(content)
		}
	})

	if processErr != nil {
		return nil, fmt.Errorf("SSE setup error: %w", processErr)
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	select {
	case <-done:
		if rawMessage != nil {
			processedResult := processor.BuildResponse(rawMessage, result.String())
			if processedResult == nil {
				return nil, fmt.Errorf("failed to build stream response")
			}
			return processedResult, nil
		}
		return nil, fmt.Errorf("stream completed but no data received")
	case <-sse.Done():
		if rawMessage != nil {
			return processor.BuildResponse(rawMessage, result.String()), nil
		}
		return nil, fmt.Errorf("stream ended unexpectedly")
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			runtime.Log("Stream processing timeout")
			return nil, fmt.Errorf("stream processing timeout")
		}
		sse.Close()
		return nil, fmt.Errorf("stream cancelled: %w", timeoutCtx.Err())
	}
}

type ollamaStreamProcessor struct{}

func (p *ollamaStreamProcessor) ProcessMessage(ev *zhttp.SSEEvent, config *streamConfig) (bool, string) {
	values := bytes.Split(ev.Undefined, []byte("\n"))
	for i := range values {
		j := zjson.ParseBytes(values[i])

		if j.Get("done").Bool() {
			return true, ""
		}

		content := j.Get("message").Get("content").String()
		if content != "" {
			return false, content
		}
	}
	return false, ""
}

func (p *ollamaStreamProcessor) BuildResponse(rawMessage []byte, result string) *zjson.Res {
	choice := zjson.ParseBytes(rawMessage)
	_ = choice.Set("done", true)
	_ = choice.Set("message.content", result)
	_ = choice.Set("message.role", "assistant")
	_ = choice.Set("done_reason", "stop")
	return choice
}

// anthropicStreamProcessor Anthropic 流式处理器实现
// 事件类型：message_start, content_block_start, content_block_delta(text), content_block_stop, message_delta(stop_reason), message_stop
type anthropicStreamProcessor struct{}

func (p *anthropicStreamProcessor) ProcessMessage(ev *zhttp.SSEEvent, config *streamConfig) (bool, string) {
	t := zjson.GetBytes(ev.Data, "type").String()
	switch t {
	case "content_block_delta":
		content := zjson.GetBytes(ev.Data, "delta.text").String()
		if content != "" {
			return false, content
		}
		return false, ""
	case "message_delta":
		if zjson.GetBytes(ev.Data, "delta.stop_reason").Exists() {
			return true, ""
		}
		return false, ""
	case "message_stop":
		return true, ""
	default:
		return false, ""
	}
}

func (p *anthropicStreamProcessor) BuildResponse(rawMessage []byte, result string) *zjson.Res {
	// 使用 map 构建，后续 ParseBytes 转换
	m := map[string]any{
		"type":    "message",
		"role":    "assistant",
		"content": []map[string]any{{"type": "text", "text": result}},
	}
	b, _ := zjson.Marshal(m)
	return zjson.ParseBytes(b)
}

// geminiStreamProcessor Gemini 流式处理器实现
// Gemini 流式响应格式: {"candidates": [{"content": {"parts": [{"text": "..."}], "role": "model"}}]}
type geminiStreamProcessor struct{}

func (p *geminiStreamProcessor) ProcessMessage(ev *zhttp.SSEEvent, config *streamConfig) (bool, string) {
	// 检查是否为结束标记
	if len(ev.Data) == 0 {
		return false, ""
	}

	data := zjson.ParseBytes(ev.Data)

	// 提取文本内容
	content := data.Get("candidates.0.content.parts.0.text")
	if content.Exists() {
		text := content.String()
		if text != "" {
			return false, text
		}
	}

	// 检查完成标志
	if data.Get("candidates.0.finishReason").Exists() {
		return true, ""
	}

	return false, ""
}

func (p *geminiStreamProcessor) BuildResponse(rawMessage []byte, result string) *zjson.Res {
	// 构建 Gemini 响应格式
	m := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": result},
					},
					"role": "model",
				},
				"finishReason": "STOP",
				"index":        0,
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     0,
			"candidatesTokenCount": 0,
			"totalTokenCount":      0,
		},
	}

	b, _ := zjson.Marshal(m)
	return zjson.ParseBytes(b)
}
