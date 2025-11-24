package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	SkillPaths []string        `json:"skill_paths" yaml:"skill_paths"`
	Loader     LoaderConfig    `json:"loader" yaml:"loader"`
	Matching   MatchingConfig  `json:"matching" yaml:"matching"`
	Injection  InjectionConfig `json:"injection" yaml:"injection"`
	Cache      CacheConfig     `json:"cache" yaml:"cache"`
	Logging    LoggingConfig   `json:"logging" yaml:"logging"`
}

type LoaderConfig struct {
	Recursive bool   `json:"recursive" yaml:"recursive"`
	SkillFile string `json:"skill_file" yaml:"skill_file"`
	MaxDepth  int    `json:"max_depth" yaml:"max_depth"`
}

type MatchingConfig struct {
	MaxSkills int     `json:"max_skills" yaml:"max_skills"`
	MinScore  float64 `json:"min_score" yaml:"min_score"`
	Timeout   int     `json:"timeout" yaml:"timeout"`
}

type InjectionConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	AsSystem       bool     `json:"as_system" yaml:"as_system"`
	AsUser         bool     `json:"as_user" yaml:"as_user"`
	Prefix         string   `json:"prefix" yaml:"prefix"`
	Suffix         string   `json:"suffix" yaml:"suffix"`
	ExcludedModels []string `json:"excluded_models" yaml:"excluded_models"`
}

type CacheConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	TTL     int  `json:"ttl" yaml:"ttl"`
	MaxSize int  `json:"max_size" yaml:"max_size"`
}

type LoggingConfig struct {
	Level    string `json:"level" yaml:"level"`
	Skills   bool   `json:"skills" yaml:"skills"`
	Matching bool   `json:"matching" yaml:"matching"`
	Errors   bool   `json:"errors" yaml:"errors"`
}

