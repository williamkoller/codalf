package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Content     string
}

var cachedSkills = make(map[string]Skill)

func LoadSkills(skillsDir string) (map[string]Skill, error) {
	if len(cachedSkills) > 0 {
		return cachedSkills, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skillFile := filepath.Join(skillPath, "SKILL.md")

		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		content := string(data)
		name := entry.Name()
		description := extractDescription(content)

		cachedSkills[name] = Skill{
			Name:        name,
			Description: description,
			Content:     content,
		}
	}

	return cachedSkills, nil
}

func GetSkillForLanguage(lang string) []Skill {
	var result []Skill
	for _, skill := range cachedSkills {
		if strings.Contains(skill.Name, lang) || strings.Contains(skill.Description, lang) {
			result = append(result, skill)
		}
	}
	return result
}

func GetSkillByName(name string) (Skill, bool) {
	skill, ok := cachedSkills[name]
	return skill, ok
}

func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			desc := strings.TrimPrefix(line, "description:")
			desc = strings.TrimSpace(desc)
			desc = strings.Trim(desc, "<>")
			return strings.TrimSpace(desc)
		}
	}
	return ""
}

func BuildSkillContext(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	langMap := map[string][]string{
		".go":   {"go-"},
		".ts":   {"typescript"},
		".tsx":  {"typescript", "react"},
		".js":   {"javascript", "react"},
		".jsx":  {"javascript", "react"},
		".py":   {"python"},
		".rs":   {"rust"},
		".java": {"java"},
		".kt":   {"kotlin"},
	}

	var skills []string
	for lang, prefixes := range langMap {
		if ext == lang {
			for _, prefix := range prefixes {
				for name, skill := range cachedSkills {
					if strings.HasPrefix(name, prefix) {
						skills = append(skills, skill.Content)
					}
				}
			}
			break
		}
	}

	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- Reference Skills ---\n\n")
	for _, s := range skills {
		sb.WriteString(s)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
