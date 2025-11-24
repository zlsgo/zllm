package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SkillLoader interface {
	LoadFromPaths(paths []string) ([]Skill, []string)
	LoadFromPath(path string) ([]Skill, []string)
	LoadSkill(dirPath string) (Skill, error)
}

type DefaultSkillLoader struct {
	recursive bool
	skillFile string
	maxDepth  int
}

func NewSkillLoader(options ...func(*DefaultSkillLoader)) SkillLoader {
	loader := &DefaultSkillLoader{
		recursive: true,
		skillFile: "SKILL.md",
		maxDepth:  3,
	}

	for _, opt := range options {
		opt(loader)
	}

	return loader
}

func WithRecursive(recursive bool) func(*DefaultSkillLoader) {
	return func(l *DefaultSkillLoader) {
		l.recursive = recursive
	}
}

func WithSkillFile(filename string) func(*DefaultSkillLoader) {
	return func(l *DefaultSkillLoader) {
		l.skillFile = filename
	}
}

func WithMaxDepth(depth int) func(*DefaultSkillLoader) {
	return func(l *DefaultSkillLoader) {
		l.maxDepth = depth
	}
}

func (l *DefaultSkillLoader) LoadFromPaths(paths []string) ([]Skill, []string) {
	var allSkills []Skill
	var allErrors []string

	for _, path := range paths {
		skills, errors := l.LoadFromPath(path)
		allSkills = append(allSkills, skills...)
		allErrors = append(allErrors, errors...)
	}

	return allSkills, allErrors
}

func (l *DefaultSkillLoader) LoadFromPath(path string) ([]Skill, []string) {
	var skills []Skill
	var errors []string

	if _, err := os.Stat(path); err != nil {
		return nil, []string{fmt.Sprintf("path not found: %s", path)}
	}

	if l.recursive {
		err := filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
			if err != nil {
				errors = append(errors, fmt.Sprintf("error accessing %s: %v", currentPath, err))
				return nil
			}

			depth := strings.Count(strings.TrimPrefix(currentPath, path), string(os.PathSeparator))
			if depth > l.maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if d.IsDir() {
				if skill, err := l.LoadSkill(currentPath); err == nil && skill != nil {
					skills = append(skills, skill)
				} else if err != nil && !os.IsNotExist(err) {
					errors = append(errors, fmt.Sprintf("failed to load skill from %s: %v", currentPath, err))
				}
			}

			return nil
		})

		if err != nil {
			errors = append(errors, fmt.Sprintf("error walking path %s: %v", path, err))
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, []string{fmt.Sprintf("error reading directory %s: %v", path, err)}
		}

		for _, entry := range entries {
			if entry.IsDir() {
				skillPath := filepath.Join(path, entry.Name())
				if skill, err := l.LoadSkill(skillPath); err == nil && skill != nil {
					skills = append(skills, skill)
				} else if err != nil && !os.IsNotExist(err) {
					errors = append(errors, fmt.Sprintf("failed to load skill from %s: %v", skillPath, err))
				}
			}
		}
	}

	return skills, errors
}

func (l *DefaultSkillLoader) LoadSkill(dirPath string) (Skill, error) {
	skillFilePath := filepath.Join(dirPath, l.skillFile)
	if _, err := os.Stat(skillFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("skill file not found: %s", skillFilePath)
	}

	content, err := os.ReadFile(skillFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %v", err)
	}

	metadata, instructions, err := parseSkillContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill content: %v", err)
	}

	if metadata.Name == "" {
		metadata.Name = filepath.Base(dirPath)
	}

	resources, err := l.discoverResources(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %v", err)
	}

	skill := &baseSkill{
		metadata:  metadata,
		instruct:  instructions,
		resources: resources,
	}

	return skill, nil
}

func (l *DefaultSkillLoader) discoverResources(dirPath string) ([]string, error) {
	var resources []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == l.skillFile || strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		validExts := map[string]bool{
			".txt":  true,
			".md":   true,
			".json": true,
			".yaml": true,
			".yml":  true,
			".csv":  true,
			".html": true,
			".xml":  true,
			".go":   true,
			".py":   true,
			".js":   true,
			".ts":   true,
		}

		if validExts[ext] {
			resources = append(resources, filepath.Join(dirPath, name))
		}
	}

	return resources, nil
}

func parseSkillContent(content string) (SkillMetadata, string, error) {
	var metadata SkillMetadata
	var instructions string

	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			yamlContent := strings.TrimSpace(parts[1])
			instructions = strings.TrimSpace(parts[2])

			if err := parseYAML(yamlContent, &metadata); err != nil {
				return SkillMetadata{}, "", fmt.Errorf("failed to parse YAML frontmatter: %v", err)
			}
		} else {
			instructions = strings.TrimSpace(content)
			metadata.Description = truncateString(instructions, 100)
		}
	} else {
		instructions = strings.TrimSpace(content)

		lines := strings.Split(instructions, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				metadata.Name = truncateString(line, 50)
				break
			}
		}

		if metadata.Name == "" {
			metadata.Name = "Untitled Skill"
		}
		metadata.Description = truncateString(instructions, 100)
	}

	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}
	metadata.UpdatedAt = time.Now()

	return metadata, instructions, nil
}

type SkillCache struct {
	skills map[string]Skill
	files  map[string]SkillFile
}

func NewSkillCache() *SkillCache {
	return &SkillCache{
		skills: make(map[string]Skill),
		files:  make(map[string]SkillFile),
	}
}

func (c *SkillCache) Get(path string) (Skill, bool) {
	skill, exists := c.skills[path]
	return skill, exists
}

func (c *SkillCache) Set(path string, skill Skill, file SkillFile) {
	c.skills[path] = skill
	c.files[path] = file
}

func (c *SkillCache) IsExpired(path string) bool {
	file, exists := c.files[path]
	if !exists {
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		return true
	}

	return info.ModTime().After(file.Modified)
}

func (c *SkillCache) Clear() {
	c.skills = make(map[string]Skill)
	c.files = make(map[string]SkillFile)
}
