package agent

import (
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zstring"
	"github.com/zlsgo/zllm/message"
)

func CompleteMessag(agent LLMAgent, body []byte) ([]byte, error) {
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
