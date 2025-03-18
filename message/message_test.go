package message_test

import (
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/zlsgo/zllm/message"
)

type testOutputFormat struct{}

var _ message.OutputFormat = testOutputFormat{}

func (p testOutputFormat) Parse(resp []byte) (any, error) {
	return zjson.GetBytes(resp, "结果").String(), nil
}

func (p testOutputFormat) Format(str string) (string, error) {
	return zjson.Set("{}", "结果", str)
}

func (p testOutputFormat) String() string {
	return `{"结果":"{}"}`
}

var testOutput = testOutputFormat{}

func TestMessages(t *testing.T) {
	tt := zlsgo.NewTest(t)

	tt.Run("Base", func(tt *zlsgo.TestUtil) {
		msg := message.NewMessages()
		msg.AppendUser("你好呀")
		tt.EqualExit("user: 你好呀", msg.String())
	})

	tt.Run("WrapOutputFormat", func(tt *zlsgo.TestUtil) {
		msg := message.NewMessages()
		msg.AppendUser("你好呀")
		msg.AppendUser("你是小明吗?", testOutput)
		tt.EqualExit("user: 你好呀\nuser: 你是小明吗?", msg.String())
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleUser, "你是小明吗?"}}, msg.History(false))
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleUser, "# System\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"结果\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你是小明吗?"}}, msg.History(true))
	})

	tt.Run("Append", func(tt *zlsgo.TestUtil) {
		msg := message.NewMessages("你好呀")
		tt.EqualExit("user: 你好呀", msg.String())
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}}, msg.History(false))

		msg.Append(message.Message{Role: "assistant", Content: "好的呀"})
		tt.EqualExit("user: 你好呀\nassistant: 好的呀", msg.String())
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}}, msg.History(false))

		msg.AppendUser("你叫什么名字", testOutput)
		tt.EqualExit("user: 你好呀\nassistant: 好的呀\nuser: 你叫什么名字", msg.String())
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}, {message.RoleUser, "你叫什么名字"}}, msg.History(false))
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}, {message.RoleUser, "# System\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"结果\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你叫什么名字"}}, msg.History(true))
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}, {message.RoleUser, "# System\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"结果\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你叫什么名字"}}, msg.History(true))

		msg.AppendAssistant("我叫小明", testOutput)
		tt.EqualExit("user: 你好呀\nassistant: 好的呀\nuser: 你叫什么名字\nassistant: 我叫小明", msg.String())
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}, {message.RoleUser, "你叫什么名字"}, {message.RoleAssistant, "我叫小明"}}, msg.History(false))
		tt.EqualExit([][]string{{message.RoleUser, "你好呀"}, {message.RoleAssistant, "好的呀"}, {message.RoleUser, "你叫什么名字"}, {message.RoleAssistant, "{\"结果\":\"我叫小明\"}"}}, msg.History(true))
	})
}

func TestPromptMessages(t *testing.T) {
	tt := zlsgo.NewTest(t)

	pmpt := message.NewPrompt("你好呀, 你叫{{name}}", func(po *message.PromptOptions) {
		po.Placeholder = map[string]string{"name": "小明"}
		po.SystemPrompt = "你是一个机器人"
	})

	msg, err := pmpt.ConvertToMessages(message.PromptConvertOptions{
		Placeholder: map[string]string{"name": "大白"},
	})
	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫大白", msg.String())
	tt.EqualExit([][]string{{message.RoleUser, "你好呀, 你叫大白"}}, msg.History(false))
	tt.EqualExit([][]string{{message.RoleUser, "# System\n你是一个机器人\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"Assistant\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你好呀, 你叫大白"}}, msg.History(true))

	msg.AppendAssistant("我现在叫大白")
	tt.EqualExit("user: 你好呀, 你叫大白\nassistant: 我现在叫大白", msg.String())
	tt.EqualExit([][]string{{message.RoleUser, "你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}}, msg.History(false))
	tt.EqualExit([][]string{{message.RoleUser, "# System\n你是一个机器人\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"Assistant\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}}, msg.History(true))

	msg.AppendUser("你叫什么名字")
	tt.EqualExit("user: 你好呀, 你叫大白\nassistant: 我现在叫大白\nuser: 你叫什么名字", msg.String())
	tt.EqualExit([][]string{{message.RoleUser, "你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}, {message.RoleUser, "你叫什么名字"}}, msg.History(false))

	tt.EqualExit([][]string{{message.RoleUser, "# System\n你是一个机器人\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"Assistant\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}, {message.RoleUser, "你叫什么名字"}}, msg.History(true))

	msg.AppendAssistant("我是大白")
	tt.EqualExit("user: 你好呀, 你叫大白\nassistant: 我现在叫大白\nuser: 你叫什么名字\nassistant: 我是大白", msg.String())
	tt.EqualExit([][]string{{message.RoleUser, "你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}, {message.RoleUser, "你叫什么名字"}, {message.RoleAssistant, "我是大白"}}, msg.History(false))
	tt.EqualExit([][]string{{message.RoleUser, "# System\n你是一个机器人\n\n## Output Format\nPlease strictly adhere to this output format, do not include any extra content, where \"{}\" represents a placeholder:\n\n{\"Assistant\":\"{}\"}\n\n\n# Input\nThe following content is entirely user input:\n\n你好呀, 你叫大白"}, {message.RoleAssistant, "我现在叫大白"}, {message.RoleUser, "你叫什么名字"}, {message.RoleAssistant, "我是大白"}}, msg.History(true))

	msg, err = pmpt.ConvertToMessages()
	tt.NoError(err)
	tt.EqualExit("user: 你好呀, 你叫小明", msg.String())
}
