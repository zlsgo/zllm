package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/agent"
	"github.com/zlsgo/zllm/message"
	"github.com/zlsgo/zllm/runtime"
)

type SkillsProvider struct {
	agent   agent.LLM
	manager SkillManager
	config  SkillsConfig
}

type SkillsConfig struct {
	Enabled          bool
	MaxSkills        int
	MinScore         float64
	InjectAsSystem   bool
	InjectAsUser     bool
	SkillPrefix      string
	SkillSuffix      string
	ExcludeProviders []string
}

func DefaultSkillsConfig() SkillsConfig {
	return SkillsConfig{
		Enabled:        true,
		MaxSkills:      3,
		MinScore:       0.3,
		InjectAsSystem: true,
		InjectAsUser:   false,
		SkillPrefix:    "\n\n--- Relevant Skills ---\n",
		SkillSuffix:    "\n--- End of Skills ---\n\n",
	}
}

func NewSkillsProvider(baseAgent agent.LLM, manager SkillManager, config SkillsConfig) agent.LLM {
	if config.MaxSkills <= 0 {
		config.MaxSkills = 3
	}
	if config.MinScore <= 0 {
		config.MinScore = 0.3
	}

	return &SkillsProvider{
		agent:   baseAgent,
		manager: manager,
		config:  config,
	}
}

func (p *SkillsProvider) Generate(ctx context.Context, data []byte) (*zjson.Res, error) {
	if !p.config.Enabled {
		runtime.Log("Skills provider disabled, delegating to base agent")
		return p.agent.Generate(ctx, data)
	}

	runtime.Log("Skills provider processing request")

	query, err := p.extractQueryFromRequest(data)
	if err != nil {
		runtime.Log("Failed to extract query from request:", err, "falling back to base agent")
		return p.agent.Generate(ctx, data)
	}

	runtime.Log("Extracted query:", query)

	skills := p.manager.FindRelevantSkills(query, p.config.MaxSkills)

	var relevantSkills []SkillMatch
	for _, match := range skills {
		if match.Score >= p.config.MinScore {
			relevantSkills = append(relevantSkills, match)
			runtime.Log("Selected relevant skill:", match.Skill.Name(), "score:", match.Score)
		}
	}

	if len(relevantSkills) == 0 {
		runtime.Log("No relevant skills found with minimum score:", p.config.MinScore, "falling back to base agent")
		return p.agent.Generate(ctx, data)
	}

	runtime.Log("Injecting", len(relevantSkills), "skills into request")

	modifiedData, err := p.injectSkills(data, relevantSkills)
	if err != nil {
		runtime.Log("Failed to inject skills:", err, "falling back to base agent")
		return p.agent.Generate(ctx, data)
	}

	return p.agent.Generate(ctx, modifiedData)
}

func (p *SkillsProvider) Stream(ctx context.Context, data []byte, callback func(string, []byte)) (<-chan *zjson.Res, error) {
	if !p.config.Enabled {
		return p.agent.Stream(ctx, data, callback)
	}

	query, err := p.extractQueryFromRequest(data)
	if err != nil {
		return p.agent.Stream(ctx, data, callback)
	}

	skills := p.manager.FindRelevantSkills(query, p.config.MaxSkills)

	var relevantSkills []SkillMatch
	for _, match := range skills {
		if match.Score >= p.config.MinScore {
			relevantSkills = append(relevantSkills, match)
		}
	}

	if len(relevantSkills) == 0 {
		return p.agent.Stream(ctx, data, callback)
	}

	modifiedData, err := p.injectSkills(data, relevantSkills)
	if err != nil {
		return p.agent.Stream(ctx, data, callback)
	}

	return p.agent.Stream(ctx, modifiedData, callback)
}

func (p *SkillsProvider) PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) ([]byte, error) {
	if !p.config.Enabled {
		return p.agent.PrepareRequest(messages, options...)
	}

	query := p.getLastUserMessage(messages)
	if query == "" {
		return p.agent.PrepareRequest(messages, options...)
	}

	skills := p.manager.FindRelevantSkills(query, p.config.MaxSkills)

	var relevantSkills []SkillMatch
	for _, match := range skills {
		if match.Score >= p.config.MinScore {
			relevantSkills = append(relevantSkills, match)
		}
	}

	if len(relevantSkills) == 0 {
		return p.agent.PrepareRequest(messages, options...)
	}

	modifiedMessages := p.injectSkillsIntoMessages(messages, relevantSkills)

	return p.agent.PrepareRequest(modifiedMessages, options...)
}

func (p *SkillsProvider) ParseResponse(body *zjson.Res) (*agent.Response, error) {
	return p.agent.ParseResponse(body)
}

