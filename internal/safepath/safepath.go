package safepath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanAbs(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}
	return filepath.Clean(abs), nil
}

func Within(root, candidate string) bool {
	r, err := CleanAbs(root)
	if err != nil {
		return false
	}
	c, err := CleanAbs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(r, c)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func Join(root string, parts ...string) (string, error) {
	r, err := CleanAbs(root)
	if err != nil {
		return "", err
	}
	for _, part := range parts {
		if filepath.IsAbs(part) {
			return "", fmt.Errorf("absolute path component is forbidden: %q", part)
		}
	}
	items := append([]string{r}, parts...)
	candidate := filepath.Join(items...)
	if !Within(r, candidate) {
		return "", fmt.Errorf("path escapes owned root: %q", candidate)
	}
	if err := rejectSymlinkComponents(r, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func rejectSymlinkComponents(root, candidate string) error {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	current := root
	if info, statErr := os.Lstat(current); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("owned root is a symlink: %q", current)
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return nil
			}
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path component is forbidden: %q", current)
		}
	}
	return nil
}

func RejectGitTree(path string) error {
	current, err := CleanAbs(path)
	if err != nil {
		return err
	}
	for {
		if _, statErr := os.Lstat(filepath.Join(current, ".git")); statErr == nil {
			return fmt.Errorf("data root may not be inside a Git worktree: %q", path)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
