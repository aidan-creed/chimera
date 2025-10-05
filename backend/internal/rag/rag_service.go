// backend/internal/rag/rag_service.go
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// RAGService provides shared utilities for the RAG platform components.
type RAGService struct {
	httpClient          *http.Client
	embeddingServiceURL string
	AIAPIKey            string
	LLM_URL             string
	logger              *slog.Logger
}

// NewRAGService creates a new instance of the RAGService.
func NewRAGService(embeddingURL string, AIKey string, LLM_URL string, logger *slog.Logger) *RAGService {
	return &RAGService{
		httpClient:          &http.Client{Timeout: 90 * time.Second},
		embeddingServiceURL: embeddingURL,
		AIAPIKey:            AIKey,
		LLM_URL:             LLM_URL,
		logger:              logger.With("component", "rag_service"),
	}
}

// EmbeddingRequest defines the structure for calling the embedding service.
type EmbeddingRequest struct {
	Text string `json:"text"`
}

// EmbeddingResponse defines the structure for the embedding service's response.
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type LLMRequestBody struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GetEmbedding is the single, platform-wide method for generating embeddings.
func (s *RAGService) GetEmbedding(ctx context.Context, textToEmbed string) ([]float32, error) {
	reqBody, err := json.Marshal(EmbeddingRequest{Text: textToEmbed})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.embeddingServiceURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}
	return embeddingResp.Embedding, nil
}

// CallLLM is the centralized method for making requests to the AI Chat Completions API.
func (s *RAGService) CallLLM(ctx context.Context, prompt string, useJSONMode bool) (string, error) {
	if s.AIAPIKey == "" {
		return "", fmt.Errorf("AI API key is not configured")
	}

	// 1. Construct the request body for the OpenAI API.
	requestBody := LLMRequestBody{
		Model: "gpt-4o", // This can be made configurable later
		Messages: []ChatMessage{
			{Sender: "user", Content: prompt},
		},
	}
	if useJSONMode {
		requestBody.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	payloadBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	// 2. Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, "POST", s.LLM_URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create AI request: %w", err)
	}

	// 3. Set the required headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.AIAPIKey)

	// 4. Execute the request.
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call AI API: %w", err)
	}
	defer resp.Body.Close()

	// 5. Handle non-successful status codes.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI API returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 6. Decode the successful response.
	var llmResponse LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", fmt.Errorf("failed to decode AI response: %w", err)
	}

	if len(llmResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from AI")
	}

	// 7. Return the content of the first message.
	return llmResponse.Choices[0].Message.Content, nil
}
