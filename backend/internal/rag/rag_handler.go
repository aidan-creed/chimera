// backend/internal/rag/rag_handler.go
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// RAGHandler is the generic API handler for all RAG-based chat interactions.
type RAGHandler struct {
	registry *RAGRegistry
	service  *RAGService
	logger   *slog.Logger
	queriers map[string]interface{}
}

// NewRAGHandler creates a new instance of the RAGHandler.
func NewRAGHandler(reg *RAGRegistry, svc *RAGService, logger *slog.Logger, queriers map[string]interface{}) *RAGHandler {
	return &RAGHandler{
		registry: reg,
		service:  svc,
		logger:   logger.With("component", "rag_handler"),
		queriers: queriers,
	}
}

// --- Structs for the RAG Pipeline ---

type RAGRequest struct {
	Context  string        `json:"context"`
	Question string        `json:"question"`
	History  []ChatMessage `json:"history"`
}

type ChatMessage struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

type PlannerResponse struct {
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	ToolName  string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}

// --- Main Handler ---

// HandleRAGQuery is the main entry point for the generic RAG API.
func (h *RAGHandler) HandleRAGQuery(c echo.Context) error {
	ctx := c.Request().Context()
	var req RAGRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// 1. Look up the context from the registry
	ragContext, found := h.registry.Get(req.Context)
	if !found {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid RAG context specified: "+req.Context)
	}

	reqLogger := h.logger.With("request_id", c.Get("requestID"), "context", req.Context)
	reqLogger.InfoContext(ctx, "Executing RAG query", "question", req.Question)

	// --- The ReAct Loop ---
	scratchpad := make(map[string]interface{})
	var finalAnswer json.RawMessage

	// use the configured limit, with safe default of 1
	maxCycles := ragContext.MaxReActCycles
	if maxCycles <= 0 {
		maxCycles = 1
	}

	for i := 0; i < maxCycles; i++ {
		reqLogger.InfoContext(ctx, "Starting ReAct Cycle", "cycle", i+1, "max_cycles", maxCycles)

		// STEP 1: PLAN - Decide which tools to use
		plan, err := h.getExecutionPlan(ctx, ragContext, req, scratchpad)
		if err != nil {
			reqLogger.ErrorContext(ctx, "Failed to get execution plan", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Error during planning phase")
		}

		if len(plan) == 1 && plan[0].ToolName == "final_answer" {
			if answer, ok := plan[0].Arguments["answer"].(string); ok {
				finalAnswer = json.RawMessage(answer)
			}
			break
		}

		// STEP 2: EXECUTE - Run the tools to fetch data
		retrievedData, err := h.executePlan(ctx, ragContext, plan)
		if err != nil {
			reqLogger.ErrorContext(ctx, "Failed to execute plan", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Error during execution phase")
		}

		for key, value := range retrievedData {
			scratchpad[key] = value
		}
	}
	// STEP 3: SYNTHESIZE - Generate a final response from the data
	var err error
	if finalAnswer == nil {
		reqLogger.InfoContext(ctx, "Max cycles reached. Synthesizing final answer from scratchpad.")
		finalAnswer, err = h.synthesizeAnswer(ctx, ragContext, req, scratchpad)
		if err != nil {
			reqLogger.ErrorContext(ctx, "Failed to synthesize answer", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Error during synthesis phase")
		}
	}
	return c.JSON(http.StatusOK, finalAnswer)
}

// --- Pipeline Helper Functions ---

func (h *RAGHandler) getExecutionPlan(ctx context.Context, ragCtx RAGContext, req RAGRequest, scratchpad map[string]interface{}) ([]ToolCall, error) {
	var promptBuffer bytes.Buffer

	templateData := map[string]interface{}{
		"UserQuestion": req.Question,
		"History":      req.History,
		"Scratchpad":   scratchpad,
	}
	if err := ragCtx.PlannerTemplate.Execute(&promptBuffer, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute planner template: %w", err)
	}

	llmResponseContent, err := h.service.CallLLM(ctx, promptBuffer.String(), true)
	if err != nil {
		return nil, fmt.Errorf("LLM call for planning failed: %w", err)
	}

	cleanedJSON := strings.Trim(strings.TrimSpace(llmResponseContent), "```json \n")
	var plannerResponse PlannerResponse
	if err := json.Unmarshal([]byte(cleanedJSON), &plannerResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool call plan from LLM: %w. Raw content: %s", err, llmResponseContent)
	}

	return plannerResponse.ToolCalls, nil
}

func (h *RAGHandler) executePlan(ctx context.Context, context RAGContext, plan []ToolCall) (map[string]interface{}, error) {
	retrievedData := make(map[string]interface{})

	// Get the user's permissions and scopes that were injected by the middleware.
	userPermissions, _ := ctx.Value("user_permissions").([]string)
	userScopes, _ := ctx.Value("user_scopes").([]string)

	// Create a map for quick permission lookups.
	permissionSet := make(map[string]struct{})
	for _, p := range userPermissions {
		permissionSet[p] = struct{}{}
	}

	for _, toolCall := range plan {
		tool, found := context.Tools[toolCall.ToolName]
		if !found {
			h.logger.WarnContext(ctx, "Planner requested an unknown tool", "tool_name", toolCall.ToolName)
			continue
		}

		// === PERMISSION CHECK (Action-Based) ===
		_, hasPermission := permissionSet[tool.RequiredPermission]
		if !hasPermission {
			h.logger.WarnContext(ctx, "User attempted to use tool without required permission", "tool_name", toolCall.ToolName, "required_permission", tool.RequiredPermission)
			retrievedData[toolCall.ToolName] = map[string]string{"error": "Access denied. You do not have permission to use this tool."}
			continue // Skip this tool
		}

		// === EXECUTE TOOL WITH SCOPES (Data-Based) ===
		// The user's authorized scopes are passed directly to the tool function.
		result, err := tool.Function(ctx, h.queriers, userScopes, toolCall.Arguments)
		if err != nil {
			h.logger.ErrorContext(ctx, "Tool execution failed", "tool_name", toolCall.ToolName, "error", err)
			retrievedData[toolCall.ToolName] = map[string]string{"error": err.Error()}
			continue
		}
		retrievedData[toolCall.ToolName] = result
	}

	return retrievedData, nil
}

func (h *RAGHandler) synthesizeAnswer(ctx context.Context, ragCtx RAGContext, req RAGRequest, data map[string]interface{}) (json.RawMessage, error) {
	var promptBuffer bytes.Buffer

	// Marshal the retrieved data so it can be injected into the prompt
	contextDataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data for synthesizer: %w", err)
	}

	templateData := map[string]interface{}{
		"UserQuestion": req.Question,
		"History":      req.History,
		"ContextData":  string(contextDataJSON),
	}

	if err := ragCtx.SynthesizerTemplate.Execute(&promptBuffer, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute synthesizer template: %w", err)
	}

	finalResponse, err := h.service.CallLLM(ctx, promptBuffer.String(), true)
	if err != nil {
		return nil, fmt.Errorf("LLM call for synthesis failed: %w", err)
	}

	// We return the raw JSON from the LLM, as it's expected to be the final, structured
	// response for the frontend (e.g., with text_response, render_table actions).
	return json.RawMessage(finalResponse), nil
}
