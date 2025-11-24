package skill

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

type mockLoader struct {
	skills map[string]Skill
	errors []string
	mu     sync.Mutex
}

func newMockLoader() *mockLoader {
	return &mockLoader{
		skills: make(map[string]Skill),
		errors: make([]string, 0),
	}
}

func (m *mockLoader) LoadFromPaths(paths []string) ([]Skill, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var skills []Skill
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}

	return skills, append([]string{}, m.errors...)
}

func (m *mockLoader) LoadFromPath(path string) ([]Skill, []string) {
	return m.LoadFromPaths([]string{path})
}

func (m *mockLoader) LoadSkill(dirPath string) (Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if skill, exists := m.skills[dirPath]; exists {
		return skill, nil
	}
	return nil, fmt.Errorf("skill not found: %s", dirPath)
}

func (m *mockLoader) addSkill(path string, skill Skill) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills[path] = skill
}

func (m *mockLoader) addError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, err)
}

func TestDefaultSkillManager_LoadSkills(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skill1 := &baseSkill{
		metadata: SkillMetadata{Name: "Test Skill 1", Description: "Description 1"},
		instruct: "Instructions 1",
	}
	skill2 := &baseSkill{
		metadata: SkillMetadata{Name: "Test Skill 2", Description: "Description 2"},
		instruct: "Instructions 2",
	}

	loader.addSkill("/path/to/skill1", skill1)
	loader.addSkill("/path/to/skill2", skill2)

	paths := []string{"/path/to"}
	err := manager.LoadSkills(paths)
	if err != nil {
		t.Errorf("LoadSkills() error = %v", err)
	}

	stats := manager.Stats()
	if stats.TotalSkills != 2 {
		t.Errorf("LoadSkills() TotalSkills = %v, want 2", stats.TotalSkills)
	}

	skill, exists := manager.GetSkill("Test Skill 1")
	if !exists {
		t.Error("GetSkill() should find loaded skill")
	}
	if skill.Name() != "Test Skill 1" {
		t.Errorf("GetSkill() returned skill with name = %v, want Test Skill 1", skill.Name())
	}
}

func TestDefaultSkillManager_LoadSkills_WithErrors(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skill1 := &baseSkill{
		metadata: SkillMetadata{Name: "Test Skill 1", Description: "Description 1"},
		instruct: "Instructions 1",
	}

	loader.addSkill("/path/to/skill1", skill1)
	loader.addError("failed to load skill2")
	loader.addError("failed to load skill3")

	paths := []string{"/path/to"}
	err := manager.LoadSkills(paths)
	if err == nil {
		t.Error("LoadSkills() should return error when there are load errors")
	}
	if !strings.Contains(err.Error(), "loaded 1 skills with 2 errors") {
		t.Errorf("LoadSkills() error message unexpected: %v", err)
	}

	stats := manager.Stats()
	if stats.TotalSkills != 1 {
		t.Errorf("LoadSkills() TotalSkills = %v, want 1", stats.TotalSkills)
	}

	if len(stats.LoadErrors) != 2 {
		t.Errorf("LoadSkills() LoadErrors count = %v, want 2", len(stats.LoadErrors))
	}
}

