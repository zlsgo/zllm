package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
	runtime_errors "github.com/zlsgo/zllm/runtime/errors"
)

// Tool 工具调用信息
type Tool struct {
	Name string `json:"name"`
	Args string `json:"args"`
}

type tool struct {
	Name string `json:"name"`
	Args string `json:"args"`
}

func parseKeys(apiKey string) []string {
	if apiKey == "" {
		return []string{}
	}

	rawKeys := strings.Split(apiKey, ",")
	result := make([]string, 0, len(rawKeys))
	for _, key := range rawKeys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func newRand(keys []string) func() string {
	if len(keys) == 0 {
		return func() string { return "" }
	}

	shuffled := make([]string, len(keys))
	copy(shuffled, keys)
	shuffled = zarray.Shuffle(shuffled)

	var index int32
	mu := sync.RWMutex{}
	return func() string {
		mu.RLock()
		defer mu.RUnlock()

		if len(shuffled) == 0 {
			return ""
		}
		currentIndex := int(atomic.AddInt32(&index, 1) - 1)
		return shuffled[currentIndex%len(shuffled)]
	}
}

func isRetry(status int, msg string) (bool, error) {
	if status == http.StatusOK {
		return false, nil
	}
	errMsg := msg
	if errMsg == "" {
		errMsg = fmt.Sprintf("status code: %d", status)
	}
	errorCode := runtime_errors.MapHTTPToCodeWithMessage(status, errMsg)
	llmErr := runtime_errors.NewLLMError(errorCode, errMsg)
	if llmErr.(runtime_errors.LLMError).IsRetryable() {
		return true, llmErr
	}
	return false, llmErr
}

// shouldRetry 判断是否可重试
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if llmErr, ok := err.(runtime_errors.LLMError); ok {
		return llmErr.IsRetryable()
	}
	return true
}

// requireAPIKey 验证API密钥
func requireAPIKey(apiKey, providerName string) error {
	if apiKey == "" {
		return runtime_errors.NewLLMError(runtime_errors.ErrUnauthorized, fmt.Sprintf("%s api key is required", providerName))
	}
	return nil
}

// buildJSONHeaders 创建JSON请求头
func buildJSONHeaders() zhttp.Header {
	return zhttp.Header{
		"Content-Type": "application/json",
	}
}

// buildAuthHeaders 创建认证请求头
func buildAuthHeaders(authToken string) zhttp.Header {
	return zhttp.Header{
		"Content-Type":  "application/json",
		"Authorization": authToken,
	}
}

// handleHTTPError 处理HTTP错误
func handleHTTPError(providerName string, status int, message string) error {
	if message == "" {
		message = fmt.Sprintf("%s status %d", providerName, status)
	}
	return runtime_errors.NewLLMError(runtime_errors.MapHTTPToCode(status), message)
}

// logRequestBody 记录请求体
func logRequestBody(body []byte) {
	if runtime.IsDebug() {
		sanitized := sanitizeSensitiveData(zstring.Bytes2String(body))
		runtime.Log("Request Body:", sanitized)
	}
}

// 用于检测日志中敏感数据的正则表达式模式
var (
	apiKeyPattern   = regexp.MustCompile(`(?i)(api[_-]?key["\s]*[:=]["\s]*)[a-zA-Z0-9_-]{10,}`)
	bearerPattern   = regexp.MustCompile(`(?i)(authorization["\s]*[:=]["\s]*bearer\s+)[a-zA-Z0-9._-]+`)
	tokenPattern    = regexp.MustCompile(`(?i)(token["\s]*[:=]["\s]*)[a-zA-Z0-9._-]{10,}`)
	passwordPattern = regexp.MustCompile(`(?i)(password["\s]*[:=]["\s]*)[^\s"']{4,}`)
)

// sanitizeSensitiveData 清理敏感数据
func sanitizeSensitiveData(input string) string {
	result := apiKeyPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := apiKeyPattern.FindStringSubmatch(match)
		if len(parts) >= 2 {
			key := parts[1]
			return key + "***REDACTED***"
		}
		return match
	})

	result = bearerPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := bearerPattern.FindStringSubmatch(match)
		if len(parts) >= 2 {
			prefix := parts[1]
			return prefix + "***REDACTED***"
		}
		return match
	})

	result = tokenPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := tokenPattern.FindStringSubmatch(match)
		if len(parts) >= 2 {
			key := parts[1]
			return key + "***REDACTED***"
		}
		return match
	})

	result = passwordPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := passwordPattern.FindStringSubmatch(match)
		if len(parts) >= 2 {
			key := parts[1]
			return key + "***REDACTED***"
		}
		return match
	})

	return result
}

