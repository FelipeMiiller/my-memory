package embedder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OllamaClient struct {
	BaseURL string
	Model   string
	client  *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// GenerateEmbedding gera o vetor de 768 dimensões usando a API do Ollama
func (o *OllamaClient) GenerateEmbedding(text string) ([]float32, error) {
	reqBody, err := json.Marshal(embedRequest{
		Model: o.Model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	resp, err := o.client.Post(o.BaseURL+"/api/embed", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no Ollama (%s): %w", o.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama retornou status HTTP %d", resp.StatusCode)
	}

	var res embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta do Ollama: %w", err)
	}

	if len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("nenhum vetor retornado pelo modelo")
	}

	return res.Embeddings[0], nil
}