func TestDefaultSkillManager_FindRelevantSkills(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skills := []Skill{
		&baseSkill{
			metadata: SkillMetadata{
				Name:        "Code Review Assistant",
				Description: "专业的代码审查工具",
				Keywords:    []string{"代码", "审查"},
				Triggers:    []string{"code review"},
			},
			instruct: "Code review instructions",
		},
		&baseSkill{
			metadata: SkillMetadata{
				Name:        "Data Analysis Tool",
				Description: "数据分析工具",
				Keywords:    []string{"数据", "分析"},
				Triggers:    []string{"data analysis"},
			},
			instruct: "Data analysis instructions",
		},
		&baseSkill{
			metadata: SkillMetadata{
				Name:        "Document Formatter",
				Description: "文档格式化工具",
				Keywords:    []string{"文档", "格式化"},
				Triggers:    []string{"format document"},
			},
			instruct: "Document formatting instructions",
		},
	}

	for i, skill := range skills {
		loader.addSkill(string(rune(i)), skill)
	}

	manager.LoadSkills([]string{"/"})

	tests := []struct {
		name        string
		query       string
		limit       int
		expectCount int
		expectNames []string
	}{
		{
			name:        "exact trigger match",
			query:       "code review",
			limit:       5,
			expectCount: 1,
			expectNames: []string{"Code Review Assistant"},
		},
		{
			name:        "partial matches",
			query:       "数据",
			limit:       5,
			expectCount: 1,
			expectNames: []string{"Data Analysis Tool"},
		},
		{
			name:        "no matches",
			query:       " unrelated query",
			limit:       5,
			expectCount: 0,
			expectNames: []string{},
		},
		{
			name:        "limit results",
			query:       "数据", // 匹配"数据分析工具"
			limit:       2,
			expectCount: 1,
			expectNames: []string{"Data Analysis Tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := manager.FindRelevantSkills(tt.query, tt.limit)

			if len(matches) != tt.expectCount {
				t.Errorf("FindRelevantSkills() count = %v, want %v", len(matches), tt.expectCount)
				return
			}

			for i, match := range matches {
				if i < len(tt.expectNames) && match.Skill.Name() != tt.expectNames[i] {
					t.Errorf("FindRelevantSkills() match[%v].Name = %v, want %v",
						i, match.Skill.Name(), tt.expectNames[i])
				}
				if match.Score <= 0 {
					t.Errorf("FindRelevantSkills() match[%v].Score = %v, should be > 0", i, match.Score)
				}
			}
		})
	}
}

func TestDefaultSkillManager_ListSkills(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skills := []Skill{
		&baseSkill{
			metadata: SkillMetadata{Name: "Skill 1", Description: "Desc 1"},
			instruct: "Instructions 1",
		},
		&baseSkill{
			metadata: SkillMetadata{Name: "Skill 2", Description: "Desc 2"},
			instruct: "Instructions 2",
		},
		&baseSkill{
			metadata: SkillMetadata{Name: "Skill 3", Description: "Desc 3"},
			instruct: "Instructions 3",
		},
	}

	for i, skill := range skills {
		loader.addSkill(string(rune(i)), skill)
	}

	manager.LoadSkills([]string{"/"})

	allSkills := manager.ListSkills()
	if len(allSkills) != 3 {
		t.Errorf("ListSkills() count = %v, want 3", len(allSkills))
	}

	names := make(map[string]bool)
	for _, skill := range allSkills {
		names[skill.Name()] = true
	}

	expectedNames := []string{"Skill 1", "Skill 2", "Skill 3"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("ListSkills() missing skill: %v", name)
		}
	}
}

func TestDefaultSkillManager_Refresh(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skill1 := &baseSkill{
		metadata: SkillMetadata{Name: "Skill 1", Description: "Desc 1"},
		instruct: "Instructions 1",
	}
	loader.addSkill("/skill1", skill1)

	manager.LoadSkills([]string{"/"})
	stats := manager.Stats()
	if stats.TotalSkills != 1 {
		t.Errorf("Initial load TotalSkills = %v, want 1", stats.TotalSkills)
	}

	skill2 := &baseSkill{
		metadata: SkillMetadata{Name: "Skill 2", Description: "Desc 2"},
		instruct: "Instructions 2",
	}
	skill3 := &baseSkill{
		metadata: SkillMetadata{Name: "Skill 3", Description: "Desc 3"},
		instruct: "Instructions 3",
	}
	loader.addSkill("/skill2", skill2)
	loader.addSkill("/skill3", skill3)

	err := manager.Refresh()
	if err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	stats = manager.Stats()
	if stats.TotalSkills != 3 {
		t.Errorf("After refresh TotalSkills = %v, want 3", stats.TotalSkills)
	}

	_, exists := manager.GetSkill("Skill 2")
	if !exists {
		t.Error("GetSkill() should find newly loaded skill after refresh")
	}
}

