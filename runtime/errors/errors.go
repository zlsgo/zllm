// Package errors LLM错误处理
package errors

import (
	"strings"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	ErrUnknown ErrorCode = iota
	ErrUnauthorized
	ErrRateLimited
	ErrBadRequest
	ErrServer
	ErrTimeout
	ErrContextCanceled
	ErrInvalidResponse
	ErrProviderUnavailable
	ErrQuotaExceeded
	ErrModelNotFound
	ErrInvalidRequest
	ErrTokenLimit
	ErrOutputFormatNotFound
)

// LLMError LLM错误结构
type LLMError struct {
	Code    ErrorCode
	Message string
	Details map[string]interface{}
}

func (e LLMError) Error() string {
	return e.Message
}

// IsRetryable 判断是否可重试
func (e LLMError) IsRetryable() bool {
	switch e.Code {
	case ErrRateLimited, ErrServer, ErrTimeout, ErrProviderUnavailable:
		return true
	case ErrQuotaExceeded:
		return false
	default:
		return false
	}
}

// GetRetryDelay 获取重试延迟建议
func (e LLMError) GetRetryDelay() string {
	switch e.Code {
	case ErrRateLimited:
		return "建议等待 1-5 分钟后重试"
	case ErrQuotaExceeded:
		return "建议等待配额重置后重试，或升级配额"
	case ErrServer:
		return "建议等待 10-30 秒后重试"
	case ErrTimeout:
		return "建议立即重试"
	case ErrProviderUnavailable:
		return "建议等待 1-2 分钟后重试"
	default:
		return "错误不可重试"
	}
}

// GetSeverity 获取错误严重性
func (e LLMError) GetSeverity() string {
	switch e.Code {
	case ErrUnauthorized, ErrInvalidRequest, ErrBadRequest:
		return "high"
	case ErrRateLimited, ErrQuotaExceeded:
		return "medium"
	case ErrServer, ErrTimeout, ErrProviderUnavailable:
		return "low"
	default:
		return "unknown"
	}
}

// NewLLMError 创建LLM错误
func NewLLMError(code ErrorCode, msg string) error {
	return LLMError{Code: code, Message: msg}
}

// NewLLMErrorWithDetails 创建带详情的LLM错误
func NewLLMErrorWithDetails(code ErrorCode, msg string, details map[string]interface{}) error {
	return LLMError{Code: code, Message: msg, Details: details}
}

// MapHTTPToCode HTTP状态码转错误码
func MapHTTPToCode(status int) ErrorCode {
	switch {
	case status == 401:
		return ErrUnauthorized
	case status == 429:
		return ErrRateLimited
	case status >= 500:
		return ErrServer
	case status >= 400:
		return ErrBadRequest
	default:
		return ErrUnknown
	}
}

// MapHTTPToCodeWithMessage HTTP状态码和消息转错误码
func MapHTTPToCodeWithMessage(status int, errMsg string) ErrorCode {
	switch {
	case status == 401:
		return ErrUnauthorized
	case status == 429:
		if containsIgnoreCase(errMsg, "quota") || containsIgnoreCase(errMsg, "limit") {
			return ErrQuotaExceeded
		}
		return ErrRateLimited
	case status == 404:
		if containsIgnoreCase(errMsg, "model") {
			return ErrModelNotFound
		}
		return ErrBadRequest
	case status == 400:
		if containsIgnoreCase(errMsg, "token") || containsIgnoreCase(errMsg, "length") {
			return ErrTokenLimit
		}
		return ErrInvalidRequest
	case status >= 500:
		return ErrServer
	case status >= 400:
		return ErrBadRequest
	default:
		return ErrUnknown
	}
}

// containsIgnoreCase 忽略大小写的字符串包含检查
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
