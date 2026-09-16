package federation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// ResolvedNode representa uma nota federada resolvida com sucesso no sistema de arquivos
type ResolvedNode struct {
	URI          string `json:"uri"`                     // URI canônica completa (ex: "memory://central/standards/oauth2#JWT")
	RepoID       string `json:"repo_id"`                 // Identificador do repositório (ex: "repo_central" ou "repo_xxx")
	RepoName     string `json:"repo_name"`               // Nome amigável do repositório
	VaultName    string `json:"vault_name"`              // Nome do vault para o Obsidian
	VaultPath    string `json:"vault_path"`              // Caminho raiz do vault
	AbsolutePath string `json:"absolute_path"`           // Caminho absoluto para o arquivo físico no disco
	RelativePath string `json:"relative_path"`           // Caminho relativo dentro do vault (ex: "standards/oauth2.md")
	Anchor       string `json:"anchor,omitempty"`        // Âncora / seção / bloco opcional
	IsCentral    bool   `json:"is_central"`              // True se pertence ao Cofre Central
}

// ParseFederatedURI disseca uma URI federada canônica em seus componentes estruturais
func ParseFederatedURI(rawURI string) (repoTarget, docPath, anchor string, err error) {
	trimmed := strings.TrimSpace(rawURI)
	if !strings.HasPrefix(trimmed, "memory://") {
		return "", "", "", fmt.Errorf("URI federada inválida: deve iniciar com 'memory://', obteve '%s'", rawURI)
	}

	content := strings.TrimPrefix(trimmed, "memory://")

	// 1. Separa âncora (#) se presente
	hashIdx := strings.IndexByte(content, '#')
	if hashIdx != -1 {
		anchor = strings.TrimSpace(content[hashIdx+1:])
		content = strings.TrimSpace(content[:hashIdx])
	}

	// 2. Separa repositório do caminho do documento pela primeira barra
	slashIdx := strings.IndexByte(content, '/')
	if slashIdx == -1 {
		repoTarget = strings.TrimSpace(content)
		docPath = ""
	} else {
		repoTarget = strings.TrimSpace(content[:slashIdx])
		docPath = strings.Trim(content[slashIdx+1:], " /\\")
	}

	if repoTarget == "" {
		return "", "", "", errors.New("URI federada inválida: identificador do repositório não pode ser vazio")
	}

	return repoTarget, docPath, anchor, nil
}

