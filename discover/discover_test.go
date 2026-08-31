package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiles(t *testing.T) {
	tests := []struct {
		name    string
		layout  []string
		paths   []string
		want    []string
		wantErr string
	}{
		{
			name:   "single file is returned as given",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/alpha.md"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "explicit files keep argument order",
			layout: []string{"skills/alpha.md", "skills/beta.md"},
			paths:  []string{"skills/beta.md", "skills/alpha.md"},
			want:   []string{"skills/beta.md", "skills/alpha.md"},
		},
		{
			name:   "directory is walked for markdown files",
			layout: []string{"skills/beta.md", "skills/alpha.md", "skills/notes.txt"},
			paths:  []string{"skills"},
			want:   []string{"skills/alpha.md", "skills/beta.md"},
		},
		{
			name:   "nested directories are included",
			layout: []string{"skills/alpha.md", "skills/nested/deep/gamma.md"},
			paths:  []string{"skills"},
			want:   []string{"skills/alpha.md", "skills/nested/deep/gamma.md"},
		},
		{
			name:   "hidden directories are skipped",
			layout: []string{"skills/alpha.md", "skills/.git/hooks.md"},
			paths:  []string{"skills"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "hidden files are skipped",
			layout: []string{"skills/alpha.md", "skills/.draft.md"},
			paths:  []string{"skills"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "markdown extension match is case insensitive",
			layout: []string{"skills/alpha.MD"},
			paths:  []string{"skills"},
			want:   []string{"skills/alpha.MD"},
		},
		{
			name:   "explicit file is taken whatever its extension",
			layout: []string{"skills/alpha.skill"},
			paths:  []string{"skills/alpha.skill"},
			want:   []string{"skills/alpha.skill"},
		},
		{
			name:   "duplicate paths are collapsed",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/alpha.md", "skills", "skills/alpha.md"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:    "missing path is reported with the path",
			layout:  []string{"skills/alpha.md"},
			paths:   []string{"skills/nope.md"},
			wantErr: "skills/nope.md",
		},
		{
			name:    "directory without skill files is an error",
			layout:  []string{"skills/notes.txt"},
			paths:   []string{"skills"},
			wantErr: "no skill files found",
		},
		{
			name:    "no paths is an error",
			layout:  []string{"skills/alpha.md"},
			paths:   nil,
			wantErr: "no skill files found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, relative := range tt.layout {
				writeFile(t, filepath.Join(root, filepath.FromSlash(relative)))
			}

			paths := make([]string, 0, len(tt.paths))
			for _, relative := range tt.paths {
				paths = append(paths, filepath.Join(root, filepath.FromSlash(relative)))
			}

			got, err := Files(paths)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Files() error = nil, want substring %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Files() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Files() error = %v, want nil", err)
			}

			want := make([]string, 0, len(tt.want))
			for _, relative := range tt.want {
				want = append(want, filepath.Join(root, filepath.FromSlash(relative)))
			}
			if len(got) != len(want) {
				t.Fatalf("Files() = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("Files()[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}

func TestFilesReportsUnreadableDirectory(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatalf("os.Mkdir() error = %v", err)
	}
	writeFile(t, filepath.Join(blocked, "alpha.md"))
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	if _, err := Files([]string{blocked}); err == nil {
		t.Fatal("Files() error = nil, want a walk failure")
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("---\nname: a\ndescription: b\n---\nbody"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

// TestFilesCollapsesEquivalentSpellings joins its paths by concatenation rather
// than filepath.Join, because Join cleans the result and would hide the very
// spellings this test is about.
func TestFilesCollapsesEquivalentSpellings(t *testing.T) {
	tests := []struct {
		name   string
		layout []string
		paths  []string
		want   []string
	}{
		{
			name:   "dot segment spelling collapses",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/alpha.md", "skills/./alpha.md"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "doubled separator spelling collapses",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/alpha.md", "skills//alpha.md"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "parent traversal spelling collapses",
			layout: []string{"skills/alpha.md", "skills/nested/beta.md"},
			paths:  []string{"skills/alpha.md", "skills/nested/../alpha.md"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "trailing slash directory collapses against the same directory",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/", "skills"},
			want:   []string{"skills/alpha.md"},
		},
		{
			name:   "explicit file collapses against its parent directory",
			layout: []string{"skills/alpha.md", "skills/beta.md"},
			paths:  []string{"skills/./alpha.md", "skills"},
			want:   []string{"skills/./alpha.md", "skills/beta.md"},
		},
		{
			name:   "first spelling is the one retained",
			layout: []string{"skills/alpha.md"},
			paths:  []string{"skills/./alpha.md", "skills/alpha.md"},
			want:   []string{"skills/./alpha.md"},
		},
		{
			name:   "genuinely different files are never collapsed",
			layout: []string{"skills/alpha.md", "skills/beta.md"},
			paths:  []string{"skills/alpha.md", "skills/./beta.md"},
			want:   []string{"skills/alpha.md", "skills/./beta.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, relative := range tt.layout {
				writeFile(t, filepath.Join(root, filepath.FromSlash(relative)))
			}

			paths := make([]string, 0, len(tt.paths))
			for _, relative := range tt.paths {
				paths = append(paths, root+string(filepath.Separator)+filepath.FromSlash(relative))
			}

			got, err := Files(paths)
			if err != nil {
				t.Fatalf("Files() error = %v, want nil", err)
			}

			want := make([]string, 0, len(tt.want))
			for _, relative := range tt.want {
				want = append(want, root+string(filepath.Separator)+filepath.FromSlash(relative))
			}
			if len(got) != len(want) {
				t.Fatalf("Files() = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("Files()[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}
