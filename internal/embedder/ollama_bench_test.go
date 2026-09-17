package embedder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeEmbeddingResponse gera uma resposta JSON válida do Ollama com vetor de N dimensões
func fakeEmbeddingResponse(dim int) string {
	vec := make([]string, dim)
	for i := range vec {
		vec[i] = fmt.Sprintf("0.%d", i%10)
	}
	return fmt.Sprintf(`{"embeddings":[[%s]]}`, strings.Join(vec, ","))
}

// BenchmarkGenerateEmbedding_Mocked mede a latência end-to-end com servidor Ollama mockado
func BenchmarkGenerateEmbedding_Mocked(b *testing.B) {
	dim := 768
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fakeEmbeddingResponse(dim)))
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, "nomic-embed-text")
	input := "Texto de exemplo em português: memória, síntese, usuário, ação, opção. " +
		"Padded com palavras pra ficar realista (~100 chars)."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vec, err := client.GenerateEmbedding(input)
		if err != nil {
			b.Fatalf("GenerateEmbedding: %v", err)
		}
		if len(vec) != dim {
			b.Fatalf("dim mismatch: got %d, want %d", len(vec), dim)
		}
	}
}

// BenchmarkEmbedRequestMarshal mede o custo de serializar o request body
func BenchmarkEmbedRequestMarshal(b *testing.B) {
	req := embedRequest{
		Model: "nomic-embed-text",
		Input: "Texto de exemplo em português: memória, síntese, usuário, ação, opção.",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatalf("Marshal: %v", err)
		}
	}
}

// BenchmarkEmbedResponseUnmarshal mede o custo de decodificar a resposta
func BenchmarkEmbedResponseUnmarshal(b *testing.B) {
	raw := []byte(fakeEmbeddingResponse(768))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res embedResponse
		if err := json.Unmarshal(raw, &res); err != nil {
			b.Fatalf("Unmarshal: %v", err)
		}
	}
}