func TestDefaultSkillManager_ConcurrentAccess(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	for i := 0; i < 5; i++ {
		skill := &baseSkill{
			metadata: SkillMetadata{
				Name:        fmt.Sprintf("Skill %v", i),
				Description: fmt.Sprintf("Description %v", i),
			},
			instruct: fmt.Sprintf("Instructions %v", i),
		}
		loader.addSkill(string(rune(i)), skill)
	}

	manager.LoadSkills([]string{"/"})

	var wg sync.WaitGroup
	done := make(chan bool, 1)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			select {
			case <-done:
				return
			default:
				matches := manager.FindRelevantSkills(fmt.Sprintf("query %v", id), 3)
				_ = matches
			}
		}(i)
	}

	wg.Wait()
}

func TestDefaultSkillManager_Stats(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	stats := manager.Stats()
	if stats.TotalSkills != 0 {
		t.Errorf("Initial TotalSkills = %v, want 0", stats.TotalSkills)
	}

	skill := &baseSkill{
		metadata: SkillMetadata{Name: "Test Skill", Description: "Test"},
		instruct: "Test instructions",
	}
	loader.addSkill("/test", skill)

	paths := []string{"/test"}
	err := manager.LoadSkills(paths)
	if err != nil {
		t.Errorf("LoadSkills() error = %v", err)
	}

	stats = manager.Stats()
	if stats.TotalSkills != 1 {
		t.Errorf("After load TotalSkills = %v, want 1", stats.TotalSkills)
	}

	if len(stats.LoadedPaths) == 0 {
		t.Error("LoadedPaths should not be empty after loading")
	}

	manager.FindRelevantSkills("test query", 3)
	manager.FindRelevantSkills("another query", 3)

	stats = manager.Stats()
	if stats.CacheHitRate < 0 {
		t.Errorf("CacheHitRate = %v, should be >= 0", stats.CacheHitRate)
	}
}

func TestSkillsContext(t *testing.T) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	skill := &baseSkill{
		metadata: SkillMetadata{
			Name:        "Test Skill",
			Description: "Test description",
			Triggers:    []string{"test trigger"},
		},
		instruct: "Test instructions",
	}
	loader.addSkill("/test", skill)

	manager.LoadSkills([]string{"/"})

	ctx := NewSkillsContext(manager)

	if !ctx.Enabled {
		t.Error("SkillsContext should be enabled by default")
	}

	skills := ctx.GetRelevantSkillsForContext("test trigger")
	if len(skills) == 0 {
		t.Error("GetRelevantSkillsForContext() should find matching skills")
	}

	disabledCtx := ctx.WithEnabled(false)
	if disabledCtx.Enabled {
		t.Error("WithEnabled(false) should disable skills")
	}

	disabledSkills := disabledCtx.GetRelevantSkillsForContext("test trigger")
	if len(disabledSkills) != 0 {
		t.Error("Disabled context should not return any skills")
	}

	limitedCtx := ctx.WithMaxSkills(1)
	if limitedCtx.MaxSkills != 1 {
		t.Errorf("WithMaxSkills(1) should set MaxSkills to 1, got %v", limitedCtx.MaxSkills)
	}
}

func BenchmarkFindRelevantSkills(b *testing.B) {
	loader := newMockLoader()
	manager := NewSkillManager(loader)

	for i := 0; i < 100; i++ {
		skill := &baseSkill{
			metadata: SkillMetadata{
				Name:        fmt.Sprintf("Skill %v", i),
				Description: fmt.Sprintf("Description for skill %v", i),
				Keywords:    []string{fmt.Sprintf("keyword%v", i)},
			},
			instruct: fmt.Sprintf("Instructions for skill %v", i),
		}
		loader.addSkill(string(rune(i)), skill)
	}

	manager.LoadSkills([]string{"/"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.FindRelevantSkills(fmt.Sprintf("benchmark query %v", i%10), 5)
	}
}
