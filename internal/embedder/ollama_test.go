package embedder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOllamaClient_Defaults(t *testing.T) {
	client := NewOllamaClient("", "")
	if client.BaseURL != "http://localhost:11434" {
		t.Errorf("esperado baseURL padrão 'http://localhost:11434', obteve %q", client.BaseURL)
	}
	if client.Model != "nomic-embed-text" {
		t.Errorf("esperado modelo padrão 'nomic-embed-text', obteve %q", client.Model)
	}
	if client.client == nil {
		t.Errorf("cliente http não deve ser nulo")
	}

	customClient := NewOllamaClient("http://custom:1234", "custom-model")
	if customClient.BaseURL != "http://custom:1234" {
		t.Errorf("esperado baseURL customizado, obteve %q", customClient.BaseURL)
	}
	if customClient.Model != "custom-model" {
		t.Errorf("esperado modelo customizado, obteve %q", customClient.Model)
	}
}

func TestGenerateEmbedding_Success(t *testing.T) {
	expectedVec := []float32{0.1, 0.2, 0.3, -0.4}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("esperado método POST, obteve %s", r.Method)
		}
		if r.URL.Path != "/api/embed" {
			t.Errorf("esperado caminho /api/embed, obteve %s", r.URL.Path)
		}

		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("erro ao decodificar requisição: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("esperado modelo 'test-model', obteve %q", req.Model)
		}
		if req.Input != "teste de embedding" {
			t.Errorf("esperado input 'teste de embedding', obteve %q", req.Input)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := embedResponse{
			Embeddings: [][]float32{expectedVec},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-model")
	vec, err := client.GenerateEmbedding("teste de embedding")
	if err != nil {
		t.Fatalf("GenerateEmbedding falhou inesperadamente: %v", err)
	}

	if len(vec) != len(expectedVec) {
		t.Fatalf("esperado vetor de tamanho %d, obteve %d", len(expectedVec), len(vec))
	}
	for i := range expectedVec {
		if vec[i] != expectedVec[i] {
			t.Errorf("posição %d: esperado %f, obteve %f", i, expectedVec[i], vec[i])
		}
	}
}

func TestGenerateEmbedding_HttpStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-model")
	_, err := client.GenerateEmbedding("texto")
	if err == nil {
		t.Fatal("esperava erro para HTTP 500, obteve nil")
	}
}

func TestGenerateEmbedding_EmptyEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(embedResponse{Embeddings: [][]float32{}})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-model")
	_, err := client.GenerateEmbedding("texto")
	if err == nil {
		t.Fatal("esperava erro para array de embeddings vazio, obteve nil")
	}
}

func TestGenerateEmbedding_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{invalid-json"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-model")
	_, err := client.GenerateEmbedding("texto")
	if err == nil {
		t.Fatal("esperava erro de decodificação JSON, obteve nil")
	}
}

func TestGenerateEmbedding_ConnectionError(t *testing.T) {
	client := NewOllamaClient("http://127.0.0.1:54321", "test-model")
	_, err := client.GenerateEmbedding("texto")
	if err == nil {
		t.Fatal("esperava erro de conexão de rede, obteve nil")
	}
}
