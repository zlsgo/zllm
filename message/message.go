package message

import (
	"io"

	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/sohaha/zlsgo/zutil"
)

type Message struct {
	Role         string `json:"role"`
	Content      string `json:"content"`
	options      MessageOptions
	outputFormat bool
}

func (p *Message) Prompt() string {
	if p.options.Format != nil {
		return p.options.Format.String()
	}

	return p.Content
}

type Messages struct {
	prompt      *Prompt
	input       string
	formatInput string
	messages    []Message
	options     PromptConvertOptions
}

func NewMessages(input ...string) *Messages {
	m := &Messages{}
	if len(input) > 0 {
		m.input = input[0]
	}

	return m
}

type MessageOptions struct {
	Format OutputFormat

	InheritFormat bool
}

func (p *Messages) AppendUser(message string, wrapOutputFormat ...OutputFormat) error {
	return p.Append(Message{
		Role:    RoleUser,
		Content: message,
	}, func(options *MessageOptions) {
		if len(wrapOutputFormat) > 0 {
			options.Format = wrapOutputFormat[0]
		} else {
			if (len(p.messages) > 0 && p.messages[len(p.messages)-1].Role != RoleUser) && p.prompt.options.OutputFormat == nil {
				options.InheritFormat = true
			}
		}
	})
}

func (p *Messages) AppendAssistant(message string, wrapFormat ...OutputFormat) error {
	return p.Append(Message{
		Role:    RoleAssistant,
		Content: message,
	}, func(options *MessageOptions) {
		if len(wrapFormat) > 0 {
			options.Format = wrapFormat[0]
		} else {
			if ((len(p.messages) > 0 && p.messages[len(p.messages)-1].Role == RoleUser) || (len(p.messages) == 0)) && (p.prompt == nil || p.prompt.options.OutputFormat == nil) {
				options.InheritFormat = true
			}
		}
	})
}

func (p *Messages) Append(message Message, options ...func(options *MessageOptions)) error {
	message.options = zutil.Optional(MessageOptions{}, options...)
	message.outputFormat = message.options.Format != nil

	switch message.Role {
	case RoleAssistant:
		if !message.outputFormat && message.options.InheritFormat {
			message.outputFormat = true
			message.options.Format = defaultOutputFormatText
			// message.options.Format, _ = p.prompt.options.Format.Format(message.Content)
		}
	case RoleUser:
		if !message.outputFormat && message.options.InheritFormat {
			message.outputFormat = true
			message.options.Format = defaultOutputFormatText
		}
	}

	p.messages = append(p.messages, message)
	return nil
}

func (p *Messages) Input() string {
	if p.formatInput != "" {
		return p.formatInput
	}

	return p.input
}

func (p *Messages) OutputFormat() OutputFormat {
	return p.options.OutputFormat
}

func (p *Messages) Clear() {
	p.messages = p.messages[:0:0]
}

func (p *Messages) ForEach(fn func(i int, message Message)) {
	for i := range p.messages {
		fn(i, p.messages[i])
	}
}

func (p *Messages) Len() int {
	return len(p.messages)
}

func (p *Messages) ParseFormat(response []byte) ([]byte, error) {
	var outputFormat OutputFormat

	// 上一条消息如果不指定格式则直接返回响应
	if len(p.messages) > 0 && p.messages[len(p.messages)-1].outputFormat {
		outputFormat = p.messages[len(p.messages)-1].options.Format
	} else if p.options.OutputFormat != nil {
		outputFormat = p.options.OutputFormat
	} else if p.prompt != nil && !p.prompt.IsEmpty() {
		outputFormat = p.prompt.options.OutputFormat
	}

	if outputFormat != nil {
		output, err := outputFormat.Parse(response)
		if err != nil {
			return nil, err
		}

		return ztype.ToBytes(output), nil
	}

	return response, nil
}

func (p *Messages) History(wrapPrompt bool) [][]string {
	m := make([][]string, 0, p.Len()+1)

	if p.formatInput != "" || p.input != "" {
		role := RoleUser
		if p.input == "" {
			role = RoleSystem
		}
		if wrapPrompt && p.formatInput != "" {
			m = append(m, []string{role, p.formatInput})
		} else {
			m = append(m, []string{role, p.input})
		}
	}

	for i := range p.messages {
		if wrapPrompt && p.messages[i].options.Format != nil {
			// 如果最后一条是用户就要把格式带上
			if p.messages[i].Role != RoleUser || (p.messages[i].Role == RoleUser && i == len(p.messages)-1) {
				if p.messages[i].Role == RoleUser {
					format := definitionOutputFormat(p.messages[i].options.Format.String())
					if format != "" {
						m = append(m, []string{p.messages[i].Role, "# System\n\n" + format + "\n\n\n# Input\nThe following content is entirely user input:\n\n" + p.messages[i].Content})
					} else {
						m = append(m, []string{p.messages[i].Role, "# System\n\n" + "\n\n\n# Input\nThe following content is entirely user input:\n\n" + p.messages[i].Content})
					}
				} else {
					c, err := p.messages[i].options.Format.Format(p.messages[i].Content)
					if err != nil {
						continue
					}
					m = append(m, []string{p.messages[i].Role, c})
				}
				continue
			}
		}

		m = append(m, []string{p.messages[i].Role, p.messages[i].Content})
	}

	return m
}

func (p *Messages) String() string {
	history := p.History(false)
	s := zstring.Buffer((len(history) * 4))

	for i := range history {
		if i > 0 {
			s.WriteString("\n")
		}

		s.WriteString(history[i][0])
		s.WriteString(": ")
		s.WriteString(history[i][1])
	}

	return s.String()
}

type PromptConvertOptions struct {
	Placeholder  map[string]string
	OutputFormat OutputFormat
}

func (p *Prompt) ConvertToMessages(options ...PromptConvertOptions) (messages *Messages, err error) {
	t := zstring.String2Bytes(p.Input)

	var o PromptConvertOptions
	if len(options) > 0 {
		o = options[0]
	} else {
		o = PromptConvertOptions{}
	}

	ut := p.Bytes(o)

	if len(o.Placeholder) > 0 || len(p.options.Placeholder) > 0 {
		ut, err = p.buildTemplate(zstring.Bytes2String(ut), o.Placeholder)
		if err != nil {
			return
		}
		t, _ = p.buildTemplate(zstring.Bytes2String(t), o.Placeholder)
	}

	messages = &Messages{
		prompt:      p,
		input:       zstring.Bytes2String(t),
		formatInput: zstring.Bytes2String(ut),
		messages:    []Message{},
		options:     o,
	}

	return
}

func (p *Prompt) buildTemplate(template string, placeholder ...map[string]string) ([]byte, error) {
	t, err := zstring.NewTemplate(template, "{{", "}}")
	if err != nil {
		return nil, err
	}

	builder := zutil.GetBuff()
	defer zutil.PutBuff(builder)

	var placeholderMap map[string]string
	if len(placeholder) > 0 {
		placeholderMap = placeholder[0]
	}

	_, err = t.Process(builder, func(w io.Writer, tag string) (int, error) {
		if placeholderMap != nil {
			if v, ok := placeholderMap[tag]; ok {
				return builder.WriteString(v)
			}
		}

		if len(p.options.Placeholder) == 0 {
			return 0, nil
		}

		if v, ok := p.options.Placeholder[tag]; ok {
			return builder.WriteString(v)
		}
		return 0, nil
	})

	return builder.Bytes(), err
}
