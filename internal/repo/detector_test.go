package repo

import (
	"os"
	"testing"
)

func TestCleanGitURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/FelipeMiiller/my-memory.git", "FelipeMiiller/my-memory"},
		{"http://github.com/org/project.git", "org/project"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"https://gitlab.com/group/subgroup/repo.git", "group/subgroup/repo"},
		{"my-local-folder", "my-local-folder"},
		{"", ""},
	}

	for _, tt := range tests {
		got := CleanGitURL(tt.input)
		if got != tt.expected {
			t.Errorf("CleanGitURL(%q) = %q, esperava %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectRepository_CurrentRepo(t *testing.T) {
	repo := DetectRepository(".")
	if repo != "FelipeMiiller/my-memory" && repo != "my-memory" {
		t.Errorf("esperava slug do repositório 'FelipeMiiller/my-memory' ou 'my-memory', obteve %q", repo)
	}
}

func TestDetectRepository_EnvOverride(t *testing.T) {
	os.Setenv("MY_MEMORY_REPO", "custom-org/custom-repo")
	defer os.Unsetenv("MY_MEMORY_REPO")

	repo := DetectRepository(".")
	if repo != "custom-org/custom-repo" {
		t.Errorf("esperava override via env 'custom-org/custom-repo', obteve %q", repo)
	}
}
