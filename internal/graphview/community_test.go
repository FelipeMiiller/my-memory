package graphview

import (
	"testing"
)

func TestBuildGraphView_CommunityEnrichment(t *testing.T) {
	docs := []RawDoc{
		{ID: "concept/auth", Title: "Authentication Concept"},
		{ID: "decision/jwt", Title: "Use JWT for Session"},
		{ID: "guide/auth", Title: "Auth Integration Guide"},
		{ID: "concept/db", Title: "Database Architecture"},
		{ID: "decision/postgres", Title: "PostgreSQL Database"},
	}

	edges := []RawEdge{
		// Cluster 1 (Auth)
		{Source: "concept/auth", Target: "decision/jwt", EpistemicStatus: "EXTRACTED", Weight: 1.0},
		{Source: "decision/jwt", Target: "guide/auth", EpistemicStatus: "EXTRACTED", Weight: 1.0},
		{Source: "guide/auth", Target: "concept/auth", EpistemicStatus: "EXTRACTED", Weight: 1.0},

		// Cluster 2 (DB)
		{Source: "concept/db", Target: "decision/postgres", EpistemicStatus: "EXTRACTED", Weight: 1.0},
	}

	gv := BuildGraphView(docs, edges, "", 0, "test-repo")

	if gv == nil {
		t.Fatal("BuildGraphView retornou nil")
	}

	if gv.Stats.CommunityCount != 2 {
		t.Fatalf("esperava 2 comunidades no GraphStats, obteve %d", gv.Stats.CommunityCount)
	}

	if len(gv.Communities) != 2 {
		t.Fatalf("esperava 2 comunidades no gv.Communities, obteve %d", len(gv.Communities))
	}

	// Verificar se todos os nós receberam CommunityID > 0 e cores de comunidade válidas
	nodeCommMap := make(map[string]int)
	for _, n := range gv.Nodes {
		if n.CommunityID <= 0 {
			t.Errorf("nó %s não possui CommunityID válido: %d", n.ID, n.CommunityID)
		}
		if n.CommunityLabel == "" {
			t.Errorf("nó %s não possui CommunityLabel", n.ID)
		}
		if n.CommunityColor == "" || n.CommunityColor == "#64748b" {
			t.Errorf("nó %s não possui cor de comunidade categórica válida: %s", n.ID, n.CommunityColor)
		}
		nodeCommMap[n.ID] = n.CommunityID
	}

	// Nós do grupo Auth devem compartilhar o mesmo CommunityID
	authID := nodeCommMap["concept/auth"]
	if nodeCommMap["decision/jwt"] != authID || nodeCommMap["guide/auth"] != authID {
		t.Errorf("nós de auth deveriam ter o mesmo CommunityID, obteve: auth=%d, jwt=%d, guide=%d",
			authID, nodeCommMap["decision/jwt"], nodeCommMap["guide/auth"])
	}

	// Nós do grupo DB devem compartilhar o mesmo CommunityID, e ser diferente de Auth
	dbID := nodeCommMap["concept/db"]
	if nodeCommMap["decision/postgres"] != dbID {
		t.Errorf("nós de db deveriam ter o mesmo CommunityID, obteve: db=%d, pg=%d",
			dbID, nodeCommMap["decision/postgres"])
	}
	if dbID == authID {
		t.Errorf("clusters desconectados Auth e DB não deveriam ter o mesmo ID: %d", authID)
	}
}

func TestGetColorForCommunity(t *testing.T) {
	c1 := GetColorForCommunity(1)
	c2 := GetColorForCommunity(2)
	c3 := GetColorForCommunity(1) // determinístico
	c0 := GetColorForCommunity(0) // caso limite neutro

	if c1 == "" || c2 == "" {
		t.Fatal("cores de comunidade não devem ser vazias")
	}
	if c1 != c3 {
		t.Errorf("GetColorForCommunity deve ser determinístico: %s != %s", c1, c3)
	}
	if c1 == c2 {
		t.Errorf("comunidades consecutivas devem ter cores diferentes: %s == %s", c1, c2)
	}
	if c0 != "#64748b" {
		t.Errorf("comunidade 0 deve retornar cinza neutro, obteve %s", c0)
	}
}

func TestFindCommunityLeader(t *testing.T) {
	members := []string{"node-A", "node-B", "node-C"}
	prScores := map[string]float64{
		"node-A": 0.1,
		"node-B": 0.6,
		"node-C": 0.3,
	}

	leader := FindCommunityLeader(members, prScores)
	if leader != "node-B" {
		t.Errorf("esperava node-B como líder pelo PageRank (0.6), obteve: %s", leader)
	}

	// Caso de empate: desempate lexicográfico
	prTied := map[string]float64{
		"node-C": 0.5,
		"node-A": 0.5,
	}
	leaderTied := FindCommunityLeader([]string{"node-C", "node-A"}, prTied)
	if leaderTied != "node-A" {
		t.Errorf("esperava node-A no desempate lexicográfico, obteve: %s", leaderTied)
	}
}

func TestFindDominantType(t *testing.T) {
	members := []string{"n1", "n2", "n3", "n4"}
	types := map[string]string{
		"n1": "decision",
		"n2": "concept",
		"n3": "decision",
		"n4": "guide",
	}

	dominant := FindDominantType(members, types)
	if dominant != "decision" {
		t.Errorf("esperava dominantType 'decision' (2 ocorrências), obteve: %s", dominant)
	}
}
