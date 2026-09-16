package drift

import (
	"context"
	"testing"
	"time"
)

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"internal/drift/git.go", true},
		{"cmd/mem/main.go", true},
		{"scripts/validate.py", true},
		{"frontend/app.tsx", true},
		{"Cargo.toml", false},
		{"docs/adr/001-test.md", false},
		{"README.md", false},
		{".memory/config.yaml", false},
		{"db/schema.sql", true},
	}

	for _, tc := range tests {
		result := IsCodeFile(tc.path)
		if result != tc.expected {
			t.Errorf("IsCodeFile(%q) = %v; esperava %v", tc.path, result, tc.expected)
		}
	}
}

func TestParseGitLogOutput(t *testing.T) {
	raw := `e1234567890abcdef1234567890abcdef1234567|e123456|Author Name|2026-09-14T18:00:00Z|feat: add new feature
f9876543210fedcba9876543210fedcba9876543|f987654|Another Dev|2026-09-14T19:30:00Z|fix: resolve issue with | separator`

	commits, err := parseGitLogOutput(raw)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("esperava 2 commits, obteve %d", len(commits))
	}

	if commits[0].ShortHash != "e123456" {
		t.Errorf("ShortHash = %q; esperava 'e123456'", commits[0].ShortHash)
	}
	if commits[0].Author != "Author Name" {
		t.Errorf("Author = %q; esperava 'Author Name'", commits[0].Author)
	}
	if commits[1].Message != "fix: resolve issue with | separator" {
		t.Errorf("Message = %q; esperava 'fix: resolve issue with | separator'", commits[1].Message)
	}
}

func TestParseGitStatusAndNumstat(t *testing.T) {
	statusRawNorm := "M\tinternal/drift/git.go\nA\tcmd/mem/drift.go\nD\told_file.go"
	statusMap := parseGitStatusMap(statusRawNorm)

	if statusMap["internal/drift/git.go"] != "M" {
		t.Errorf("esperava 'M', obteve %q", statusMap["internal/drift/git.go"])
	}
	if statusMap["cmd/mem/drift.go"] != "A" {
		t.Errorf("esperava 'A', obteve %q", statusMap["cmd/mem/drift.go"])
	}

	numstatRaw := "25\t5\tinternal/drift/git.go\n150\t0\tcmd/mem/drift.go\n0\t40\told_file.go"
	changes, err := parseGitNumstatOutput(numstatRaw, statusMap)
	if err != nil {
		t.Fatalf("erro no numstat: %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("esperava 3 changes, obteve %d", len(changes))
	}

	if changes[0].Additions != 25 || changes[0].Deletions != 5 || changes[0].Status != "M" {
		t.Errorf("change 0 incorreta: %+v", changes[0])
	}
	if !changes[0].IsCodeFile {
		t.Errorf("change 0 deveria ser código")
	}
}

func TestMockGitRunner(t *testing.T) {
	ctx := context.Background()
	mock := &MockGitRunner{
		Commits: []GitCommit{
			{Hash: "abc", ShortHash: "abc", Author: "Dev", Date: time.Now(), Message: "test"},
		},
		Changes: []FileChange{
			{Path: "pkg/core.go", Status: "M", Additions: 10, Deletions: 2, IsCodeFile: true},
		},
	}

	commits, err := mock.Log(ctx, "", "")
	if err != nil || len(commits) != 1 {
		t.Fatalf("falha ao obter commits do mock")
	}

	changes, err := mock.Diff(ctx, "", "")
	if err != nil || len(changes) != 1 {
		t.Fatalf("falha ao obter changes do mock")
	}
}
