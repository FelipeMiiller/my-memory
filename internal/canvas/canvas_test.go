package canvas

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if len(id1) != 16 {
		t.Errorf("ID gerado deve ter 16 caracteres hex, obteve %d (%s)", len(id1), id1)
	}
	if id1 == id2 {
		t.Errorf("IDs consecutivos não devem colidir: %s == %s", id1, id2)
	}
}

func TestFromNeighbors(t *testing.T) {
	center := "Arquitetura"
	neighbors := []string{"Autenticacao", "Banco de Dados", "Cache Redis", "Fila RabbitMQ"}

	c := FromNeighbors(center, neighbors)

	// Deve ter 1 nó central + 4 vizinhos = 5 nós
	if len(c.Nodes) != 5 {
		t.Fatalf("Esperava 5 nós, obteve %d", len(c.Nodes))
	}

	// Deve ter 4 arestas (1 do centro para cada vizinho)
	if len(c.Edges) != 4 {
		t.Fatalf("Esperava 4 arestas, obteve %d", len(c.Edges))
	}

	// Verifica se todos os nós têm IDs únicos
	nodeIDs := make(map[string]bool)
	for _, n := range c.Nodes {
		if nodeIDs[n.ID] {
			t.Errorf("ID de nó duplicado encontrado: %s", n.ID)
		}
		nodeIDs[n.ID] = true
	}

	// Verifica se as arestas apontam para nós que realmente existem (integridade referencial)
	for _, e := range c.Edges {
		if !nodeIDs[e.FromNode] {
			t.Errorf("fromNode %s não existe na lista de nós", e.FromNode)
		}
		if !nodeIDs[e.ToNode] {
			t.Errorf("toNode %s não existe na lista de nós", e.ToNode)
		}
		if e.ToEnd != "arrow" {
			t.Errorf("Esperava toEnd='arrow', obteve %s", e.ToEnd)
		}
	}

	// Valida serialização JSON
	data, err := c.ToJSON()
	if err != nil {
		t.Fatalf("Erro ao serializar JSON Canvas: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON gerado inválido: %v", err)
	}

	if _, ok := parsed["nodes"]; !ok {
		t.Errorf("Chave 'nodes' ausente no JSON gerado")
	}
	if _, ok := parsed["edges"]; !ok {
		t.Errorf("Chave 'edges' ausente no JSON gerado")
	}
}

func TestSaveToFile(t *testing.T) {
	c := FromNeighbors("NotaCentral", []string{"Vizinho1"})

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.canvas")

	if err := c.SaveToFile(filePath); err != nil {
		t.Fatalf("Falha ao salvar canvas: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Falha ao ler arquivo salvo: %v", err)
	}

	if !strings.Contains(string(content), "NotaCentral.md") {
		t.Errorf("Conteúdo não contém o nó esperado: %s", string(content))
	}
}
