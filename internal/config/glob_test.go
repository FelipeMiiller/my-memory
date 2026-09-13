package config

import (
	"testing"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"**/*.md", "note.md", true},
		{"**/*.md", "docs/guide.md", true},
		{"**/*.md", "a/b/c/deep.md", true},
		{"**/*.md", "note.txt", false},
		{"**/*.md", "docs/image.png", false},

		{".git/**", ".git", true},
		{".git/**", ".git/config", true},
		{".git/**", ".git/hooks/pre-commit", true},
		{".git/**", "sub/.git/HEAD", true},
		{".git/**", "git_notes/foo.md", false},

		{"node_modules/**", "node_modules/express/index.js", true},
		{"node_modules/**", "packages/app/node_modules/react/index.js", true},

		{"drafts/*", "drafts/idea.md", true},
		{"drafts/*", "drafts/sub/idea.md", false},

		{"temp_*.md", "temp_1.md", true},
		{"temp_*.md", "temp_abc.md", true},
		{"temp_*.md", "sub/temp_abc.md", true},
		{"temp_*.md", "final.md", false},

		{"/root_only.md", "root_only.md", true},
		{"/root_only.md", "sub/root_only.md", false},
	}

	for _, tc := range tests {
		got := MatchGlob(tc.pattern, tc.path)
		if got != tc.expected {
			t.Errorf("MatchGlob(%q, %q) = %v; esperava %v", tc.pattern, tc.path, got, tc.expected)
		}
	}
}

func TestShouldIndexWindowsSeparators(t *testing.T) {
	cfg := DefaultConfig()

	// Caminhos estilo Windows com backslash
	windowsPaths := []struct {
		path     string
		expected bool
	}{
		{`docs\architecture\system.md`, true},
		{`notes\daily\2026-09-13.md`, true},
		{`.git\objects\12\3456`, false},
		{`node_modules\pkg\README.md`, false},
		{`.obsidian\workspace.json`, false},
		{`vendor\lib\test.md`, false},
		{`.memory\config.yaml`, false},
		{`docs\image.png`, false},
	}

	for _, tc := range windowsPaths {
		got := cfg.ShouldIndex(tc.path)
		if got != tc.expected {
			t.Errorf("cfg.ShouldIndex(%q) = %v; esperava %v", tc.path, got, tc.expected)
		}
	}
}

func TestShouldIndexCustomConfig(t *testing.T) {
	cfg := Config{
		Version: 1,
		Include: []string{
			"docs/**/*.md",
			"wiki/**/*.markdown",
			"CHANGELOG.md",
		},
		Exclude: []string{
			"docs/drafts/**",
			"**/*_test.md",
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"docs/intro.md", true},
		{"docs/api/v1.md", true},
		{"docs/drafts/wip.md", false},  // Excluído por docs/drafts/**
		{"docs/api/v1_test.md", false}, // Excluído por **/*_test.md
		{"wiki/architecture.markdown", true},
		{"CHANGELOG.md", true},
		{"notes/personal.md", false},   // Não listado nos Includes
		{".git/config", false},         // Excluído por pasta padrão
		{"node_modules/doc.md", false}, // Excluído por pasta padrão
	}

	for _, tc := range tests {
		got := cfg.ShouldIndex(tc.path)
		if got != tc.expected {
			t.Errorf("cfg.ShouldIndex(%q) = %v; esperava %v", tc.path, got, tc.expected)
		}
	}
}

func TestShouldIndexNilConfig(t *testing.T) {
	var cfg *Config = nil

	if !cfg.ShouldIndex("docs/guide.md") {
		t.Errorf("nil config deveria usar DefaultConfig e permitir docs/guide.md")
	}
	if cfg.ShouldIndex(".git/config") {
		t.Errorf("nil config deveria rejeitar .git/config")
	}
	if cfg.ShouldIndex("image.png") {
		t.Errorf("nil config deveria rejeitar arquivos não .md")
	}
	if cfg.ShouldIndex("") {
		t.Errorf("caminho vazio deveria ser rejeitado")
	}
}
