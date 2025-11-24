package skill

import (
	"time"
)

type Skill interface {
	Name() string
	Description() string
	Metadata() SkillMetadata
	Instructions() string
	Match(query string) float64
	Resources() []string
}

type SkillMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Version     string            `yaml:"version" json:"version"`
	Author      string            `yaml:"author,omitempty" json:"author,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Category    string            `yaml:"category,omitempty" json:"category,omitempty"`
	Keywords    []string          `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Triggers    []string          `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Required    []string          `yaml:"required,omitempty" json:"required,omitempty"`
	Optional    []string          `yaml:"optional,omitempty" json:"optional,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time         `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
	Config      map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

type SkillFile struct {
	Path     string
	Modified time.Time
	Size     int64
}

type SkillMatch struct {
	Skill  Skill
	Score  float64
	Reason string
}

func (m SkillMetadata) IsTriggered(query string) bool {
	if len(m.Triggers) == 0 {
		return false
	}

	queryLower := toLower(query)
	for _, trigger := range m.Triggers {
		if contains(queryLower, toLower(trigger)) {
			return true
		}
	}
	return false
}

func (m SkillMetadata) GetRelevanceScore(query string) float64 {
	score := 0.0
	queryLower := toLower(query)

	if contains(toLower(m.Name), queryLower) || contains(queryLower, toLower(m.Name)) {
		score += 0.8
	}

	if contains(toLower(m.Description), queryLower) {
		score += 0.4
	}

	for _, keyword := range m.Keywords {
		if contains(queryLower, toLower(keyword)) {
			score += 0.3
		}
	}

	for _, tag := range m.Tags {
		if contains(queryLower, toLower(tag)) {
			score += 0.2
		}
	}

	if m.Category != "" && contains(queryLower, toLower(m.Category)) {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

type baseSkill struct {
	metadata  SkillMetadata
	instruct  string
	resources []string
}

func (s *baseSkill) Name() string {
	return s.metadata.Name
}

func (s *baseSkill) Description() string {
	return s.metadata.Description
}

func (s *baseSkill) Metadata() SkillMetadata {
	return s.metadata
}

func (s *baseSkill) Instructions() string {
	return s.instruct
}

func (s *baseSkill) Resources() []string {
	return s.resources
}

func (s *baseSkill) Match(query string) float64 {
	if s.metadata.IsTriggered(query) {
		return 1.0
	}

	return s.metadata.GetRelevanceScore(query)
}