func DefaultConfig() *Config {
	return &Config{
		SkillPaths: []string{
			"./skills",
			"~/.claude/skills",
			"/usr/local/share/claude/skills",
		},
		Loader: LoaderConfig{
			Recursive: true,
			SkillFile: "SKILL.md",
			MaxDepth:  3,
		},
		Matching: MatchingConfig{
			MaxSkills: 3,
			MinScore:  0.3,
			Timeout:   5,
		},
		Injection: InjectionConfig{
			Enabled:  true,
			AsSystem: true,
			AsUser:   false,
			Prefix:   "\n\n--- Relevant Skills ---\n",
			Suffix:   "\n--- End of Skills ---\n",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     300,
			MaxSize: 1000,
		},
		Logging: LoggingConfig{
			Level:    "info",
			Skills:   true,
			Matching: false,
			Errors:   true,
		},
	}
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := DefaultConfig()

	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		if err := parseJSON(content, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %v", err)
		}
	case ".yaml", ".yml":
		if err := parseYAML(string(content), config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	config.processEnvVars()

	return config, nil
}

func (c *Config) SaveConfig(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	var content []byte
	var err error

	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		content, err = formatJSON(c)
		if err != nil {
			return fmt.Errorf("failed to format JSON config: %v", err)
		}
	case ".yaml", ".yml":
		content, err = formatYAML(c)
		if err != nil {
			return fmt.Errorf("failed to format YAML config: %v", err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if len(c.SkillPaths) == 0 {
		return fmt.Errorf("at least one skill path must be specified")
	}

	if c.Matching.MaxSkills <= 0 {
		return fmt.Errorf("max_skills must be positive")
	}

	if c.Matching.MinScore < 0 || c.Matching.MinScore > 1 {
		return fmt.Errorf("min_score must be between 0 and 1")
	}

	if c.Loader.MaxDepth <= 0 {
		return fmt.Errorf("loader max_depth must be positive")
	}

	if c.Loader.SkillFile == "" {
		return fmt.Errorf("loader skill_file cannot be empty")
	}

	return nil
}

func (c *Config) GetEffectivePaths() []string {
	var paths []string
	homeDir, _ := os.UserHomeDir()

	for _, path := range c.SkillPaths {
		expandedPath := path

		if strings.HasPrefix(path, "~/") {
			expandedPath = filepath.Join(homeDir, path[2:])
		}

		if _, err := os.Stat(expandedPath); err == nil {
			paths = append(paths, expandedPath)
		}
	}

	return paths
}

func (c *Config) processEnvVars() {
	for i, path := range c.SkillPaths {
		c.SkillPaths[i] = os.ExpandEnv(path)
	}

	c.Loader.SkillFile = os.ExpandEnv(c.Loader.SkillFile)
	c.Injection.Prefix = os.ExpandEnv(c.Injection.Prefix)
	c.Injection.Suffix = os.ExpandEnv(c.Injection.Suffix)
	c.Logging.Level = os.ExpandEnv(c.Logging.Level)
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	if len(other.SkillPaths) > 0 {
		c.SkillPaths = append(c.SkillPaths, other.SkillPaths...)
	}

	if other.Loader.Recursive {
		c.Loader.Recursive = other.Loader.Recursive
	}
	if other.Loader.SkillFile != "" {
		c.Loader.SkillFile = other.Loader.SkillFile
	}
	if other.Loader.MaxDepth > 0 {
		c.Loader.MaxDepth = other.Loader.MaxDepth
	}

	if other.Matching.MaxSkills > 0 {
		c.Matching.MaxSkills = other.Matching.MaxSkills
	}
	if other.Matching.MinScore > 0 {
		c.Matching.MinScore = other.Matching.MinScore
	}
	if other.Matching.Timeout > 0 {
		c.Matching.Timeout = other.Matching.Timeout
	}

	if other.Injection.Enabled {
		c.Injection.Enabled = other.Injection.Enabled
	}
	if other.Injection.AsSystem {
		c.Injection.AsSystem = other.Injection.AsSystem
	}
	if other.Injection.AsUser {
		c.Injection.AsUser = other.Injection.AsUser
	}
	if other.Injection.Prefix != "" {
		c.Injection.Prefix = other.Injection.Prefix
	}
	if other.Injection.Suffix != "" {
		c.Injection.Suffix = other.Injection.Suffix
	}
	if len(other.Injection.ExcludedModels) > 0 {
		c.Injection.ExcludedModels = append(c.Injection.ExcludedModels, other.Injection.ExcludedModels...)
	}

	if other.Cache.Enabled {
		c.Cache.Enabled = other.Cache.Enabled
	}
	if other.Cache.TTL > 0 {
		c.Cache.TTL = other.Cache.TTL
	}
	if other.Cache.MaxSize > 0 {
		c.Cache.MaxSize = other.Cache.MaxSize
	}

	if other.Logging.Level != "" {
		c.Logging.Level = other.Logging.Level
	}
	if other.Logging.Skills {
		c.Logging.Skills = other.Logging.Skills
	}
	if other.Logging.Matching {
		c.Logging.Matching = other.Logging.Matching
	}
	if other.Logging.Errors {
		c.Logging.Errors = other.Logging.Errors
	}
}

func (c *Config) LoadFromEnv() {
	if path := os.Getenv("CLAUDE_SKILL_PATHS"); path != "" {
		c.SkillPaths = strings.Split(path, ":")
	}

	if recursive := os.Getenv("CLAUDE_SKILL_RECURSIVE"); recursive != "" {
		c.Loader.Recursive = recursive == "true"
	}

	if skillFile := os.Getenv("CLAUDE_SKILL_FILE"); skillFile != "" {
		c.Loader.SkillFile = skillFile
	}

	if maxDepth := os.Getenv("CLAUDE_SKILL_MAX_DEPTH"); maxDepth != "" {
		if depth, err := strconv.Atoi(maxDepth); err == nil {
			c.Loader.MaxDepth = depth
		}
	}

	if maxSkills := os.Getenv("CLAUDE_SKILL_MAX_SKILLS"); maxSkills != "" {
		if skills, err := strconv.Atoi(maxSkills); err == nil {
			c.Matching.MaxSkills = skills
		}
	}

	if minScore := os.Getenv("CLAUDE_SKILL_MIN_SCORE"); minScore != "" {
		if score, err := strconv.ParseFloat(minScore, 64); err == nil {
			c.Matching.MinScore = score
		}
	}

	if enabled := os.Getenv("CLAUDE_SKILL_ENABLED"); enabled != "" {
		c.Injection.Enabled = enabled == "true"
	}

	if level := os.Getenv("CLAUDE_SKILL_LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
}

func GetConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	wd, _ := os.Getwd()

	return []string{
		filepath.Join(wd, "claude-skills.json"),
		filepath.Join(wd, "claude-skills.yaml"),
		filepath.Join(wd, "claude-skills.yml"),
		filepath.Join(homeDir, ".claude", "skills.json"),
		filepath.Join(homeDir, ".claude", "skills.yaml"),
		filepath.Join(homeDir, ".claude", "skills.yml"),
		"/etc/claude/skills.json",
		"/etc/claude/skills.yaml",
		"/etc/claude/skills.yml",
	}
}

func findConfig() string {
	for _, path := range GetConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func LoadDefaultConfig() (*Config, error) {
	if configPath := findConfig(); configPath != "" {
		return LoadConfig(configPath)
	}
	return DefaultConfig(), nil
}
