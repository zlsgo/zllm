package skill

import (
	"fmt"
	"sort"
	"sync"

	"github.com/zlsgo/zllm/runtime"
)

type SkillManager interface {
	LoadSkills(paths []string) error
	FindRelevantSkills(query string, limit int) []SkillMatch
	GetSkill(name string) (Skill, bool)
	ListSkills() []Skill
	Refresh() error
	Stats() ManagerStats
}

type ManagerStats struct {
	TotalSkills  int
	LoadedPaths  []string
	LastRefresh  string
	CacheHitRate float64
	LoadErrors   []string
}

type DefaultSkillManager struct {
	mu     sync.RWMutex
	skills map[string]Skill
	loader SkillLoader
	cache  map[string][]SkillMatch
	paths  []string
	stats  ManagerStats
}

func NewSkillManager(loader SkillLoader) SkillManager {
	return &DefaultSkillManager{
		skills: make(map[string]Skill),
		loader: loader,
		cache:  make(map[string][]SkillMatch),
		stats: ManagerStats{
			LoadedPaths: make([]string, 0),
			LoadErrors:  make([]string, 0),
		},
	}
}

func (m *DefaultSkillManager) LoadSkills(paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.Log("Loading skills from paths:", paths)

	m.stats.LoadedPaths = append(m.stats.LoadedPaths, paths...)

	skills, errors := m.loader.LoadFromPaths(paths)

	m.skills = make(map[string]Skill)

	for _, skill := range skills {
		m.skills[skill.Name()] = skill
		runtime.Log("Loaded skill:", skill.Name())
	}

	m.stats.TotalSkills = len(m.skills)
	m.stats.LoadErrors = errors

	runtime.Log("Skills loading completed:", len(m.skills), "skills loaded,", len(errors), "errors")

	if len(errors) > 0 {
		runtime.Log("Skill loading errors:", errors)
		return fmt.Errorf("loaded %d skills with %d errors", len(m.skills), len(errors))
	}

	return nil
}

func (m *DefaultSkillManager) FindRelevantSkills(query string, limit int) []SkillMatch {
	m.mu.RLock()
	if cached, exists := m.cache[query]; exists {
		m.mu.RUnlock()
		runtime.Log("Cache hit for query:", query, "found", len(cached), "matches")
		return cached[:min(limit, len(cached))]
	}

	m.mu.RUnlock()

	runtime.Log("Finding relevant skills for query:", query, "limit:", limit)

	m.mu.RLock()
	skillsCopy := make([]Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		skillsCopy = append(skillsCopy, skill)
	}
	m.mu.RUnlock()

	var matches []SkillMatch
	for _, skill := range skillsCopy {
		score := skill.Match(query)
		if score > 0 {
			matches = append(matches, SkillMatch{
				Skill:  skill,
				Score:  score,
				Reason: m.getMatchReason(skill, query),
			})
			runtime.Log("Skill match found:", skill.Name(), "score:", score)
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Skill.Name() < matches[j].Skill.Name()
		}
		return matches[i].Score > matches[j].Score
	})

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	m.mu.Lock()
	m.cache[query] = matches
	m.mu.Unlock()

	runtime.Log("Query processed:", query, "found", len(matches), "relevant skills")
	return matches
}

func (m *DefaultSkillManager) GetSkill(name string) (Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skill, exists := m.skills[name]
	return skill, exists
}

func (m *DefaultSkillManager) ListSkills() []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skills := make([]Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}

	return skills
}

func (m *DefaultSkillManager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache = make(map[string][]SkillMatch)

	skills, errors := m.loader.LoadFromPaths(m.paths)

	m.skills = make(map[string]Skill)
	for _, skill := range skills {
		m.skills[skill.Name()] = skill
	}

	m.stats.TotalSkills = len(m.skills)
	m.stats.LoadErrors = errors

	return nil
}

func (m *DefaultSkillManager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalCache := len(m.cache)
	hitRate := 0.0
	if totalCache > 0 {
		hitRate = float64(totalCache) / float64(m.stats.TotalSkills+totalCache)
	}

	stats := m.stats
	stats.CacheHitRate = hitRate

	return stats
}

func (m *DefaultSkillManager) getMatchReason(skill Skill, query string) string {
	metadata := skill.Metadata()

	if metadata.IsTriggered(query) {
		return "Explicit trigger match"
	}

	if contains(toLower(metadata.Name), toLower(query)) {
		return "Name match"
	}

	for _, keyword := range metadata.Keywords {
		if contains(toLower(query), toLower(keyword)) {
			return fmt.Sprintf("Keyword match: %s", keyword)
		}
	}

	for _, tag := range metadata.Tags {
		if contains(toLower(query), toLower(tag)) {
			return fmt.Sprintf("Tag match: %s", tag)
		}
	}

	return "General relevance"
}

type SkillsContext struct {
	Manager   SkillManager
	Enabled   bool
	MaxSkills int
}

func NewSkillsContext(manager SkillManager) *SkillsContext {
	return &SkillsContext{
		Manager:   manager,
		Enabled:   true,
		MaxSkills: 3,
	}
}

func (ctx *SkillsContext) GetRelevantSkillsForContext(context string) []SkillMatch {
	if !ctx.Enabled || ctx.Manager == nil {
		return nil
	}

	return ctx.Manager.FindRelevantSkills(context, ctx.MaxSkills)
}

func (ctx *SkillsContext) WithMaxSkills(max int) *SkillsContext {
	ctx.MaxSkills = max
	return ctx
}

func (ctx *SkillsContext) WithEnabled(enabled bool) *SkillsContext {
	ctx.Enabled = enabled
	return ctx
}
