package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestIndexWalkFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Arquivos que DEVEM ser aceitos
	acceptedFiles := []string{
		"docs/intro.md",
		"docs/architecture.md",
		"notes/daily.md",
	}

	// Arquivos que DEVEM ser rejeitados (por extensão, pasta de sistema ou exclude customizado)
	rejectedFiles := []string{
		"docs/image.png",
		"docs/drafts/wip.md",
		"node_modules/dep/README.md",
		".git/info/exclude.md",
		".obsidian/workspace.json",
		"notes/temp_scratch.md",
	}

	for _, rel := range append(acceptedFiles, rejectedFiles...) {
		full := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# Test Note"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		Version: 1,
		Include: []string{
			"**/*.md",
		},
		Exclude: []string{
			".git/**",
			"node_modules/**",
			".obsidian/**",
			"docs/drafts/**",
			"**/temp_*.md",
		},
	}

	var scannedAccepted []string
	var scannedRejected []string

	err := filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(tmpDir, path)
		if relErr != nil {
			relPath = path
		}

		if d.IsDir() {
			if relPath != "." {
				base := d.Name()
				for _, defEx := range config.DefaultExcludedDirs {
					if base == defEx {
						return filepath.SkipDir
					}
				}
				for _, ex := range cfg.Exclude {
					cleanEx := strings.TrimSuffix(ex, "/**")
					if relPath == cleanEx || strings.HasSuffix(relPath, "/"+cleanEx) {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		if !cfg.ShouldIndex(relPath) {
			scannedRejected = append(scannedRejected, filepath.ToSlash(relPath))
			return nil
		}

		scannedAccepted = append(scannedAccepted, filepath.ToSlash(relPath))
		return nil
	})

	if err != nil {
		t.Fatalf("erro ao caminhar pelo diretório: %v", err)
	}

	// Verifica se todos os arquivos esperados foram aceitos
	if len(scannedAccepted) != len(acceptedFiles) {
		t.Errorf("esperava %d arquivos aceitos, obteve %d: %v", len(acceptedFiles), len(scannedAccepted), scannedAccepted)
	}
	for _, expected := range acceptedFiles {
		found := false
		for _, acc := range scannedAccepted {
			if acc == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("arquivo esperado %s não foi encontrado nos aceitos: %v", expected, scannedAccepted)
		}
	}

	// Garante que nenhum rejeitado foi aceito
	for _, rej := range rejectedFiles {
		for _, acc := range scannedAccepted {
			if acc == rej {
				t.Errorf("arquivo rejeitado %s foi aceito indevidamente!", rej)
			}
		}
	}
}
