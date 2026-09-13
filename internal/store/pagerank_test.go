package store

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestPageRankNode_Serialization(t *testing.T) {
	node := PageRankNode{
		ID:        "doc-arquitetura",
		Name:      "Arquitetura do Sistema",
		Score:     0.4523,
		Rank:      1,
		InDegree:  12,
		OutDegree: 4,
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Erro ao serializar PageRankNode: %v", err)
	}

	var decoded PageRankNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Erro ao deserializar PageRankNode: %v", err)
	}

	if decoded.ID != "doc-arquitetura" || decoded.Rank != 1 || decoded.Score != 0.4523 {
		t.Errorf("Dados de PageRankNode incorretos: %+v", decoded)
	}
	if decoded.InDegree != 12 || decoded.OutDegree != 4 {
		t.Errorf("Graus de entrada/saída incorretos: %+v", decoded)
	}
}

func TestPageRankNode_Sorting(t *testing.T) {
	nodes := []PageRankNode{
		{ID: "B", Score: 0.20, InDegree: 2},
		{ID: "A", Score: 0.50, InDegree: 5},
		{ID: "C", Score: 0.30, InDegree: 3},
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Score != nodes[j].Score {
			return nodes[i].Score > nodes[j].Score
		}
		return nodes[i].InDegree > nodes[j].InDegree
	})

	for i := range nodes {
		nodes[i].Rank = i + 1
	}

	if nodes[0].ID != "A" || nodes[0].Rank != 1 {
		t.Errorf("Primeiro nó deveria ser A (rank 1), obteve: %+v", nodes[0])
	}
	if nodes[1].ID != "C" || nodes[1].Rank != 2 {
		t.Errorf("Segundo nó deveria ser C (rank 2), obteve: %+v", nodes[1])
	}
	if nodes[2].ID != "B" || nodes[2].Rank != 3 {
		t.Errorf("Terceiro nó deveria ser B (rank 3), obteve: %+v", nodes[2])
	}
}
