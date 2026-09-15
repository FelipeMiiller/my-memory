package federation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"gopkg.in/yaml.v3"
)

// StandardFolders lista as 11 pastas canônicas da taxonomia de governança de conhecimento
var StandardFolders = []string{
	"standards",
	"architecture",
	"security",
	"infrastructure",
	"operations",
	"data",
	"ai-agents",
	"domain",
	"guides",
	"templates",
	"staging",
}

// FolderDescriptions descreve o propósito de cada pasta canônica
var FolderDescriptions = map[string]string{
	"standards":      "Diretrizes de codificação, convenções de código, naming, linting e style guides corporativos.",
	"architecture":   "Decisões arquiteturais corporativas (ADRs), topologias, diagramas C4 e padrões de microsserviços.",
	"security":       "Políticas de autenticação, RBAC, gestão de segredos, criptografia e requisitos de conformidade.",
	"infrastructure": "Modelos de IaC (Terraform, Ansible), topologia de nuvem, Kubernetes e políticas de rede.",
	"operations":     "Observabilidade, métricas, SLOs, runbooks de incidentes e playbooks operacionais.",
	"data":           "Modelos de dados canônicos, esquemas de bancos relacionais e NoSQL, e governança de dados.",
	"ai-agents":      "Skills compartilhadas para IA, personas de agentes, instruções operacionais e diretrizes MCP.",
	"domain":         "Glossário ubíquo corporativo, taxonomia de negócio e regras de domínio transversais.",
	"guides":         "Manuais de onboarding de novos engenheiros, guias práticos 'how-to' e tutorais passo a passo.",
	"templates":      "Modelos reutilizáveis para documentação técnica padronizada (ADR, RFC, Runbook, Spec).",
	"staging":        "Área transitória para captura rápida de rascunhos e ideação antes da curadoria e promoção.",
}

// IsCentralVaultInitialized verifica se o diretório especificado já está inicializado como cofre central
func IsCentralVaultInitialized(vaultPath string) bool {
	if strings.TrimSpace(vaultPath) == "" {
		return false
	}
	cfgPath := filepath.Join(vaultPath, ".memory", "config.yaml")
	if stat, err := os.Stat(cfgPath); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

// BootstrapCentralVault realiza o Zero-Touch Auto-Bootstrap do cofre central virgem.
// Se o diretório já contiver a estrutura básica, a operação é idempotente (no-op).
func BootstrapCentralVault(vaultPath string) error {
	if strings.TrimSpace(vaultPath) == "" {
		return fmt.Errorf("caminho do cofre central não pode ser vazio")
	}

	cleanPath := filepath.Clean(config.ExpandPath(vaultPath))

	// Se já inicializado, não faz nada (idempotente)
	if IsCentralVaultInitialized(cleanPath) {
		return nil
	}

	// 1. Cria a pasta raiz do vault
	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório raiz do cofre central '%s': %w", cleanPath, err)
	}

	// 2. Cria as 11 pastas canônicas e seus README.md de seção
	for _, folder := range StandardFolders {
		folderPath := filepath.Join(cleanPath, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return fmt.Errorf("erro ao criar pasta canônica '%s': %w", folderPath, err)
		}

		readmePath := filepath.Join(folderPath, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			desc := FolderDescriptions[folder]
			title := strings.Title(strings.ReplaceAll(folder, "-", " "))
			content := fmt.Sprintf(`---
title: "%s"
category: memory
tags: [%s, knowledge-vault, central]
---

# %s

%s

---

## 📑 Notas nesta seção
- [[../README|Retornar ao Mapa de Conteúdo (MOC)]]
`, title, folder, title, desc)

			if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("erro ao criar README em '%s': %w", readmePath, err)
			}
		}
	}

	// 3. Provisiona os templates canônicos em templates/
	if err := provisionTemplates(cleanPath); err != nil {
		return err
	}

	// 4. Cria o MOC raiz (README.md)
	if err := provisionRootMOC(cleanPath); err != nil {
		return err
	}

	// 5. Cria a estrutura oculta isolada .memory/ e banco SQLite
	if err := provisionMemoryIsolation(cleanPath); err != nil {
		return err
	}

	return nil
}

