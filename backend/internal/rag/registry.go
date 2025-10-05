// backend/internal/rag/registry.go
package rag

import (
	"context"
	"fmt"
	"text/template"
)

// ToolFunc defines the signature for any function that can be used as a tool by the RAG agent.
// It accepts a map of queriers and a map of arguments from the LLM planner.
type ToolFunc func(ctx context.Context, queriers map[string]interface{}, userScopes []string, args map[string]interface{}) (interface{}, error)

// Tool bundles the function with the required permission
type Tool struct {
	Function           ToolFunc
	RequiredPermission string
}

// RAGContext holds the specific configuration for a single RAG application personality.
type RAGContext struct {
	Name                string
	PlannerTemplate     *template.Template
	SynthesizerTemplate *template.Template
	Tools               map[string]Tool
	MaxReActCycles      int
}

// RAGRegistry holds all the registered RAG contexts for the platform.
type RAGRegistry struct {
	contexts map[string]RAGContext
}

// NewRAGRegistry creates and returns a new registry.
func NewRAGRegistry() *RAGRegistry {
	return &RAGRegistry{
		contexts: make(map[string]RAGContext),
	}
}

// Register adds a new RAG context to the registry.
func (r *RAGRegistry) Register(context RAGContext) {
	if _, exists := r.contexts[context.Name]; exists {
		panic(fmt.Sprintf("RAG context '%s' is already registered", context.Name))
	}
	r.contexts[context.Name] = context
}

// Get retrieves a RAG context from the registry by name.
func (r *RAGRegistry) Get(name string) (RAGContext, bool) {
	context, found := r.contexts[name]
	return context, found
}
