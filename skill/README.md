# Skills - Claude Skills 支持

这个包为 zllm 项目添加了 Claude Skills 支持，让 agent 能够自动发现、匹配和使用相关技能。

## 特性

- 🔍 **自动技能发现** - 从文件系统自动发现和加载技能
- 🎯 **智能匹配** - 基于查询内容智能匹配相关技能
- 🔄 **热更新** - 支持技能的热更新和缓存
- ⚙️ **灵活配置** - 支持多种配置选项和环境变量
- 🔌 **无缝集成** - 与现有 agent 系统无缝集成

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
    "github.com/zlsgo/zllm/skill"
)

func main() {
    // 创建基础 agent
    baseAgent := agent.NewAnthropic(agent.BaseConfig{
        Model:   "claude-3-sonnet-20240229",
        APIKey:  "your-api-key",
    })

    // 创建技能管理器
    loader := skill.NewSkillLoader()
    manager := skill.NewSkillManager(loader)

    // 加载技能
    err := manager.LoadSkills([]string{"./skills"})
    if err != nil {
        panic(err)
    }

    // 创建支持技能的代理
    skillsAgent := skill.NewSkillsProvider(baseAgent, manager, skill.DefaultSkillsConfig())

    // 使用代理
    messages := message.NewMessages()
    messages.AppendUser("请帮我审查这段代码的质量")

    response, err := skillsAgent.Generate(context.Background(), messages)
    if err != nil {
        panic(err)
    }

    fmt.Println(string(response.Content))
}
```

### 2. 技能格式

技能是一个包含 `SKILL.md` 文件的文件夹，格式如下：

```markdown
---
name: Code Review Assistant
description: 专业的代码审查助手
version: 1.0.0
author: Your Name
category: Development
tags:
  - code-review
  - development
keywords:
  - 代码审查
  - code review
triggers:
  - "代码审查"
  - "code review"
  - "review this code"
---

# 技能说明

这里是技能的详细说明和使用指南...

```

### 3. 配置选项

```go
config := skill.DefaultSkillsConfig()
config.Enabled = true
config.MaxSkills = 3
config.MinScore = 0.3
config.InjectAsSystem = true

skillsAgent := skill.NewSkillsProvider(baseAgent, manager, config)
```

## API 文档

### 核心接口

#### Skill

```go
type Skill interface {
    Name() string
    Description() string
    Metadata() SkillMetadata
    Instructions() string
    Match(query string) float64
    Resources() []string
}
```

#### SkillManager

```go
type SkillManager interface {
    LoadSkills(paths []string) error
    FindRelevantSkills(query string, limit int) []SkillMatch
    GetSkill(name string) (Skill, bool)
    ListSkills() []Skill
    Refresh() error
    Stats() ManagerStats
}
```

#### SkillsProvider

```go
type SkillsProvider struct {
    agent   agent.LLMAgent
    manager SkillManager
    config  SkillsConfig
}
```

### 配置结构

#### SkillMetadata

```go
type SkillMetadata struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Version     string            `yaml:"version"`
    Author      string            `yaml:"author,omitempty"`
    Tags        []string          `yaml:"tags,omitempty"`
    Category    string            `yaml:"category,omitempty"`
    Keywords    []string          `yaml:"keywords,omitempty"`
    Triggers    []string          `yaml:"triggers,omitempty"`
    Required    []string          `yaml:"required,omitempty"`
    Optional    []string          `yaml:"optional,omitempty"`
    CreatedAt   time.Time         `yaml:"created_at,omitempty"`
    UpdatedAt   time.Time         `yaml:"updated_at,omitempty"`
    Config      map[string]string `yaml:"config,omitempty"`
}
```

#### SkillsConfig

```go
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
```

## 技能匹配算法

技能匹配基于以下因素：

1. **显式触发器** - 精确匹配触发词
2. **名称匹配** - 技能名称与查询的相似度
3. **描述匹配** - 技能描述与查询的相关性
4. **关键词匹配** - 关键词权重计算
5. **标签匹配** - 标签相关性评分
6. **分类匹配** - 分类相关性评分

匹配分数范围：0.0 - 1.0，默认最小匹配分数为 0.3。

## 配置文件

支持 JSON 和 YAML 格式的配置文件：

```yaml
# claude-skills.yaml
skill_paths:
  - "./skills"
  - "~/.claude/skills"

loader:
  recursive: true
  skill_file: "SKILL.md"
  max_depth: 3

matching:
  max_skills: 3
  min_score: 0.3
  timeout: 5

injection:
  enabled: true
  as_system: true
  as_user: false
  prefix: "\n\n--- Relevant Skills ---\n"
  suffix: "\n--- End of Skills ---\n"

cache:
  enabled: true
  ttl: 300
  max_size: 1000

logging:
  level: "info"
  skills: true
  matching: false
  errors: true
```

## 环境变量

支持以下环境变量配置：

- `CLAUDE_SKILL_PATHS` - 技能搜索路径（冒号分隔）
- `CLAUDE_SKILL_RECURSIVE` - 是否递归搜索
- `CLAUDE_SKILL_FILE` - 技能文件名
- `CLAUDE_SKILL_MAX_DEPTH` - 最大搜索深度
- `CLAUDE_SKILL_MAX_SKILLS` - 最大匹配技能数
- `CLAUDE_SKILL_MIN_SCORE` - 最小匹配分数
- `CLAUDE_SKILL_ENABLED` - 是否启用技能
- `CLAUDE_SKILL_LOG_LEVEL` - 日志级别