func provisionTemplates(cleanPath string) error {
	templatesDir := filepath.Join(cleanPath, "templates")

	templates := map[string]string{
		"adr.md": `---
title: "ADR-{NUMBER}: {TITLE}"
category: memory
status: proposed
tags: [architecture, adr, decision]
---

# ADR-{NUMBER}: {TITLE}

## Contexto e Declaração do Problema
{Descreva o contexto técnico e a necessidade de negócio que motivou a decisão.}

## Decisores
- {Nome 1}
- {Nome 2}

## Opções Consideradas
1. {Opção 1}
2. {Opção 2}

## Decisão Escolhida
{Explique a opção escolhida e a justificativa principal.}

## Consequências
### Positivas
- {Consequência positiva 1}

### Negativas / Trade-offs
- {Trade-off aceito}
`,
		"rfc.md": `---
title: "RFC-{NUMBER}: {TITLE}"
category: memory
status: draft
tags: [rfc, proposal, engineering]
---

# RFC-{NUMBER}: {TITLE}

## Resumo Executivo
{Breve resumo da proposta de mudança.}

## Motivação
{Qual problema de engenharia estamos resolvendo?}

## Proposta Detalhada
{Especificações técnicas, diagramas de sequência ou dados.}

## Alternativas Rejeitadas
{Quais outras abordagens foram consideradas e por que foram descartadas?}

## Plano de Rollout e Migração
{Passos de implementação e contingência.}
`,
		"runbook.md": `---
title: "RUNBOOK: {SERVICE_OR_ALERT_NAME}"
category: memory
status: active
tags: [operations, runbook, incident-response]
---

# RUNBOOK: {SERVICE_OR_ALERT_NAME}

## Descrição do Serviço / Alerta
{O que este alerta significa e qual o impacto no usuário.}

## Diagnóstico Rápido
1. {Comando ou link de dashboard para verificar métricas}
2. {Checagem de logs relevantes}

## Passos de Remediação
1. {Passo 1}
2. {Passo 2}

## Escalação
- Contato On-Call: {canal / responsável}
`,
		"spec.md": `---
title: "SPEC: {FEATURE_NAME}"
category: memory
status: draft
tags: [spec, requirements, ears]
---

# SPEC: {FEATURE_NAME}

## Contexto
{Contexto do negócio e requisitos.}

## Requisitos em Notação EARS
- [UBIQUITOUS] O sistema DEVE {comportamento contínuo}.
- [EVENT-DRIVEN] QUANDO {evento ocorrer}, o sistema DEVE {resposta}.
- [STATE-DRIVEN] ENQUANTO {estado estiver ativo}, o sistema DEVE {comportamento}.
- [OPTIONAL] ONDE {recurso opcional disponível}, o sistema DEVE {comportamento}.
- [UNWANTED] SE {falha ou erro}, ENTÃO o sistema DEVE {mitigação}.
`,
	}

	for fileName, content := range templates {
		destPath := filepath.Join(templatesDir, fileName)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("erro ao criar template '%s': %w", destPath, err)
			}
		}
	}

	return nil
}

