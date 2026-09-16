package federation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestParseFederatedURI(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantRepo   string
		wantDoc    string
		wantAnchor string
		wantErr    bool
	}{
		{
			name:       "central simple doc",
			raw:        "memory://central/standards/oauth2",
			wantRepo:   "central",
			wantDoc:    "standards/oauth2",
			wantAnchor: "",
			wantErr:    false,
		},
		{
			name:       "central with anchor",
			raw:        "memory://repo_central/architecture/database#PostgreSQL",
			wantRepo:   "repo_central",
			wantDoc:    "architecture/database",
			wantAnchor: "PostgreSQL",
			wantErr:    false,
		},
		{
			name:       "satellite repo by id",
			raw:        "memory://repo_123456789abc/docs/api-spec#^sec1",
			wantRepo:   "repo_123456789abc",
			wantDoc:    "docs/api-spec",
			wantAnchor: "^sec1",
			wantErr:    false,
		},
		{
			name:       "central root without doc",
			raw:        "memory://central",
			wantRepo:   "central",
			wantDoc:    "",
			wantAnchor: "",
			wantErr:    false,
		},
		{
			name:    "invalid scheme",
			raw:     "https://central/doc",
			wantErr: true,
		},
		{
			name:    "empty repo",
			raw:     "memory://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, doc, anchor, err := ParseFederatedURI(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFederatedURI error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if repo != tt.wantRepo {
					t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
				}
				if doc != tt.wantDoc {
					t.Errorf("doc = %q, want %q", doc, tt.wantDoc)
				}
				if anchor != tt.wantAnchor {
					t.Errorf("anchor = %q, want %q", anchor, tt.wantAnchor)
				}
			}
		})
	}
}

func TestResolveFederatedURI(t *testing.T) {
	// Cria estrutura temporária simulando Central Vault e Satélite
	baseDir := t.TempDir()

	centralVaultDir := filepath.Join(baseDir, "central-vault")
	if err := os.MkdirAll(filepath.Join(centralVaultDir, "standards"), 0755); err != nil {
		t.Fatal(err)
	}
	readmeContent := "# Central Brain README\n"
	if err := os.WriteFile(filepath.Join(centralVaultDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}
	authNote := "# Padrão OAuth2\n"
	if err := os.WriteFile(filepath.Join(centralVaultDir, "standards", "oauth2.md"), []byte(authNote), 0644); err != nil {
		t.Fatal(err)
	}

	satelliteDir := filepath.Join(baseDir, "satellite-repo")
	if err := os.MkdirAll(filepath.Join(satelliteDir, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	apiContract := "# API Contract\n"
	if err := os.WriteFile(filepath.Join(satelliteDir, "docs", "contract.md"), []byte(apiContract), 0644); err != nil {
		t.Fatal(err)
	}

	gcfg := &config.GlobalConfig{
		Version: 1,
		CentralVault: config.CentralVaultConfig{
			Path: centralVaultDir,
		},
		Repositories: []config.RepositoryCatalogEntry{
			{
				ID:   "repo_sat12345678",
				Path: satelliteDir,
				Name: "acme/satellite",
			},
		},
	}

	lcfg := &config.Config{
		Version: 1,
		RepoID:  "repo_local99999",
	}

	t.Run("Resolve central vault note with and without md extension", func(t *testing.T) {
		resolved, err := ResolveFederatedURI("memory://central/standards/oauth2", gcfg, lcfg)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !resolved.IsCentral {
			t.Errorf("esperava IsCentral=true")
		}
		if resolved.RepoID != config.CentralRepoID {
			t.Errorf("esperava RepoID=%s, obteve %s", config.CentralRepoID, resolved.RepoID)
		}
		if resolved.RelativePath != "standards/oauth2.md" {
			t.Errorf("esperava relative path 'standards/oauth2.md', obteve %s", resolved.RelativePath)
		}
		if resolved.VaultName != filepath.Base(centralVaultDir) {
			t.Errorf("esperava VaultName=%s, obteve %s", filepath.Base(centralVaultDir), resolved.VaultName)
		}

		// Com .md explícito e âncora
		resolvedWithExt, err := ResolveFederatedURI("memory://repo_central/standards/oauth2.md#JWT", gcfg, lcfg)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resolvedWithExt.Anchor != "JWT" {
			t.Errorf("esperava Anchor='JWT', obteve %s", resolvedWithExt.Anchor)
		}
	})

	t.Run("Resolve satellite repository by ID", func(t *testing.T) {
		resolved, err := ResolveFederatedURI("memory://repo_sat12345678/docs/contract", gcfg, lcfg)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resolved.IsCentral {
			t.Errorf("esperava IsCentral=false para satélite")
		}
		if resolved.RepoID != "repo_sat12345678" {
			t.Errorf("esperava RepoID='repo_sat12345678', obteve %s", resolved.RepoID)
		}
		if resolved.RelativePath != "docs/contract.md" {
			t.Errorf("esperava RelativePath='docs/contract.md', obteve %s", resolved.RelativePath)
		}
	})

	t.Run("Resolve satellite repository by Name slug", func(t *testing.T) {
		resolved, err := ResolveFederatedURI("memory://acme/satellite/docs/contract", gcfg, lcfg)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resolved.RepoName != "acme/satellite" {
			t.Errorf("esperava RepoName='acme/satellite', obteve %s", resolved.RepoName)
		}
	})

	t.Run("Error when central vault is missing or unmounted", func(t *testing.T) {
		badGCFG := &config.GlobalConfig{
			Version: 1,
			CentralVault: config.CentralVaultConfig{
				Path: filepath.Join(baseDir, "non-existent-drive-mount"),
			},
		}
		_, err := ResolveFederatedURI("memory://central/standards/oauth2", badGCFG, lcfg)
		if err == nil {
			t.Errorf("esperava erro ao referenciar cofre central inexistente")
		}
	})

	t.Run("Error when document not found in vault", func(t *testing.T) {
		_, err := ResolveFederatedURI("memory://central/standards/non-existent", gcfg, lcfg)
		if err == nil {
			t.Errorf("esperava erro ao referenciar documento inexistente")
		}
	})

	t.Run("Error when repository not in catalog", func(t *testing.T) {
		_, err := ResolveFederatedURI("memory://repo_unknown/docs/readme", gcfg, lcfg)
		if err == nil {
			t.Errorf("esperava erro para repositório não catalogado")
		}
	})
}