// WithToolCallHint 添加工具调用选项
func WithToolCallHint(tools any) func(ztype.Map) ztype.Map {
	return func(m ztype.Map) ztype.Map {
		if tools != nil {
			m["tools"] = tools
		}
		if _, ok := m["tool_choice"]; !ok {
			m["tool_choice"] = "auto"
		}
		return m
	}
}

// completeMessage 完成消息转换
func completeMessage(agent LLM, body []byte) ([]byte, error) {
	var err error
	if !zjson.ValidBytes(body) {
		msg := message.NewMessages()
		msg.AppendUser(zstring.Bytes2String(body))
		body, err = agent.PrepareRequest(msg)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// doRequestWithRetry 执行带有重试逻辑的 HTTP 请求
// 参数 ctx 请求上下文
// 参数 providerName 提供商名称，用于错误上下文
// 参数 maxRetries 最大重试尝试次数
// 参数 requestFn 执行实际 HTTP 请求的函数
// 返回 响应体和任何错误
func doRequestWithRetry(ctx context.Context, providerName string, maxRetries uint, requestFn func() (int, []byte, error)) ([]byte, error) {
	var result []byte
	err := zutil.DoRetry(int(maxRetries), func() error {
		status, body, err := requestFn()
		if err != nil {
			return err
		}

		if status == http.StatusOK {
			result = body
			return nil
		}

		shouldRetry, retryErr := isRetry(status, string(body))
		if !shouldRetry {
			return retryErr
		}
		return retryErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// doStreamWithRetry 执行带有重试逻辑的流式 HTTP 请求
// 参数 ctx 请求上下文
// 参数 providerName 提供商名称，用于错误上下文
// 参数 maxRetries 最大重试尝试次数
// 参数 requestFn 创建 SSE 连接的函数
// 参数 onSSE 处理 SSE 流的函数
// 返回 累积的响应和任何错误
func doStreamWithRetry(ctx context.Context, providerName string, maxRetries uint, requestFn func() (*zhttp.SSEEngine, error), onSSE func(*zhttp.SSEEngine) (*zjson.Res, error)) ([]byte, error) {
	var result []byte
	err := zutil.DoRetry(int(maxRetries), func() error {
		sse, err := requestFn()
		if err != nil {
			return err
		}

		res, err := onSSE(sse)
		if err != nil {
			sse.Close()
			if !shouldRetry(err) {
				return err
			}
			return err
		}

		if res != nil {
			result = res.Bytes()
			return nil
		}
		return fmt.Errorf("empty response")
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// preferToolCallsInResponse 从 LLM 响应中提取工具调用（如果存在）
// 参数 body 解析后的 JSON 响应
// 返回 工具列表、内容字节和是否找到工具
func preferToolCallsInResponse(body *zjson.Res) ([]tool, []byte, bool) {
	toolCalls := body.Get("choices.0.message.tool_calls")
	if toolCalls.Exists() && toolCalls.IsArray() && len(toolCalls.Array()) > 0 {
		var tools []tool
		for _, v := range toolCalls.Array() {
			tools = append(tools, tool{
				Name: v.Get("function.name").String(),
				Args: v.Get("function.arguments").String(),
			})
		}
		content := body.Get("choices.0.message.content").Bytes()
		return tools, content, true
	}
	return nil, nil, false
}

// extractContentOrError 从 LLM 响应中提取内容或返回错误
// 参数 body 解析后的 JSON 响应
// 返回 内容字节和遇到的任何错误
func extractContentOrError(body *zjson.Res) ([]byte, error) {
	data := body.Get("choices.0.message.content")
	if !data.Exists() {
		return nil, fmt.Errorf("error parsing response: %s", body.String())
	}
	s := data.String()
	if strings.TrimSpace(s) == "" {
		return nil, errors.New("empty response from API")
	}
	return []byte(s), nil
}
