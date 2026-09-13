package canvas

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// NodeType define os tipos suportados pela especificação JSON Canvas 1.0
type NodeType string

const (
	NodeText  NodeType = "text"
	NodeFile  NodeType = "file"
	NodeLink  NodeType = "link"
	NodeGroup NodeType = "group"
)

// Node representa um nó na tela do JSON Canvas
type Node struct {
	ID     string   `json:"id"`
	Type   NodeType `json:"type"`
	X      int      `json:"x"`
	Y      int      `json:"y"`
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Color  string   `json:"color,omitempty"`

	// Propriedades específicas de cada tipo
	Text    string `json:"text,omitempty"`
	File    string `json:"file,omitempty"`
	Subpath string `json:"subpath,omitempty"`
	URL     string `json:"url,omitempty"`
	Label   string `json:"label,omitempty"`
}

// Edge representa uma conexão direcionada entre nós no JSON Canvas
type Edge struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	ToNode   string `json:"toNode"`
	FromSide string `json:"fromSide,omitempty"` // "top", "right", "bottom", "left"
	ToSide   string `json:"toSide,omitempty"`   // "top", "right", "bottom", "left"
	FromEnd  string `json:"fromEnd,omitempty"`  // "none", "arrow"
	ToEnd    string `json:"toEnd,omitempty"`    // "none", "arrow" (padrão: "arrow")
	Color    string `json:"color,omitempty"`
	Label    string `json:"label,omitempty"`
}

// Canvas é o contêiner raiz de acordo com a especificação JSON Canvas 1.0
type Canvas struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// NewCanvas cria uma instância vazia com slices inicializados
func NewCanvas() *Canvas {
	return &Canvas{
		Nodes: make([]Node, 0),
		Edges: make([]Edge, 0),
	}
}

// GenerateID gera um identificador hexadecimal aleatório de 16 caracteres
func GenerateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback determinístico caso o gerador de entropia falhe
		return "a1b2c3d4e5f67890"
	}
	return hex.EncodeToString(bytes)
}

// AddTextNode adiciona um nó com conteúdo Markdown
func (c *Canvas) AddTextNode(text string, x, y, width, height int, color string) Node {
	n := Node{
		ID:     GenerateID(),
		Type:   NodeText,
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Text:   text,
		Color:  color,
	}
	c.Nodes = append(c.Nodes, n)
	return n
}

// AddFileNode adiciona um nó vinculado a um arquivo do vault Obsidian
func (c *Canvas) AddFileNode(file string, x, y, width, height int, color string) Node {
	filePath := file
	if !strings.HasSuffix(strings.ToLower(filePath), ".md") && !strings.Contains(filePath, ".") {
		filePath += ".md"
	}

	n := Node{
		ID:     GenerateID(),
		Type:   NodeFile,
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		File:   filePath,
		Color:  color,
	}
	c.Nodes = append(c.Nodes, n)
	return n
}

// AddEdge conecta dois nós com uma aresta direcionada
func (c *Canvas) AddEdge(fromID, toID, fromSide, toSide, label, color string) Edge {
	e := Edge{
		ID:       GenerateID(),
		FromNode: fromID,
		ToNode:   toID,
		FromSide: fromSide,
		ToSide:   toSide,
		ToEnd:    "arrow",
		Label:    label,
		Color:    color,
	}
	c.Edges = append(c.Edges, e)
	return e
}

// FromNeighbors gera um Canvas radial centrado em centerNode conectado a seus vizinhos
func FromNeighbors(centerNode string, neighbors []string) *Canvas {
	c := NewCanvas()

	// 1. Nó central (destaque em cor preset "1" - geralmente vermelho/coral no Obsidian)
	centerW := 300
	centerH := 120
	center := c.AddFileNode(centerNode, 0, 0, centerW, centerH, "1")

	n := len(neighbors)
	if n == 0 {
		return c
	}

	// 2. Disposição radial dos vizinhos (raio calculado para evitar sobreposição)
	radius := 420.0
	if n > 8 {
		radius = 550.0
	}
	if n > 16 {
		radius = 700.0
	}

	neighborW := 260
	neighborH := 90

	for i, neighbor := range neighbors {
		angle := (2.0 * math.Pi * float64(i)) / float64(n)

		x := int(math.Round(radius * math.Cos(angle)))
		y := int(math.Round(radius * math.Sin(angle)))

		// Centraliza o nó no ponto calculado
		nodeX := x - (neighborW / 2)
		nodeY := y - (neighborH / 2)

		// Preset de cor "4" (verde/menta no Obsidian)
		neighborNode := c.AddFileNode(neighbor, nodeX, nodeY, neighborW, neighborH, "4")

		// Define lados de conexão baseados na posição relativa
		fromSide := "right"
		toSide := "left"
		cosVal := math.Cos(angle)
		sinVal := math.Sin(angle)

		if cosVal > 0.5 {
			fromSide = "right"
			toSide = "left"
		} else if cosVal < -0.5 {
			fromSide = "left"
			toSide = "right"
		} else if sinVal > 0 {
			fromSide = "bottom"
			toSide = "top"
		} else {
			fromSide = "top"
			toSide = "bottom"
		}

		c.AddEdge(center.ID, neighborNode.ID, fromSide, toSide, "conecta", "")
	}

	return c
}

// ToJSON serializa o Canvas no formato JSON padronizado com indentação
func (c *Canvas) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// SaveToFile salva o conteúdo do Canvas em um arquivo .canvas
func (c *Canvas) SaveToFile(filePath string) error {
	data, err := c.ToJSON()
	if err != nil {
		return fmt.Errorf("falha ao serializar json canvas: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}
