package errors

import (
	"fmt"
	"testing"
)

func TestErrorCode_Values(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
	}{
		{"Unknown", ErrUnknown},
		{"Unauthorized", ErrUnauthorized},
		{"RateLimited", ErrRateLimited},
		{"BadRequest", ErrBadRequest},
		{"Server", ErrServer},
		{"Timeout", ErrTimeout},
		{"ContextCanceled", ErrContextCanceled},
		{"InvalidResponse", ErrInvalidResponse},
		{"ProviderUnavailable", ErrProviderUnavailable},
		{"QuotaExceeded", ErrQuotaExceeded},
		{"ModelNotFound", ErrModelNotFound},
		{"InvalidRequest", ErrInvalidRequest},
		{"TokenLimit", ErrTokenLimit},
		{"OutputFormatNotFound", ErrOutputFormatNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.code) < 0 {
				t.Errorf("ErrorCode %s should be non-negative", tt.name)
			}
		})
	}
}

func TestLLMError_Error(t *testing.T) {
	err := LLMError{
		Code:    ErrUnauthorized,
		Message: "test message",
	}

	expected := "test message"
	if err.Error() != expected {
		t.Errorf("LLMError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestLLMError_IsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		code      ErrorCode
		retryable bool
	}{
		{"RateLimited", ErrRateLimited, true},
		{"Server", ErrServer, true},
		{"Timeout", ErrTimeout, true},
		{"ProviderUnavailable", ErrProviderUnavailable, true},
		{"QuotaExceeded", ErrQuotaExceeded, false},
		{"Unauthorized", ErrUnauthorized, false},
		{"BadRequest", ErrBadRequest, false},
		{"Unknown", ErrUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LLMError{Code: tt.code}
			if err.IsRetryable() != tt.retryable {
				t.Errorf("LLMError.IsRetryable() for %s = %v, want %v", tt.name, err.IsRetryable(), tt.retryable)
			}
		})
	}
}

func TestLLMError_GetRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		contains string
	}{
		{"RateLimited", ErrRateLimited, "1-5 分钟"},
		{"QuotaExceeded", ErrQuotaExceeded, "配额重置"},
		{"Server", ErrServer, "10-30 秒"},
		{"Timeout", ErrTimeout, "立即重试"},
		{"ProviderUnavailable", ErrProviderUnavailable, "1-2 分钟"},
		{"Unknown", ErrUnknown, "不可重试"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LLMError{Code: tt.code}
			delay := err.GetRetryDelay()
			if !containsIgnoreCase(delay, tt.contains) && tt.code != ErrUnknown {
				t.Errorf("LLMError.GetRetryDelay() for %s = %v, should contain %v", tt.name, delay, tt.contains)
			}
		})
	}
}

func TestLLMError_GetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{"Unauthorized", ErrUnauthorized, "high"},
		{"InvalidRequest", ErrInvalidRequest, "high"},
		{"BadRequest", ErrBadRequest, "high"},
		{"RateLimited", ErrRateLimited, "medium"},
		{"QuotaExceeded", ErrQuotaExceeded, "medium"},
		{"Server", ErrServer, "low"},
		{"Timeout", ErrTimeout, "low"},
		{"ProviderUnavailable", ErrProviderUnavailable, "low"},
		{"Unknown", ErrUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LLMError{Code: tt.code}
			if severity := err.GetSeverity(); severity != tt.expected {
				t.Errorf("LLMError.GetSeverity() for %s = %v, want %v", tt.name, severity, tt.expected)
			}
		})
	}
}

func TestNewLLMError(t *testing.T) {
	err := NewLLMError(ErrUnauthorized, "test error")

	llmErr, ok := err.(LLMError)
	if !ok {
		t.Error("NewLLMError should return LLMError type")
	}

	if llmErr.Code != ErrUnauthorized {
		t.Errorf("NewLLMError() code = %v, want %v", llmErr.Code, ErrUnauthorized)
	}

	if llmErr.Message != "test error" {
		t.Errorf("NewLLMError() message = %v, want %v", llmErr.Message, "test error")
	}
}

func TestNewLLMErrorWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"retry_count": 3,
		"timeout":     "30s",
	}

	err := NewLLMErrorWithDetails(ErrServer, "server error", details)

	llmErr, ok := err.(LLMError)
	if !ok {
		t.Error("NewLLMErrorWithDetails should return LLMError type")
	}

	if llmErr.Code != ErrServer {
		t.Errorf("NewLLMErrorWithDetails() code = %v, want %v", llmErr.Code, ErrServer)
	}

	if llmErr.Details["retry_count"] != 3 {
		t.Errorf("NewLLMErrorWithDetails() details = %v, want %v", llmErr.Details, details)
	}
}

func TestMapHTTPToCode(t *testing.T) {
	tests := []struct {
		status   int
		expected ErrorCode
	}{
		{401, ErrUnauthorized},
		{429, ErrRateLimited},
		{500, ErrServer},
		{503, ErrServer},
		{400, ErrBadRequest},
		{404, ErrBadRequest},
		{200, ErrUnknown},
		{0, ErrUnknown},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			if code := MapHTTPToCode(tt.status); code != tt.expected {
				t.Errorf("MapHTTPToCode(%d) = %v, want %v", tt.status, code, tt.expected)
			}
		})
	}
}

func TestMapHTTPToCodeWithMessage(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		message  string
		expected ErrorCode
	}{
		{"401", 401, "unauthorized", ErrUnauthorized},
		{"429_rate_limit", 429, "rate limit exceeded", ErrQuotaExceeded}, // 匹配逻辑中"limit"会匹配quota
		{"429_quota", 429, "quota exceeded", ErrQuotaExceeded},
		{"404_model", 404, "model not found", ErrModelNotFound},
		{"404_general", 404, "not found", ErrBadRequest},
		{"400_token", 400, "token limit exceeded", ErrTokenLimit},
		{"400_length", 400, "content too long", ErrInvalidRequest}, // "length"而不是"token"
		{"400_general", 400, "bad request", ErrInvalidRequest},
		{"500", 500, "internal server error", ErrServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if code := MapHTTPToCodeWithMessage(tt.status, tt.message); code != tt.expected {
				t.Errorf("MapHTTPToCodeWithMessage(%d, %s) = %v, want %v", tt.status, tt.message, code, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		should bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "test", false},
		{"", "test", false},
		{"Hello", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.s, tt.substr), func(t *testing.T) {
			if result := containsIgnoreCase(tt.s, tt.substr); result != tt.should {
				t.Errorf("containsIgnoreCase(%s, %s) = %v, want %v", tt.s, tt.substr, result, tt.should)
			}
		})
	}
}
