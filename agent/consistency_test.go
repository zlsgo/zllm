package agent

import (
	"testing"

	runtime_errors "github.com/zlsgo/zllm/runtime/errors"
)

func TestProviderConsistency(t *testing.T) {
	tests := []struct {
		name     string
		provider LLM
	}{
		{
			name: "OpenAI Provider",
			provider: NewOpenAI(func(oa *OpenAIOptions) {
				oa.APIKey = "key1,key2,key3"
				oa.BaseURL = "https://api.openai.com/v1,https://api.backup.com/v1"
			}),
		},
		{
			name: "DeepSeek Provider",
			provider: NewDeepseek(func(oa *DeepseekOptions) {
				oa.APIKey = "key1,key2,key3"
				oa.BaseURL = "https://api.deepseek.com,https://api.backup.deepseek.com"
			}),
		},
		{
			name: "Anthropic Provider",
			provider: NewAnthropic(func(oa *AnthropicOptions) {
				oa.APIKey = "key1,key2,key3"
				oa.BaseURL = "https://api.anthropic.com,https://api.backup.anthropic.com"
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.provider == nil {
				t.Errorf("Provider creation failed")
				return
			}

			t.Logf("%s created successfully", tt.name)
		})
	}
}

func TestParseKeysFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single key",
			input:    "key1",
			expected: []string{"key1"},
		},
		{
			name:     "Multiple keys",
			input:    "key1,key2,key3",
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name:     "Keys with spaces",
			input:    "key1, key2 , key3",
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name:     "Empty keys filtered out",
			input:    "key1,,key3,",
			expected: []string{"key1", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeys(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseKeys(%q) length mismatch: got %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, key := range result {
				if key != tt.expected[i] {
					t.Errorf("parseKeys(%q)[%d] = %q, want %q", tt.input, i, key, tt.expected[i])
				}
			}
		})
	}
}

func TestNewRandFunction(t *testing.T) {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	selector := newRand(keys)

	for i := 0; i < 100; i++ {
		result := selector()
		if result == "" {
			t.Error("newRand returned empty string")
			break
		}

		found := false
		for _, key := range keys {
			if result == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("newRand returned unexpected key: %s", result)
		}
	}

	t.Log("newRand function test passed")
}

func TestErrorHandlingConsistency(t *testing.T) {
	testCases := []struct {
		status      int
		shouldRetry bool
		description string
	}{
		{200, false, "OK - should not retry"},
		{400, false, "Bad Request - should not retry"},
		{401, false, "Unauthorized - should not retry"},
		{429, true, "Rate Limited - should retry"},
		{500, true, "Internal Server Error - should retry"},
		{502, true, "Bad Gateway - should retry"},
		{503, true, "Service Unavailable - should retry"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			shouldRetry, err := isRetry(tc.status, "test message")
			if shouldRetry != tc.shouldRetry {
				t.Errorf("isRetry(%d, _) returned shouldRetry=%v, want %v", tc.status, shouldRetry, tc.shouldRetry)
			}

			if err != nil {
				if _, isLLMError := err.(runtime_errors.LLMError); !isLLMError {
					t.Errorf("isRetry(%d, _) returned non-LLMError: %T", tc.status, err)
				}
			}
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	configTests := []struct {
		name     string
		apiKeys  string
		baseURLs string
	}{
		{
			name:     "Multiple API keys and endpoints",
			apiKeys:  "key1,key2,key3",
			baseURLs: "https://api1.com/v1,https://api2.com/v1,https://api3.com/v1",
		},
		{
			name:     "Single API key and endpoint",
			apiKeys:  "single-key",
			baseURLs: "https://api.single.com/v1",
		},
		{
			name:     "Keys with spaces",
			apiKeys:  " key1 , key2 , key3 ",
			baseURLs: "https://api1.com/v1, https://api2.com/v1 ,https://api3.com/v1",
		},
	}

	for _, config := range configTests {
		t.Run(config.name+"-OpenAI", func(t *testing.T) {
			provider := NewOpenAI(func(oa *OpenAIOptions) {
				oa.APIKey = config.apiKeys
				oa.BaseURL = config.baseURLs
			})
			if provider == nil {
				t.Error("Failed to create OpenAI provider")
			}
		})

		t.Run(config.name+"-DeepSeek", func(t *testing.T) {
			provider := NewDeepseek(func(oa *DeepseekOptions) {
				oa.APIKey = config.apiKeys
				oa.BaseURL = config.baseURLs
			})
			if provider == nil {
				t.Error("Failed to create DeepSeek provider")
			}
		})

		t.Run(config.name+"-Anthropic", func(t *testing.T) {
			provider := NewAnthropic(func(oa *AnthropicOptions) {
				oa.APIKey = config.apiKeys
				oa.BaseURL = config.baseURLs
			})
			if provider == nil {
				t.Error("Failed to create Anthropic provider")
			}
		})
	}
}