func (p *SkillsProvider) extractQueryFromRequest(data []byte) (string, error) {
	req := zjson.ParseBytes(data)
	if !req.Exists() {
		return "", fmt.Errorf("invalid request format")
	}

	messages := req.Get("messages").Array()
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages found")
	}

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Get("role").String() == "user" {
			content := msg.Get("content")
			if content.IsArray() {
				for _, part := range content.Array() {
					if part.Get("type").String() == "text" {
						return part.Get("text").String(), nil
					}
				}
			} else {
				return content.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no user message found")
}

func (p *SkillsProvider) injectSkills(data []byte, skills []SkillMatch) ([]byte, error) {
	req := zjson.ParseBytes(data)
	if !req.Exists() {
		return nil, fmt.Errorf("invalid request format")
	}

	skillsText := p.formatSkills(skills)
	skillContent := p.config.SkillPrefix + skillsText + p.config.SkillSuffix

	var request map[string]interface{}
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("failed to parse request JSON: %v", err)
	}

	messages, ok := request["messages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid messages format")
	}

	if p.config.InjectAsSystem {
		skillMsg := map[string]interface{}{
			"role":    "system",
			"content": skillContent,
		}

		if len(messages) > 0 {
			if firstMsg, ok := messages[0].(map[string]interface{}); ok {
				if role, exists := firstMsg["role"]; exists && role == "system" {
					if content, exists := firstMsg["content"]; exists {
						if contentStr, ok := content.(string); ok {
							firstMsg["content"] = contentStr + skillContent
						}
					} else {
						firstMsg["content"] = skillContent
					}
				} else {
					messages = append([]interface{}{skillMsg}, messages...)
				}
			}
		} else {
			messages = []interface{}{skillMsg}
		}
	} else if p.config.InjectAsUser {
		skillMsg := map[string]interface{}{
			"role":    "user",
			"content": skillContent,
		}

		insertIndex := len(messages)
		for i := len(messages) - 1; i >= 0; i-- {
			if msg, ok := messages[i].(map[string]interface{}); ok {
				if role, exists := msg["role"]; exists && role == "user" {
					insertIndex = i
					break
				}
			}
		}

		newMessages := make([]interface{}, 0, len(messages)+1)
		newMessages = append(newMessages, messages[:insertIndex]...)
		newMessages = append(newMessages, skillMsg)
		newMessages = append(newMessages, messages[insertIndex:]...)
		messages = newMessages
	}

	request["messages"] = messages
	return json.Marshal(request)
}

func (p *SkillsProvider) injectSkillsIntoMessages(messages *message.Messages, skills []SkillMatch) *message.Messages {
	skillsText := p.formatSkills(skills)

	history := messages.History(true)

	if p.config.InjectAsSystem {
		finalMessages := &message.Messages{}

		skillMsg := message.Message{
			Role:    "system",
			Content: p.config.SkillPrefix + skillsText + p.config.SkillSuffix,
		}
		finalMessages.Append(skillMsg)

		for _, msg := range history {
			newMsg := message.Message{
				Role:    msg[0],
				Content: msg[1],
			}
			finalMessages.Append(newMsg)
		}

		return finalMessages
	}

	newMessages := &message.Messages{}
	for _, msg := range history {
		newMsg := message.Message{
			Role:    msg[0],
			Content: msg[1],
		}
		newMessages.Append(newMsg)
	}

	return newMessages
}

func (p *SkillsProvider) formatSkills(skills []SkillMatch) string {
	if len(skills) == 0 {
		return ""
	}

	var parts []string
	for _, match := range skills {
		skill := match.Skill
		part := fmt.Sprintf("## %s\n", skill.Name())
		if skill.Description() != "" {
			part += fmt.Sprintf("Description: %s\n", skill.Description())
		}
		part += fmt.Sprintf("Instructions: %s\n", skill.Instructions())
		part += fmt.Sprintf("Relevance: %.2f", match.Score)
		parts = append(parts, part)
	}

	return strings.Join(parts, "\n\n")
}

func (p *SkillsProvider) getLastUserMessage(messages *message.Messages) string {
	history := messages.History(true)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i][0] == "user" {
			return history[i][1]
		}
	}
	return ""
}

func WithSkills(agent agent.LLM, manager SkillManager, options ...func(SkillsConfig)) agent.LLM {
	config := DefaultSkillsConfig()
	for _, opt := range options {
		opt(config)
	}
	return NewSkillsProvider(agent, manager, config)
}

func WithSkillsEnabled(enabled bool) func(SkillsConfig) {
	return func(config SkillsConfig) {
		config.Enabled = enabled
	}
}

func WithMaxSkills(max int) func(SkillsConfig) {
	return func(config SkillsConfig) {
		config.MaxSkills = max
	}
}

func WithMinScore(score float64) func(SkillsConfig) {
	return func(config SkillsConfig) {
		config.MinScore = score
	}
}

func WithSkillInjection(asSystem, asUser bool) func(SkillsConfig) {
	return func(config SkillsConfig) {
		config.InjectAsSystem = asSystem
		config.InjectAsUser = asUser
	}
}
