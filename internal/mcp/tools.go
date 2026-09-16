package mcp

import (
	"context"
	"encoding/json"
)

// ListToolsResult representa a resposta da listagem de ferramentas MCP
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// Schemas das ferramentas expostas pelo My-Memory
var (
	ToolMemorySearch = Tool{
		Name:        "memory_search",
		Description: "Busca híbrida com Reciprocal Rank Fusion (RRF) combinando texto exato FTS5/tsvector, vetores k-NN e expansão de grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Texto ou pergunta para busca na base de memória de notas",
				},
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"hybrid", "vector", "fts"},
					"description": "Modo de busca: 'hybrid' (padrão, RRF unificando FTS + vetores + grafo), 'vector' (apenas semântico k-NN) ou 'fts' (apenas texto exato)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Número máximo de resultados a retornar (padrão: 5)",
				},
				"k": map[string]any{
					"type":        "integer",
					"description": "Constante de suavização do algoritmo RRF (padrão: 60)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para filtrar a busca (opcional). Se omitido, busca no repositório padrão ou global.",
				},
				"decay": map[string]any{
					"type":        "boolean",
					"description": "Ativa o decaimento temporal exponencial para priorizar notas mais recentes na busca híbrida (padrão: false)",
				},
				"half_life": map[string]any{
					"type":        "number",
					"description": "Tempo de meia-vida em dias para a curva de decaimento temporal (padrão: 30.0)",
				},
				"decay_weight": map[string]any{
					"type":        "number",
					"description": "Peso do fator temporal entre 0.0 (sem efeito) e 1.0 (decaimento máximo) (padrão: 0.3)",
				},
				"detail_level": map[string]any{
					"type":        "string",
					"enum":        []string{"l0", "l1", "l2"},
					"description": "Nível de densidade de contexto (Progressive Context Loading): 'l0' (micro-abstract cirúrgico, ~30-50 tokens, sem blocos de texto bruto - Zero File Reads), 'l1' (overview padrão com metadados e trecho relevante) ou 'l2' (detalhes completos)",
				},
				"category": map[string]any{
					"type":        "string",
					"enum":        []string{"resource", "memory", "skill"},
					"description": "Filtra por taxonomia de conhecimento: 'resource' (especificações técnicas, arquitetura), 'memory' (regras de conduta, lições aprendidas, padrões) ou 'skill' (habilidades, comandos operacionais, procedimentos)",
				},
			},
			"required": []string{"query"},
		},
	}

	ToolMemoryGetNeighbors = Tool{
		Name:        "memory_get_neighbors",
		Description: "Retorna nós e notas vizinhas no grafo relacional a partir de um node_id via CTE recursivo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador do nó (caminho do arquivo ou título da nota)",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de travessia no grafo (padrão: 1)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a travessia de vizinhos (opcional).",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Formato de saída: 'text' (padrão) ou 'json' estruturado com sinalização federada (opcional)",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryExportCanvas = Tool{
		Name:        "memory_export_canvas",
		Description: "Exporta um subgrafo centrado em node_id no formato aberto JSON Canvas 1.0 (.canvas) para visualização espacial no Obsidian",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador ou nome da nota central do subgrafo",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de travessia no grafo (padrão: 1)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a busca de conexões (opcional)",
				},
				"output_path": map[string]any{
					"type":        "string",
					"description": "Caminho de arquivo para salvar o .canvas (opcional). Se omitido, retorna a string JSON do Canvas.",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryGetHubs = Tool{
		Name:        "memory_get_hubs",
		Description: "Retorna os nós com maior centralidade de conexões (God Nodes / Hubs de conhecimento) no grafo de notas",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"top": map[string]any{
					"type":        "integer",
					"description": "Número máximo de nós centrais a retornar (padrão: 10)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a busca de hubs (opcional)",
				},
				"algorithm": map[string]any{
					"type":        "string",
					"enum":        []string{"degree", "pagerank"},
					"description": "Algoritmo de centralidade: 'degree' (grau total in+out) ou 'pagerank' (autoridade estrutural iterativa ponderada). Padrão: 'degree'",
				},
			},
		},
	}

	ToolMemoryGetInsights = Tool{
		Name:        "memory_get_insights",
		Description: "Retorna conexões latentes e surpreendentes (Surprising Connections) entre notas com alta similaridade sem links diretos no grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "integer",
					"description": "Número máximo de conexões latentes a retornar (padrão: 10)",
				},
				"min_similarity": map[string]any{
					"type":        "number",
					"description": "Limiar mínimo de similaridade semântica entre as notas (padrão: 0.70)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para filtrar as conexões inesperadas (opcional)",
				},
			},
		},
	}

	ToolMemoryDoctor = Tool{
		Name:        "memory_doctor",
		Description: "Audita a integridade do grafo e tabelas de notas, identificando dead links, notas órfãs, self-loops e calculando o Health Score",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar o diagnóstico (opcional)",
				},
				"fix": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, remove automaticamente anomalias conhecidas (self-loops e dead links) (padrão: false)",
				},
			},
		},
	}

	ToolMemoryWriteNote = Tool{
		Name:        "memory_write_note",
		Description: "Grava ou substitui uma nota atômica em Markdown no vault com frontmatter estruturado e conexões tipadas, disparando indexação cirúrgica imediata no grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Caminho relativo do arquivo Markdown dentro do vault (ex: 'concepts/auth.md')",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Título principal da nota (opcional, derivado do nome do arquivo se omitido)",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Conteúdo textual da nota em Markdown",
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Lista de tags conceituais para a nota (opcional)",
				},
				"aliases": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Lista de títulos alternativos ou apelidos (opcional)",
				},
				"note_type": map[string]any{
					"type":        "string",
					"description": "Classificação da nota: 'concept', 'decision', 'summary', 'entity' (padrão: 'concept')",
				},
				"relations": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"target": map[string]any{
								"type":        "string",
								"description": "Nome da nota de destino conectada",
							},
							"relation": map[string]any{
								"type":        "string",
								"description": "Tipo da relação semântica (ex: 'implements', 'depends_on', 'supports', 'links_to')",
							},
						},
						"required": []string{"target"},
					},
					"description": "Lista de arestas tipadas direcionadas para outras notas (opcional)",
				},
				"overwrite": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, sobrescreve arquivo existente caso já exista (padrão: false)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"path", "content"},
		},
	}

	ToolMemoryAppendSection = Tool{
		Name:        "memory_append_section",
		Description: "Anexa cirurgicamente um novo bloco de texto sob um cabeçalho existente ou ao final da nota Markdown no vault, atualizando o índice imediatamente",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Caminho do arquivo Markdown dentro do vault",
				},
				"heading": map[string]any{
					"type":        "string",
					"description": "Título da seção sob a qual anexar o conteúdo (ex: '## Decisões Recentes')",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Conteúdo a ser anexado sob a seção",
				},
				"create_if_missing": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, cria o arquivo se ele ainda não existir (padrão: true)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"path", "heading", "content"},
		},
	}

	ToolMemoryCompileNote = Tool{
		Name:        "memory_compile_note",
		Description: "Compila e sintetiza conhecimento sobre um tópico a partir de buscas híbridas no repositório (padrão Compile-not-Retrieve / Karpathy LLM Wiki), gravando nota estruturada com backlinks",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic": map[string]any{
					"type":        "string",
					"description": "Tópico ou conceito a ser investigado e compilado (ex: 'fluxo de autenticação')",
				},
				"target_path": map[string]any{
					"type":        "string",
					"description": "Caminho relativo de destino onde a nota compilada será gravada (ex: 'syntheses/auth.md')",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Título da nota compilada (opcional)",
				},
				"search_mode": map[string]any{
					"type":        "string",
					"enum":        []string{"hybrid", "vector", "fts"},
					"description": "Modo de recuperação dos fragmentos fonte (padrão: 'hybrid')",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Quantidade máxima de fragmentos a sintetizar (padrão: 5)",
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Tags adicionais a incluir no frontmatter da nota compilada (opcional)",
				},
				"overwrite": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, sobrescreve arquivo existente (padrão: false)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"topic", "target_path"},
		},
	}

	ToolMemoryVisualizeGraph = Tool{
		Name:        "memory_visualize_graph",
		Description: "Exporta uma visualização interativa do grafo da memória para uma página HTML standalone com física de forças, busca e PageRank",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"root_node": map[string]any{
					"type":        "string",
					"description": "Identificador da nota raiz para isolar um subgrafo local. Se omitido, exporta o grafo global da memória.",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de conexões para subgrafos focados (padrão: 2)",
				},
				"output_path": map[string]any{
					"type":        "string",
					"description": "Caminho do arquivo HTML de saída (padrão: graph.html ou graph_<nota>.html)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
		},
	}

	ToolMemoryGetClusters = Tool{
		Name:        "memory_get_clusters",
		Description: "Detecta partições temáticas e comunidades no grafo relacional usando Weighted Label Propagation e calcula a Modularidade Newman-Girvan Q",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"min_size": map[string]any{
					"type":        "integer",
					"description": "Tamanho mínimo de nós para incluir um cluster no resultado (padrão: 2)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
		},
	}

	ToolMemoryGetImpact = Tool{
		Name:        "memory_get_impact",
		Description: "Calcula a análise de impacto e raio de destruição (blast radius) de dependências reversas antes de alterações em notas ou código",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho de arquivo do nó alvo no grafo",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de propagação das dependências reversas (padrão: 2)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryInspectNode = Tool{
		Name:        "memory_inspect_node",
		Description: "Inspeção cirúrgica em 3 colunas (Triptych) de um nó no grafo: chamadores/dependentes (inbound), núcleo do nó e risco de blast radius, e referências de saída (outbound) sem leitura manual de arquivos",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho de arquivo do nó alvo a inspecionar",
				},
				"max_content_length": map[string]any{
					"type":        "integer",
					"description": "Tamanho máximo do preview de conteúdo textual da nota (padrão: 500 caracteres, 0 para ilimitado)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryFindPath = Tool{
		Name:        "memory_find_path",
		Description: "Encontra o menor caminho e traça a rota ponderada entre dois nós no grafo de conhecimento, considerando certeza epistêmica (EXTRACTED vs INFERRED vs TAG) ou número de saltos",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho de arquivo do nó de origem",
				},
				"target": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho de arquivo do nó de destino",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de saltos a explorar (padrão: 6)",
				},
				"directed": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, navega apenas no sentido direcionado das arestas (A -> B). Se falso, navega bidirecionalmente (padrão: true)",
				},
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"epistemic", "hops"},
					"description": "Modo de custo: 'epistemic' (padrão, prioriza arestas com maior certeza documental) ou 'hops' (menor quantidade de arestas)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"source", "target"},
		},
	}

	ToolMemoryPackContext = Tool{
		Name:        "memory_pack_context",
		Description: "Extrai e consolida um subgrafo conexo centrado em uma nota raiz, empacotando documentos centrais (L2), resumos periféricos (L0/L1) e diagrama Mermaid em um bundle coerente com limite de tokens",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"root_node": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho da nota raiz a partir da qual o subgrafo será explorado",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de saltos de exploração no grafo (padrão: 2)",
				},
				"max_tokens": map[string]any{
					"type":        "integer",
					"description": "Orçamento máximo de tokens para o pacote gerado (padrão: 4000)",
				},
				"direction": map[string]any{
					"type":        "string",
					"enum":        []string{"both", "outbound", "inbound"},
					"description": "Direção da navegação pelas arestas ('both', 'outbound', 'inbound', padrão: 'both')",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
			},
			"required": []string{"root_node"},
		},
	}

	ToolMemoryOpenNode = Tool{
		Name:        "memory_open_node",
		Description: "Gera deep links canônicos (Obsidian, VS Code, File) e opcionalmente abre o documento no editor indicado",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador, título ou caminho do arquivo da nota a ser aberta ou linkada",
				},
				"app": map[string]any{
					"type":        "string",
					"enum":        []string{"obsidian", "vscode", "system"},
					"description": "Aplicativo alvo para navegação: 'obsidian' (padrão), 'vscode' ou 'system'",
				},
				"line": map[string]any{
					"type":        "integer",
					"description": "Número de linha para focar no editor (opcional)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório (opcional)",
				},
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"links_only", "open"},
					"description": "Modo de execução: 'links_only' (retorna apenas as URIs acionáveis, padrão seguro) ou 'open' (dispara a abertura do processo no SO)",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryGetDrift = Tool{
		Name:        "memory_get_drift",
		Description: "Analisa o desvio semântico entre alterações recentes de código-fonte no Git e as notas/ADRs da base de memória, identificando documentação defasada e código órfão sem decisões arquiteturais vinculadas",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"since": map[string]any{
					"type":        "string",
					"description": "Faixa de commits do Git para análise (ex: 'HEAD~5..HEAD', 'commit..HEAD' ou 'HEAD~10'). Se omitido, analisa os últimos 5 commits.",
				},
				"threshold": map[string]any{
					"type":        "number",
					"description": "Limite mínimo de score de drift normalizado (0.0 a 1.0) para incluir notas no relatório (padrão: 0.20)",
				},
				"include_uncovered": map[string]any{
					"type":        "boolean",
					"description": "Indica se deve incluir arquivos de código modificados que não possuem notas associadas (padrão: true)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou identificador do repositório para contextualizar a análise (opcional)",
				},
			},
		},
	}
)

// RegisterTool adiciona ou atualiza uma ferramenta e seu respectivo handler no servidor
func (s *Server) RegisterTool(tool Tool, handler ToolHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, t := range s.tools {
		if t.Name == tool.Name {
			s.tools[i] = tool
			found = true
			break
		}
	}
	if !found {
		s.tools = append(s.tools, tool)
	}

	if handler != nil {
		s.toolHandlers[tool.Name] = handler
	}
}

// RegisterToolHandler registra apenas a função executora de uma ferramenta
func (s *Server) RegisterToolHandler(name string, handler ToolHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolHandlers[name] = handler
}

// GetTools retorna uma cópia de todas as ferramentas registradas
func (s *Server) GetTools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, len(s.tools))
	copy(tools, s.tools)
	return tools
}

// handleToolsList responde à requisição tools/list retornando o catálogo de ferramentas registradas
func (s *Server) handleToolsList(ctx context.Context, params json.RawMessage) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ListToolsResult{
		Tools: s.tools,
	}, nil
}
