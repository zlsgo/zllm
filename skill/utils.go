package skill

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func parseYAML(content string, target interface{}) error {
	return yaml.Unmarshal([]byte(content), target)
}

type SkillValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e SkillValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

func ValidateSkill(metadata SkillMetadata) []SkillValidationError {
	var errors []SkillValidationError

	if strings.TrimSpace(metadata.Name) == "" {
		errors = append(errors, SkillValidationError{
			Field:   "name",
			Message: "skill name cannot be empty",
		})
	}

	if len(metadata.Name) > 100 {
		errors = append(errors, SkillValidationError{
			Field:   "name",
			Value:   metadata.Name,
			Message: "skill name too long (max 100 characters)",
		})
	}

	if strings.TrimSpace(metadata.Description) == "" {
		errors = append(errors, SkillValidationError{
			Field:   "description",
			Message: "skill description cannot be empty",
		})
	}

	if len(metadata.Description) > 500 {
		errors = append(errors, SkillValidationError{
			Field:   "description",
			Message: "skill description too long (max 500 characters)",
		})
	}

	hasValidTrigger := false
	for i, trigger := range metadata.Triggers {
		trimmed := strings.TrimSpace(trigger)
		if trimmed != "" && len(trigger) <= 200 {
			hasValidTrigger = true
		}
		if trimmed == "" {
			errors = append(errors, SkillValidationError{
				Field:   fmt.Sprintf("triggers[%d]", i),
				Message: "trigger cannot be empty",
			})
		}
		if len(trigger) > 200 {
			errors = append(errors, SkillValidationError{
				Field:   fmt.Sprintf("triggers[%d]", i),
				Value:   trigger,
				Message: "trigger too long (max 200 characters)",
			})
		}
	}

	hasValidKeyword := false
	for i, keyword := range metadata.Keywords {
		if strings.TrimSpace(keyword) == "" {
			errors = append(errors, SkillValidationError{
				Field:   fmt.Sprintf("keywords[%d]", i),
				Message: "keyword cannot be empty",
			})
		} else if len(keyword) <= 50 {
			hasValidKeyword = true
		}
		if len(keyword) > 50 {
			errors = append(errors, SkillValidationError{
				Field:   fmt.Sprintf("keywords[%d]", i),
				Value:   keyword,
				Message: "keyword too long (max 50 characters)",
			})
		}
	}

	if (len(metadata.Triggers) > 0 || len(metadata.Keywords) > 0) && !hasValidTrigger {
		alreadyHasTriggerError := false
		for _, e := range errors {
			if e.Field == "triggers" {
				alreadyHasTriggerError = true
				break
			}
		}
		if !alreadyHasTriggerError {
			errors = append(errors, SkillValidationError{
				Field:   "triggers",
				Message: "at least one valid trigger is required",
			})
		}
	}
	if (len(metadata.Triggers) > 0 || len(metadata.Keywords) > 0) && !hasValidKeyword {
		errors = append(errors, SkillValidationError{
			Field:   "keywords",
			Message: "at least one valid keyword is required",
		})
	}

	if len(metadata.Version) == 0 {
		errors = append(errors, SkillValidationError{
			Field:   "version",
			Message: "version cannot be empty",
		})
	}

	return errors
}

type SkillFilter struct {
	Categories []string
	Tags       []string
	Author     string
	MinVersion string
}

func (f SkillFilter) Match(skill Skill) bool {
	metadata := skill.Metadata()

	if len(f.Categories) > 0 {
		found := false
		for _, category := range f.Categories {
			if strings.EqualFold(metadata.Category, category) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.Tags) > 0 {
		found := false
		for _, requiredTag := range f.Tags {
			for _, skillTag := range metadata.Tags {
				if strings.EqualFold(skillTag, requiredTag) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	if f.Author != "" && !strings.EqualFold(metadata.Author, f.Author) {
		return false
	}

	return true
}

type SkillSorter struct {
	By    string
	Order string
}

func SortSkills(skills []Skill, sorter SkillSorter) {
	switch sorter.By {
	case "name":
		sortByName(skills, sorter.Order)
	case "date":
		sortByDate(skills, sorter.Order)
	default:
		sortByName(skills, "asc")
	}
}

func sortByName(skills []Skill, order string) {
	for i := 0; i < len(skills)-1; i++ {
		for j := i + 1; j < len(skills); j++ {
			shouldSwap := false
			if order == "desc" {
				shouldSwap = skills[i].Name() < skills[j].Name()
			} else {
				shouldSwap = skills[i].Name() > skills[j].Name()
			}
			if shouldSwap {
				skills[i], skills[j] = skills[j], skills[i]
			}
		}
	}
}

func sortByDate(skills []Skill, order string) {
	for i := 0; i < len(skills)-1; i++ {
		for j := i + 1; j < len(skills); j++ {
			shouldSwap := false
			dateI := skills[i].Metadata().UpdatedAt
			dateJ := skills[j].Metadata().UpdatedAt
			if order == "desc" {
				shouldSwap = dateI.Before(dateJ)
			} else {
				shouldSwap = dateI.After(dateJ)
			}
			if shouldSwap {
				skills[i], skills[j] = skills[j], skills[i]
			}
		}
	}
}
