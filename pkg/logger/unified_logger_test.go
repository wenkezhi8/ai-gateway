package logger

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoStandaloneLogrusNewOutsidePackageLogger(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))

	var found []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || base == ".idea" || base == ".vscode" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".worktrees/") {
			return nil
		}

		if rel == "pkg/logger/logger.go" {
			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		if strings.Contains(string(b), "logrus.New()") {
			found = append(found, rel)
		}
		return nil
	})

	assert.NoError(t, err)
	sort.Strings(found)
	assert.Empty(t, found, "found standalone logrus.New() usages: %v", found)
}
