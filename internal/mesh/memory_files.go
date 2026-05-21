package mesh

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const memoryFileName = "memory.md"

func LocalMemoryContext(root string, scope Scope, maxBytesPerNode int) ([]MemoryNode, error) {
	root = expandLocalPath(root)
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}

	nodes := make([]MemoryNode, 0, len(scope.Path()))
	for _, pathScope := range scope.Path() {
		path := LocalMemoryPath(root, pathScope)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nodes, err
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			continue
		}
		text = trimStringBytes(text, maxBytesPerNode)
		nodes = append(nodes, MemoryNode{
			Scope:     pathScope,
			Title:     pathScope.Key(),
			Summary:   text,
			UpdatedAt: time.Now(),
		})
	}
	return nodes, nil
}

func LocalMemoryPath(root string, scope Scope) string {
	root = expandLocalPath(root)
	parts := []string{root}
	if scope.Kind() == ScopeGlobal {
		return filepath.Join(append(parts, "global.md")...)
	}

	if strings.TrimSpace(scope.Host) != "" {
		parts = append(parts, "hosts", memoryPathPart(scope.Host))
	}
	if strings.TrimSpace(scope.Session) != "" {
		parts = append(parts, "sessions", memoryPathPart(scope.Session))
	}
	if strings.TrimSpace(scope.Window) != "" {
		parts = append(parts, "windows", memoryPathPart(scope.Window))
	}
	if strings.TrimSpace(scope.Pane) != "" {
		parts = append(parts, "panes", memoryPathPart(scope.Pane))
	}
	return filepath.Join(append(parts, memoryFileName)...)
}

func expandLocalPath(path string) string {
	path = strings.TrimSpace(os.ExpandEnv(path))
	if path == "" {
		return ""
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func memoryPathPart(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func trimStringBytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	for index := range value {
		if index > maxBytes {
			return strings.TrimSpace(value[:index])
		}
	}
	return value
}
