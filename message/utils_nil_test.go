package message

import (
	"testing"
)

func TestNilOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		wantErr  bool
	}{
		{
			name:     "正常响应",
			response: []byte("这是一个普通的响应"),
			wantErr:  false,
		},
		{
			name:     "JSON响应",
			response: []byte(`{"key": "value"}`),
			wantErr:  false,
		},
		{
			name:     "空响应",
			response: []byte(""),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := NilOutputFormat()

			// 测试Parse方法
			result, err := format.Parse(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("NilOutputFormat.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// nil格式化器应该返回nil
			if result != nil {
				t.Errorf("NilOutputFormat.Parse() = %v, want nil", result)
			}

			// 测试Format方法
			formatted, err := format.Format(string(tt.response))
			if err != nil {
				t.Errorf("NilOutputFormat.Format() error = %v", err)
				return
			}

			// nil格式化器应该返回原始内容
			if formatted != string(tt.response) {
				t.Errorf("NilOutputFormat.Format() = %v, want %v", formatted, string(tt.response))
			}

			// 测试String方法
			if format.String() != "nil" {
				t.Errorf("NilOutputFormat.String() = %v, want nil", format.String())
			}
		})
	}
}

func TestMessagesParseFormatWithNil(t *testing.T) {
	tests := []struct {
		name           string
		messages       *Messages
		response       []byte
		expectedResult []byte
		wantErr        bool
	}{
		{
			name:           "无格式化器",
			messages:       NewMessages(),
			response:       []byte("原始响应"),
			expectedResult: []byte("原始响应"),
			wantErr:        false,
		},
		{
			name: "使用nil格式化器",
			messages: func() *Messages {
				m := NewMessages()
				m.AppendUser("测试消息", NilOutputFormat())
				return m
			}(),
			response:       []byte("不需要格式化的响应"),
			expectedResult: []byte("不需要格式化的响应"),
			wantErr:        false,
		},
		{
			name: "使用WithNilFormat",
			messages: func() *Messages {
				m := NewMessages()
				m.AppendUser("测试消息", WithNilFormat())
				return m
			}(),
			response:       []byte("另一个不需要格式化的响应"),
			expectedResult: []byte("另一个不需要格式化的响应"),
			wantErr:        false,
		},
		{
			name: "JSON格式化器（对比测试）",
			messages: func() *Messages {
				m := NewMessages()
				m.AppendUser("测试消息", DefaultOutputFormat())
				return m
			}(),
			response:       []byte(`{"Assistant": "测试内容"}`),
			expectedResult: []byte(`{"Assistant":"测试内容"}`),
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.messages.ParseFormat(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("Messages.ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if string(result) != string(tt.expectedResult) {
					t.Errorf("Messages.ParseFormat() = %v, want %v", string(result), string(tt.expectedResult))
				}
			}
		})
	}
}

func TestWithNilFormat(t *testing.T) {
	format := WithNilFormat()

	// 确保返回的是nil格式化器
	if format.String() != "nil" {
		t.Errorf("WithNilFormat() = %v, want nil format", format.String())
	}

	// 测试解析结果
	result, err := format.Parse([]byte("test"))
	if err != nil {
		t.Errorf("WithNilFormat().Parse() error = %v", err)
	}

	if result != nil {
		t.Errorf("WithNilFormat().Parse() = %v, want nil", result)
	}
}

// 测试nil格式化器实际表现
func TestNilFormatInRealScenario(t *testing.T) {
	// 模拟一个不需要格式化的LLM响应场景
	llmResponse := `这是一段普通的文本响应，不需要任何格式化处理。
它可能包含：
- 普通文本
- 特殊字符 !@#$%
- 甚至JSON内容：{"data": "value"}
- 但整体不需要结构化解析`

	messages := NewMessages()
	messages.AppendUser("请生成一段文本", NilOutputFormat())

	// 测试解析
	result, err := messages.ParseFormat([]byte(llmResponse))
	if err != nil {
		t.Errorf("ParseFormat failed: %v", err)
	}

	// 结果应该与原始响应完全相同
	if string(result) != llmResponse {
		t.Errorf("Result mismatch:\nGot:      %s\nExpected: %s", string(result), llmResponse)
	}
}

// 对比测试：nil vs JSON格式化器
func TestCompareFormats(t *testing.T) {
	response := `{"key": "value", "text": "这是一段文本"}`

	// 使用nil格式化器
	nilMessages := NewMessages()
	nilMessages.AppendUser("测试", NilOutputFormat())
	nilResult, err := nilMessages.ParseFormat([]byte(response))
	if err != nil {
		t.Errorf("Nil format failed: %v", err)
	}

	// 使用JSON格式化器
	jsonMessages := NewMessages()
	jsonMessages.AppendUser("测试", DefaultOutputFormat())
	jsonResult, err := jsonMessages.ParseFormat([]byte(response))
	if err != nil {
		t.Errorf("JSON format failed: %v", err)
	}

	// nil格式化器应该返回原始内容
	if string(nilResult) != response {
		t.Errorf("Nil format result mismatch: got %s, want %s", string(nilResult), response)
	}

	// JSON格式化器应该返回解析后的内容（只取Assistant字段）
	expectedJSON := `{"Assistant":""}`
	if string(jsonResult) != expectedJSON {
		t.Errorf("JSON format result mismatch: got %s, want %s", string(jsonResult), expectedJSON)
	}

	// 确保两者结果不同
	if string(nilResult) == string(jsonResult) {
		t.Error("Nil format and JSON format should produce different results")
	}
}
