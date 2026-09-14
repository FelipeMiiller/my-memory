package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/deeplink"
)

// OpenNodeFunc assinatura para resolução do caminho de um nó a partir do nodeID
type OpenNodeFunc func(ctx context.Context, repo string, nodeID string) (resolvedPath string, err error)

// OpenNodeResult define a estrutura de resposta para a ferramenta memory_open_node
type OpenNodeResult struct {
	NodeID    string             `json:"node_id"`
	Path      string             `json:"path"`
	App       string             `json:"app"`
	Line      int                `json:"line,omitempty"`
	Links     deeplink.DeepLinks `json:"links"`
	TargetURI string             `json:"target_uri"`
	Action    string             `json:"action"`
	Success   bool               `json:"success"`
	Message   string             `json:"message"`
}

// NewMemoryOpenNodeHandler cria o executor para a ferramenta MCP memory_open_node
func NewMemoryOpenNodeHandler(resolveFn OpenNodeFunc, repoRoot string, vaultName string, launcher deeplink.Launcher) ToolHandlerFunc {
	if launcher == nil {
		launcher = deeplink.DefaultLauncher
	}

	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		var nodeID string
		if rawNode, ok := rawMap["node_id"]; ok {
			_ = json.Unmarshal(rawNode, &nodeID)
		}
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'node_id' é obrigatório", nil)
		}

		var app string
		if rawApp, ok := rawMap["app"]; ok {
			_ = json.Unmarshal(rawApp, &app)
		}
		app = strings.TrimSpace(app)
		if app == "" {
			app = "obsidian"
		}

		var line int
		if rawLine, ok := rawMap["line"]; ok {
			_ = json.Unmarshal(rawLine, &line)
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		var action string
		if rawAction, ok := rawMap["action"]; ok {
			_ = json.Unmarshal(rawAction, &action)
		}
		action = strings.ToLower(strings.TrimSpace(action))
		if action == "" {
			action = "links_only"
		}

		// 1. Resolução do caminho do nó
		resolvedPath := nodeID
		if resolveFn != nil {
			if p, err := resolveFn(ctx, repo, nodeID); err == nil && p != "" {
				resolvedPath = p
			}
		}

		// 2. Geração dos links
		links := deeplink.GenerateLinks(repoRoot, vaultName, resolvedPath, line)

		var targetURI string
		switch strings.ToLower(app) {
		case "vscode", "code":
			targetURI = links.VSCode
		case "system", "file", "default":
			targetURI = links.File
		case "obsidian":
			targetURI = links.Obsidian
		default:
			targetURI = links.Obsidian
		}

		statusMsg := ""
		success := true

		// 3. Execução se solicitado modo 'open'
		if action == "open" {
			openedURI, err := deeplink.Open(resolvedPath, app, line, repoRoot, vaultName, launcher)
			if err != nil {
				success = false
				statusMsg = fmt.Sprintf("⚠️ Falha ao abrir no aplicativo '%s': %v", app, err)
			} else {
				targetURI = openedURI
				statusMsg = fmt.Sprintf("🚀 Nota disparada para abertura no %s com sucesso!", app)
			}
		} else {
			statusMsg = "Links gerados para uso imediato pelo agente ou usuário."
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### 🔗 Navegação e Deep Links: `%s`\n\n", nodeID))
		sb.WriteString(fmt.Sprintf("- **Caminho Resolvido**: `%s`\n", resolvedPath))
		sb.WriteString(fmt.Sprintf("- **Aplicativo Alvo**: `%s`\n", app))
		if line > 0 {
			sb.WriteString(fmt.Sprintf("- **Linha**: `%d`\n", line))
		}
		sb.WriteString(fmt.Sprintf("- **URI Principal**: `%s`\n", targetURI))
		sb.WriteString(fmt.Sprintf("- **Ação**: `%s`\n", action))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n\n", statusMsg))

		sb.WriteString("#### Deep Links Disponíveis:\n")
		sb.WriteString(fmt.Sprintf("- **Obsidian**: `%s`\n", links.Obsidian))
		sb.WriteString(fmt.Sprintf("- **VS Code**:  `%s`\n", links.VSCode))
		sb.WriteString(fmt.Sprintf("- **Arquivo**:  `%s`\n", links.File))

		_ = success
		return NewTextResult(sb.String()), nil
	}
}

// SetOpenHandler registra ou atualiza o executor de open_node no servidor MCP
func (s *Server) SetOpenHandler(resolveFn OpenNodeFunc, repoRoot string, vaultName string, launcher deeplink.Launcher) {
	s.RegisterTool(ToolMemoryOpenNode, NewMemoryOpenNodeHandler(resolveFn, repoRoot, vaultName, launcher))
}