func provisionRootMOC(cleanPath string) error {
	mocPath := filepath.Join(cleanPath, "README.md")
	if _, err := os.Stat(mocPath); err == nil {
		return nil
	}

	mocContent := `# 🏛️ Central Knowledge Vault

Bem-vindo ao **Cofre Central de Conhecimento** corporativo integrado ao ecossistema [[My-Memory]].
Este vault funciona como a **Memória Global** (*Global Brain*) compartilhada entre repositórios satélites, agentes de IA e equipes de engenharia.

---

## 🗺️ Mapa de Conteúdo (MOC)

| Seção | Descrição | Wikilink |
| :--- | :--- | :--- |
| **Padrões** | Diretrizes de codificação, convenções, linters e style guides | [[standards/README\|Standards]] |
| **Arquitetura** | Decisões arquiteturais corporativas (ADRs), topologias e C4 | [[architecture/README\|Architecture]] |
| **Segurança** | Políticas de autenticação, RBAC, segredos e conformidade | [[security/README\|Security]] |
| **Infraestrutura** | IaC, Kubernetes, topologia de nuvem e redes | [[infrastructure/README\|Infrastructure]] |
| **Operações** | SLOs, observabilidade, alertas e runbooks de incidentes | [[operations/README\|Operations]] |
| **Dados** | Modelos canônicos, esquemas de dados e governança | [[data/README\|Data]] |
| **Agentes de IA** | Skills compartilhadas, diretrizes MCP e personas de IA | [[ai-agents/README\|AI Agents]] |
| **Domínio** | Glossário ubíquo corporativo e regras de negócio transversais | [[domain/README\|Domain]] |
| **Guias** | Playbooks, manuais de onboarding e guias 'how-to' | [[guides/README\|Guides]] |
| **Templates** | Modelos canônicos para ADR, RFC, Runbook e Especificações | [[templates/README\|Templates]] |
| **Staging** | Área transitória para ideação rápida antes da curadoria e promoção | [[staging/README\|Staging]] |

---

## 🧭 Ciclo de Vida do Conhecimento (Staging → Curadoria → Promoção)

1. **Captura Inicial**: Qualquer nova ideia, anotação de reunião ou rascunho de arquitetura inicia em [[staging/README|staging/]].
2. **Curadoria e Refinamento**: O time e agentes de IA refinam a nota, adicionando metadados YAML (category, summary, tags) e wikilinks.
3. **Promoção**: Uma vez validada, a nota é movida para sua pasta temática definitiva ([[standards/README|standards/]], [[architecture/README|architecture/]], etc.).
`

	if err := os.WriteFile(mocPath, []byte(mocContent), 0644); err != nil {
		return fmt.Errorf("erro ao criar MOC raiz '%s': %w", mocPath, err)
	}
	return nil
}

func provisionMemoryIsolation(cleanPath string) error {
	memDir := filepath.Join(cleanPath, ".memory")
	storageDir := filepath.Join(memDir, "storage")

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar pasta .memory/storage em '%s': %w", cleanPath, err)
	}

	// 1. .memory/config.yaml
	cfgPath := filepath.Join(memDir, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		centralConfig := config.Config{
			Version:    1,
			RepoID:     config.CentralRepoID,
			Repository: "central/vault",
			VaultName:  "Central Knowledge Vault",
			Include: []string{
				"**/*.md",
			},
			Exclude: []string{
				".git/**",
				".obsidian/**",
				".trash/**",
				".memory/**",
			},
			Storage: config.StorageConfig{
				Engine:     "sqlite",
				SQLitePath: "storage/memory.db",
			},
			Embedding: config.EmbeddingConfig{
				Provider:  "ollama",
				Model:     "nomic-embed-text",
				URL:       "http://localhost:11434",
				Dimension: 768,
			},
			Search: config.SearchConfig{
				Mode:        "hybrid",
				Limit:       5,
				K:           60,
				Decay:       false,
				HalfLife:    30.0,
				DecayWeight: 0.3,
				UseTurbo:    false,
				Level:       "l1",
			},
			Watcher: config.WatcherConfig{
				DebounceMs: 500,
				IntervalMs: 1000,
			},
			Editor: config.EditorConfig{
				DefaultApp: "obsidian",
			},
			MCP: config.MCPConfig{
				Port: 8080,
			},
		}

		data, err := yaml.Marshal(centralConfig)
		if err != nil {
			return fmt.Errorf("erro ao serializar config central: %w", err)
		}

		header := []byte("# Configuração Declarativa do Cofre Central (Global Brain)\n\n")
		content := append(header, data...)
		if err := os.WriteFile(cfgPath, content, 0644); err != nil {
			return fmt.Errorf("erro ao salvar .memory/config.yaml central: %w", err)
		}
	}

	// 2. .memory/.gitignore
	gitIgnorePath := filepath.Join(memDir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		gitIgnoreContent := `# Isolamento do SQLite central (impede poluição do Obsidian)
storage/*.db
storage/*.db-journal
storage/*.db-wal
storage/*.db-shm
*.db
*.db-journal
*.db-wal
*.db-shm

# Segredos locais
.env
*.env
*.local
`
		if err := os.WriteFile(gitIgnorePath, []byte(gitIgnoreContent), 0644); err != nil {
			return fmt.Errorf("erro ao criar .memory/.gitignore: %w", err)
		}
	}

	return nil
}
