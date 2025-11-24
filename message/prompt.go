package message

import (
	"fmt"

	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
)

// CacheType 缓存策略类型
type CacheType string

const (
	CacheTypeEphemeral CacheType = "ephemeral"
)

// PromptMessage 提示词消息
type PromptMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CacheType CacheType `json:"cache_type,omitempty"`
	Name      string    `json:"name,omitempty"`
}

// Prompt 结构化提示词
type Prompt struct {
	Input           string
	SystemCacheType CacheType
	Messages        []PromptMessage
	options         PromptOptions
}

// PromptOptions 提示词配置选项
type PromptOptions struct {
	OutputFormat OutputFormat
	MaxLength    int
	Rules        []string
	Steps        []string
	Examples     [][2]string
	SystemPrompt string
	Placeholder  map[string]string
}

// NewPrompt 创建新的提示词
func NewPrompt(input string, options ...func(*PromptOptions)) *Prompt {
	return &Prompt{
		Input:   input,
		options: zutil.Optional(PromptOptions{OutputFormat: defaultOutputFormatText}, options...),
	}
}

// ParseResponse 解析响应
func (p *Prompt) ParseResponse(resp []byte) (any, error) {
	return p.options.OutputFormat.Parse(resp)
}

// IsEmpty 检查提示词是否为空
func (p *Prompt) IsEmpty() bool {
	return p.options.SystemPrompt == "" && len(p.Messages) == 0 && len(p.options.Examples) == 0 && len(p.options.Rules) == 0 && p.options.MaxLength == 0 && len(p.options.Steps) == 0 // && p.options.Role == ""
}

// Bytes 生成字节数组形式的提示词
func (p *Prompt) Bytes(options ...PromptConvertOptions) []byte {
	if p.IsEmpty() {
		return []byte(p.Input)
	}

	builder := zutil.GetBuff()
	defer zutil.PutBuff(builder)

	builder.WriteString("# System\n")

	if p.options.SystemPrompt != "" {
		builder.WriteString(p.options.SystemPrompt)
		builder.WriteString("\n\n")
	} else {
		builder.WriteString("\n")
	}

	if len(p.options.Steps) > 0 {
		builder.WriteString("## Steps\n")
		builder.WriteString("Please strictly follow these steps:\n\n")
		for i := range p.options.Steps {
			builder.WriteString("  ")
			builder.WriteString(ztype.ToString(i + 1))
			builder.WriteString(". ")
			builder.WriteString(p.options.Steps[i])
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if len(p.options.Rules) > 0 {
		builder.WriteString("## Rules\n")
		builder.WriteString("Please note and strictly adhere to the following rules:\n\n")
		for _, d := range p.options.Rules {
			builder.WriteString("  - ")
			builder.WriteString(d)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	outputFormat := p.options.OutputFormat
	if len(options) > 0 && options[0].OutputFormat != nil {
		outputFormat = options[0].OutputFormat
	}
	if outputFormat != nil {
		format := definitionOutputFormat(outputFormat.String())
		if format != "" {
			builder.WriteString(format)
			builder.WriteString("\n\n")
		}
	}

	if len(p.options.Examples) > 0 {
		builder.WriteString("## Examples\n")
		builder.WriteString("Here are some examples to guide:\n\n")
		for i := range p.options.Examples {
			builder.WriteString(fmt.Sprintf("  **Example %d:**\n", i+1))
			builder.WriteString("    - **Input**: ")
			builder.WriteString(p.options.Examples[i][0])
			builder.WriteString("\n")
			builder.WriteString("    - **Output**: ")
			builder.WriteString(p.options.Examples[i][1])
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if p.options.MaxLength > 0 {
		builder.WriteString("## Max Length\n")
		builder.WriteString(fmt.Sprintf("Please limit your response to approximately %d words.", p.options.MaxLength))
		builder.WriteString("\n\n")
	}

	if len(p.Messages) > 0 {
		builder.WriteString("## Messages\n")
		for _, msg := range p.Messages {
			builder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
			if msg.CacheType != "" {
				builder.WriteString(fmt.Sprintf("(Cache: %s)\n", msg.CacheType))
			}
		}
		builder.WriteString("\n\n")
	}

	if p.Input != "" {
		builder.WriteString("\n# Input\n")
		builder.WriteString("The following content is entirely user input:\n\n")
		builder.WriteString(p.Input)
	}

	return builder.Bytes()
}

// String 返回字符串形式的提示词
func (p *Prompt) String() string {
	return zstring.Bytes2String(p.Bytes())
}
