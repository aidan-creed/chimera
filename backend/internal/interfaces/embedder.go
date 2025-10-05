package interfaces

import "context"

// EmbedderFunc defines the signature for any function that can generate embeddings
type EmbedderFunc func(ctx context.Context, text string) ([]float32, error)
