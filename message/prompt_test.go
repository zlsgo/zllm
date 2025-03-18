package message_test

import (
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/zlsgo/zllm/message"
)

func TestNewPrompt(t *testing.T) {
	tt := zlsgo.NewTest(t)

	tt.Run("Base", func(tt *zlsgo.TestUtil) {
		p := message.NewPrompt("你好呀")
		tt.EqualExit("你好呀", p.String())
	})

	tt.Run("With MaxLength", func(tt *zlsgo.TestUtil) {
		p := message.NewPrompt("形容一下人工智能做什么?", func(po *message.PromptOptions) {
			po.MaxLength = 8
			po.Steps = []string{"分析用户输入", "生成回答"}
		})
		tt.EqualExit(`# System

## Steps
Please strictly follow these steps:

  1. 分析用户输入
  2. 生成回答

## Output Format
Please strictly adhere to this output format, do not include any extra content, where "{}" represents a placeholder:

{"Assistant":"{}"}

## Max Length
Please limit your response to approximately 8 words.


# Input
The following content is entirely user input:

形容一下人工智能做什么?`, p.String())
	})

	tt.Run("With System Prompt", func(tt *zlsgo.TestUtil) {
		p := message.NewPrompt("你好", func(po *message.PromptOptions) {
			po.SystemPrompt = "您是轻舟，一个古文生成机器人，富有浪漫主意，古诗词底蕴也很强。"
			po.Rules = []string{
				"将用户的白话文转成故事的形式",
				"富有浪漫主意，古诗词底蕴也很强",
				"转化的故事要符合原文的语义",
			}
		})
		tt.EqualExit(`# System
您是轻舟，一个古文生成机器人，富有浪漫主意，古诗词底蕴也很强。

## Rules
Please note and strictly adhere to the following rules:

  - 将用户的白话文转成故事的形式
  - 富有浪漫主意，古诗词底蕴也很强
  - 转化的故事要符合原文的语义

## Output Format
Please strictly adhere to this output format, do not include any extra content, where "{}" represents a placeholder:

{"Assistant":"{}"}


# Input
The following content is entirely user input:

你好`, p.String())
	})
}

func TestPromptSet(t *testing.T) {
	tt := zlsgo.NewTest(t)

	pmpt := message.NewPrompt("你好呀, 你叫{{name}}，今年{{age}}岁", func(po *message.PromptOptions) {
		po.Placeholder = map[string]string{"name": "小明", "age": "18"}
	})

	msg, err := pmpt.ConvertToMessages()
	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫小明，今年18岁", msg.String())
	msg.AppendAssistant("谢谢")

	msg, err = pmpt.ConvertToMessages(message.PromptConvertOptions{
		Placeholder: map[string]string{"name": "大白", "age": "20"},
	})

	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫大白，今年20岁", msg.String())
	msg.AppendAssistant("我叫大白")

	msg, err = pmpt.ConvertToMessages(message.PromptConvertOptions{
		Placeholder: map[string]string{"name": "🐸"},
	})
	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫🐸，今年18岁", msg.String())
}

func TestPromptEmpty(t *testing.T) {
	tt := zlsgo.NewTest(t)

	pmpt := message.NewPrompt("你好呀, 你叫{{name}}，今年{{age}}岁，你来自{{address}}", func(po *message.PromptOptions) {
		po.Placeholder = map[string]string{"name": "小明", "age": "18"}
	})

	msg, err := pmpt.ConvertToMessages()
	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫小明，今年18岁，你来自", msg.String())
}
