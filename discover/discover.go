// Package discover expands command line paths into the skill files to scan.
//
// It knows nothing about skills beyond how they are named on disk: a path that
// names a file is taken as given, and a directory is walked for markdown files.
package discover

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skillExtension is the file extension a discovered skill file must carry.
// Explicitly named files bypass this check: naming a file is a deliberate act.
const skillExtension = ".md"

// ErrNoSkillFiles is returned when the given paths contain nothing to scan.
var ErrNoSkillFiles = errors.New("no skill files found")

// Files expands paths into the skill files to scan. Explicit files keep their
// argument order, directory matches are sorted, and duplicates are collapsed so
// that overlapping arguments scan a file once.
func Files(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	add := func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		files = append(files, path)
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("read path %s: %w", path, err)
		}

		if !info.IsDir() {
			add(path)
			continue
		}

		found, err := walkDirectory(path)
		if err != nil {
			return nil, err
		}
		for _, file := range found {
			add(file)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%w in: %s", ErrNoSkillFiles, strings.Join(paths, ", "))
	}

	return files, nil
}

// walkDirectory returns the sorted skill files beneath root, skipping hidden
// entries so that directories such as .git never reach the scanner.
func walkDirectory(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && hidden(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !isSkillFile(entry.Name()) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", root, err)
	}

	sort.Strings(files)

	return files, nil
}

func hidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func isSkillFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), skillExtension)
}