// ResolveFederatedURI localiza o arquivo físico no disco correspondente a uma URI canônica
func ResolveFederatedURI(rawURI string, gcfg *config.GlobalConfig, lcfg *config.Config) (*ResolvedNode, error) {
	trimmed := strings.TrimSpace(rawURI)
	if !strings.HasPrefix(trimmed, "memory://") {
		return nil, fmt.Errorf("URI federada inválida: deve iniciar com 'memory://', obteve '%s'", rawURI)
	}

	content := strings.TrimPrefix(trimmed, "memory://")
	anchor := ""
	if hashIdx := strings.IndexByte(content, '#'); hashIdx != -1 {
		anchor = strings.TrimSpace(content[hashIdx+1:])
		content = strings.TrimSpace(content[:hashIdx])
	}

	var repoTarget, docPath string
	var matchedRepo *config.RepositoryCatalogEntry
	isCentral := false

	// 1. Verifica se o destino é o Cofre Central (central ou repo_central)
	if strings.EqualFold(content, "central") || strings.EqualFold(content, config.CentralRepoID) {
		repoTarget = config.CentralRepoID
		docPath = ""
		isCentral = true
	} else if strings.HasPrefix(strings.ToLower(content), "central/") {
		repoTarget = config.CentralRepoID
		docPath = strings.TrimPrefix(content[8:], "/")
		isCentral = true
	} else if strings.HasPrefix(strings.ToLower(content), config.CentralRepoID+"/") {
		repoTarget = config.CentralRepoID
		docPath = strings.TrimPrefix(content[len(config.CentralRepoID)+1:], "/")
		isCentral = true
	}

	// 2. Se não for central, busca no catálogo (suporta slugs com barra ex: owner/repo)
	if !isCentral {
		if gcfg != nil {
			for i, r := range gcfg.Repositories {
				if strings.EqualFold(content, r.ID) {
					matchedRepo = &gcfg.Repositories[i]
					repoTarget = r.ID
					docPath = ""
					break
				}
				if strings.HasPrefix(strings.ToLower(content), strings.ToLower(r.ID)+"/") {
					matchedRepo = &gcfg.Repositories[i]
					repoTarget = r.ID
					docPath = strings.TrimPrefix(content[len(r.ID)+1:], "/")
					break
				}
				if r.Name != "" {
					if strings.EqualFold(content, r.Name) {
						matchedRepo = &gcfg.Repositories[i]
						repoTarget = r.ID
						docPath = ""
						break
					}
					if strings.HasPrefix(strings.ToLower(content), strings.ToLower(r.Name)+"/") {
						matchedRepo = &gcfg.Repositories[i]
						repoTarget = r.ID
						docPath = strings.TrimPrefix(content[len(r.Name)+1:], "/")
						break
					}
				}
			}
		}

		// Fallback para divisão na primeira barra se não casou contra o catálogo
		if matchedRepo == nil {
			slashIdx := strings.IndexByte(content, '/')
			if slashIdx == -1 {
				repoTarget = content
				docPath = ""
			} else {
				repoTarget = content[:slashIdx]
				docPath = strings.Trim(content[slashIdx+1:], " /\\")
			}
		}
	}

	// 3. Resolução para o Cofre Central
	if isCentral {
		centralPath := ""
		if lcfg != nil && strings.TrimSpace(lcfg.CentralVault.Path) != "" {
			centralPath = lcfg.CentralVault.Path
		} else if gcfg != nil && strings.TrimSpace(gcfg.CentralVault.Path) != "" {
			centralPath = gcfg.CentralVault.Path
		} else if envPath := os.Getenv("MY_MEMORY_CENTRAL_VAULT"); envPath != "" {
			centralPath = envPath
		}

		if strings.TrimSpace(centralPath) == "" {
			return nil, errors.New("caminho do cofre central não configurado em ~/.memory/config.yaml nem .memory/config.yaml")
		}

		cleanCentral := filepath.Clean(config.ExpandPath(centralPath))
		if stat, err := os.Stat(cleanCentral); err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("cofre central em '%s' não encontrado ou inacessível (verifique a sincronização/montagem do Google Drive ou OneDrive)", cleanCentral)
		}

		vaultName := ""
		if lcfg != nil && strings.TrimSpace(lcfg.CentralVault.VaultName) != "" {
			vaultName = strings.TrimSpace(lcfg.CentralVault.VaultName)
		} else if gcfg != nil && strings.TrimSpace(gcfg.CentralVault.VaultName) != "" {
			vaultName = strings.TrimSpace(gcfg.CentralVault.VaultName)
		}
		if vaultName == "" {
			vaultName = filepath.Base(cleanCentral)
		}
		absFile, relFile, err := findDocumentInVault(cleanCentral, docPath)
		if err != nil {
			return nil, fmt.Errorf("documento '%s' no cofre central: %w", docPath, err)
		}

		return &ResolvedNode{
			URI:          rawURI,
			RepoID:       config.CentralRepoID,
			RepoName:     "Central Knowledge Vault",
			VaultName:    vaultName,
			VaultPath:    cleanCentral,
			AbsolutePath: absFile,
			RelativePath: relFile,
			Anchor:       anchor,
			IsCentral:    true,
		}, nil
	}

	// Verifica se é o repositório local atual
	if matchedRepo == nil && lcfg != nil {
		if strings.EqualFold(lcfg.RepoID, repoTarget) ||
			strings.EqualFold(lcfg.Repository, repoTarget) {
			cwd, _ := os.Getwd()
			matchedRepo = &config.RepositoryCatalogEntry{
				ID:   lcfg.RepoID,
				Path: cwd,
				Name: lcfg.Repository,
			}
		}
	}

	if matchedRepo == nil {
		return nil, fmt.Errorf("repositório federado '%s' não encontrado no catálogo de ~/.memory/config.yaml", repoTarget)
	}

	cleanRepoPath := filepath.Clean(config.ExpandPath(matchedRepo.Path))
	if stat, err := os.Stat(cleanRepoPath); err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("pasta do repositório federado '%s' (%s) não encontrada no disco", matchedRepo.Name, cleanRepoPath)
	}

	vaultName := strings.TrimSpace(matchedRepo.VaultName)
	if vaultName == "" {
		vaultName = matchedRepo.Name
	}
	if vaultName == "" {
		vaultName = filepath.Base(cleanRepoPath)
	}

	absFile, relFile, err := findDocumentInVault(cleanRepoPath, docPath)
	if err != nil {
		return nil, fmt.Errorf("documento '%s' no repositório '%s': %w", docPath, matchedRepo.Name, err)
	}

	return &ResolvedNode{
		URI:          rawURI,
		RepoID:       matchedRepo.ID,
		RepoName:     matchedRepo.Name,
		VaultName:    vaultName,
		VaultPath:    cleanRepoPath,
		AbsolutePath: absFile,
		RelativePath: relFile,
		Anchor:       anchor,
		IsCentral:    false,
	}, nil
}

// findDocumentInVault busca o arquivo no vault testando com e sem extensão .md
func findDocumentInVault(vaultRoot, docPath string) (absPath, relPath string, err error) {
	if strings.TrimSpace(docPath) == "" {
		// Se docPath for vazio, aponta para README.md do vault
		readmePath := filepath.Join(vaultRoot, "README.md")
		if stat, err := os.Stat(readmePath); err == nil && !stat.IsDir() {
			return readmePath, "README.md", nil
		}
		return vaultRoot, "", nil
	}

	// Normaliza separadores
	normalized := filepath.FromSlash(docPath)

	candidates := []string{
		filepath.Join(vaultRoot, normalized),
	}
	if !strings.HasSuffix(strings.ToLower(normalized), ".md") {
		candidates = append(candidates, filepath.Join(vaultRoot, normalized+".md"))
	}

	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && !stat.IsDir() {
			absClean := filepath.Clean(c)
			rel, relErr := filepath.Rel(vaultRoot, absClean)
			if relErr != nil {
				rel = filepath.Base(absClean)
			}
			return absClean, filepath.ToSlash(rel), nil
		}
	}

	return "", "", fmt.Errorf("arquivo não encontrado em '%s' (tentativas: %v)", vaultRoot, candidates)
}
