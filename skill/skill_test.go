package skill

import (
	"testing"
)

func TestSkillMetadata_IsTriggered(t *testing.T) {
	tests := []struct {
		name     string
		metadata SkillMetadata
		query    string
		want     bool
	}{
		{
			name: "exact trigger match",
			metadata: SkillMetadata{
				Triggers: []string{"code review", "代码审查"},
			},
			query: "请帮我进行 code review",
			want:  true,
		},
		{
			name: "partial trigger match",
			metadata: SkillMetadata{
				Triggers: []string{"数据分析"},
			},
			query: "我需要一些数据分析的帮助",
			want:  true,
		},
		{
			name: "no triggers",
			metadata: SkillMetadata{
				Triggers: []string{},
			},
			query: "随便的查询",
			want:  false,
		},
		{
			name: "no match",
			metadata: SkillMetadata{
				Triggers: []string{"代码审查"},
			},
			query: "我想学习编程",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metadata.IsTriggered(tt.query); got != tt.want {
				t.Errorf("SkillMetadata.IsTriggered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkillMetadata_GetRelevanceScore(t *testing.T) {
	tests := []struct {
		name     string
		metadata SkillMetadata
		query    string
		want     float64
	}{
		{
			name: "name match",
			metadata: SkillMetadata{
				Name:        "Code Review Assistant",
				Description: "专业的代码审查工具",
				Tags:        []string{"development"},
				Keywords:    []string{"analysis"}, // 不匹配 query
			},
			query: "code review",
			want:  0.8, // 仅名称匹配
		},
		{
			name: "description match",
			metadata: SkillMetadata{
				Name:        "代码助手",
				Description: "专业的代码审查工具",
				Tags:        []string{"development"},
				Keywords:    []string{"programming"},
			},
			query: "审查工具",
			want:  0.4, // 仅描述匹配
		},
		{
			name: "keyword match",
			metadata: SkillMetadata{
				Name:        "文件管理助手", // 名称不包含查询词
				Description: "文件处理工具", // 描述不包含查询词
				Tags:        []string{"documentation"},
				Keywords:    []string{"格式化", "排版"},
			},
			query: "格式化",
			want:  0.3, // 仅关键词匹配
		},
		{
			name: "no match",
			metadata: SkillMetadata{
				Name:        "数据分析",
				Description: "数据统计工具",
				Tags:        []string{"analytics"},
				Keywords:    []string{"统计", "图表"},
			},
			query: "编程学习",
			want:  0.0,
		},
		{
			name: "multiple matches",
			metadata: SkillMetadata{
				Name:        "Code Review",
				Description: "专业的代码审查工具",
				Tags:        []string{"code", "review"},
				Keywords:    []string{"代码", "审查"},
				Category:    "Development",
			},
			query: "code review 代码审查",
			want:  1.0, // 多重匹配，但限制在1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.GetRelevanceScore(tt.query)
			if got != tt.want {
				t.Logf("Debug: Query='%s', Name='%s', Keywords=%v", tt.query, tt.metadata.Name, tt.metadata.Keywords)
				t.Errorf("SkillMetadata.GetRelevanceScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseSkill(t *testing.T) {
	metadata := SkillMetadata{
		Name:        "Test Skill",
		Description: "A test skill for unit testing",
		Version:     "1.0.0",
		Author:      "Test Author",
		Tags:        []string{"test", "unit"},
		Keywords:    []string{"测试", "单元"},
		Category:    "Testing",
	}

	instructions := "This is a test skill with detailed instructions."
	resources := []string{"test1.txt", "test2.md"}

	skill := &baseSkill{
		metadata:  metadata,
		instruct:  instructions,
		resources: resources,
	}

	if skill.Name() != metadata.Name {
		t.Errorf("baseSkill.Name() = %v, want %v", skill.Name(), metadata.Name)
	}

	if skill.Description() != metadata.Description {
		t.Errorf("baseSkill.Description() = %v, want %v", skill.Description(), metadata.Description)
	}

	if skill.Metadata().Name != metadata.Name || skill.Metadata().Description != metadata.Description {
		t.Errorf("baseSkill.Metadata() = %v, want %v", skill.Metadata(), metadata)
	}

	if skill.Instructions() != instructions {
		t.Errorf("baseSkill.Instructions() = %v, want %v", skill.Instructions(), instructions)
	}

	if len(skill.Resources()) != len(resources) {
		t.Errorf("baseSkill.Resources() length = %v, want %v", len(skill.Resources()), len(resources))
	}
	for i, res := range skill.Resources() {
		if res != resources[i] {
			t.Errorf("baseSkill.Resources()[%v] = %v, want %v", i, res, resources[i])
		}
	}

	score := skill.Match("test unit testing")
	if score <= 0 {
		t.Errorf("baseSkill.Match() should return positive score for matching query, got %v", score)
	}

	triggersMetadata := SkillMetadata{
		Name:     "Trigger Test",
		Triggers: []string{"test trigger"},
	}
	triggerSkill := &baseSkill{
		metadata: triggersMetadata,
		instruct: "Test with trigger",
	}

	score1 := triggerSkill.Match("This is a test trigger")
	score2 := triggerSkill.Match("This is a test trigger") // Second call should use cache
	if score1 != score2 {
		t.Errorf("Match() should return same score on second call due to caching: %v != %v", score1, score2)
	}
	if score1 != 1.0 {
		t.Errorf("Match() should return 1.0 for exact trigger match, got %v", score1)
	}
}

func TestValidateSkill(t *testing.T) {
	tests := []struct {
		name     string
		metadata SkillMetadata
		wantErr  bool
		errCount int
	}{
		{
			name: "valid skill",
			metadata: SkillMetadata{
				Name:        "Valid Skill",
				Description: "A valid skill for testing",
				Version:     "1.0.0",
				Triggers:    []string{"valid trigger"},
				Keywords:    []string{"valid"},
			},
			wantErr:  false,
			errCount: 0,
		},
		{
			name: "empty name",
			metadata: SkillMetadata{
				Name:        "",
				Description: "Skill with empty name",
				Version:     "1.0.0",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "name too long",
			metadata: SkillMetadata{
				Name:        string(make([]byte, 101)), // 101 characters
				Description: "Skill with long name",
				Version:     "1.0.0",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "empty description",
			metadata: SkillMetadata{
				Name:        "Test Skill",
				Description: "",
				Version:     "1.0.0",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "multiple errors",
			metadata: SkillMetadata{
				Name:        "",
				Description: "",
				Version:     "1.0.0",
				Triggers:    []string{"", "valid trigger"},
				Keywords:    []string{"", string(make([]byte, 51))}, // empty and too long
			},
			wantErr:  true,
			errCount: 6, // name, description, trigger[0], keyword[0], keyword[1] length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateSkill(tt.metadata)
			hasErr := len(errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("ValidateSkill() hasErr = %v, wantErr %v", hasErr, tt.wantErr)
				return
			}
			if len(errors) != tt.errCount {
				t.Errorf("ValidateSkill() error count = %v, want %v", len(errors), tt.errCount)
				for _, err := range errors {
					t.Logf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestSkillFilter(t *testing.T) {
	skills := []Skill{
		&baseSkill{
			metadata: SkillMetadata{
				Name:     "Code Review",
				Category: "Development",
				Tags:     []string{"code", "review"},
				Author:   "Dev Team",
			},
		},
		&baseSkill{
			metadata: SkillMetadata{
				Name:     "Data Analysis",
				Category: "Analytics",
				Tags:     []string{"data", "analysis"},
				Author:   "Data Team",
			},
		},
		&baseSkill{
			metadata: SkillMetadata{
				Name:     "Document Format",
				Category: "Documentation",
				Tags:     []string{"doc", "format"},
				Author:   "Doc Team",
			},
		},
	}

	tests := []struct {
		name    string
		filter  SkillFilter
		matches []int // indices of matching skills
	}{
		{
			name: "filter by category",
			filter: SkillFilter{
				Categories: []string{"Development"},
			},
			matches: []int{0},
		},
		{
			name: "filter by tag",
			filter: SkillFilter{
				Tags: []string{"data"},
			},
			matches: []int{1},
		},
		{
			name: "filter by author",
			filter: SkillFilter{
				Author: "Doc Team",
			},
			matches: []int{2},
		},
		{
			name: "filter by multiple categories",
			filter: SkillFilter{
				Categories: []string{"Development", "Analytics"},
			},
			matches: []int{0, 1},
		},
		{
			name: "no matches",
			filter: SkillFilter{
				Categories: []string{"Nonexistent"},
			},
			matches: []int{},
		},
		{
			name:    "empty filter",
			filter:  SkillFilter{},
			matches: []int{0, 1, 2}, // All should match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched []int
			for i, skill := range skills {
				if tt.filter.Match(skill) {
					matched = append(matched, i)
				}
			}

			if len(matched) != len(tt.matches) {
				t.Errorf("SkillFilter.Match() matched count = %v, want %v", len(matched), len(tt.matches))
				return
			}

			for i, match := range matched {
				if match != tt.matches[i] {
					t.Errorf("SkillFilter.Match() matches[%v] = %v, want %v", i, match, tt.matches[i])
				}
			}
		})
	}
}
