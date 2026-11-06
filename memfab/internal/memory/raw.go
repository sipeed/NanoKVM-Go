package memory

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadRaw(memoryDir, name string) (string, string, error) {
	path, err := MemoryPath(memoryDir, name)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return path, string(data), nil
}

func StripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return trimmed
	}
	first := strings.TrimSpace(lines[0])
	if first != "```" && first != "```markdown" && first != "```md" {
		return trimmed
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}

func SetFrontmatterScalar(text, key, value string) (string, error) {
	front, body, err := splitFrontmatter(text)
	if err != nil {
		return "", err
	}
	lines := strings.Split(front, "\n")
	written := false
	for i, line := range lines {
		name, _, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(name) == key {
			lines[i] = key + ": " + value
			written = true
			break
		}
	}
	if !written {
		lines = append(lines, key+": "+value)
	}
	return "---\n" + strings.Join(lines, "\n") + "\n---" + body, nil
}

func SetFrontmatterBool(text, key string, value bool) (string, error) {
	if value {
		return SetFrontmatterScalar(text, key, "true")
	}
	return SetFrontmatterScalar(text, key, "false")
}

func SetFrontmatterInt(text, key string, value int) (string, error) {
	return SetFrontmatterScalar(text, key, strconv.Itoa(value))
}

func splitFrontmatter(text string) (string, string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", "", fmt.Errorf("markdown must start with frontmatter")
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return "", "", fmt.Errorf("frontmatter closing marker missing")
	}
	front := text[4 : 4+end]
	bodyStart := 4 + end + len("\n---")
	body := text[bodyStart:]
	return front, body, nil
}
