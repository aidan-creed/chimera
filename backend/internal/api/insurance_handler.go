// backend/internal/api/insurance_handler.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jjckrbbt/chimera/backend/internal/apps/insurance"
	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/labstack/echo/v4"
	"github.com/pgvector/pgvector-go"
	"github.com/shopspring/decimal"
)

// --- Structs for RAG pipeline ---
type ChatMessage struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}
type InsuranceQueryRequest struct {
	Question string        `json:"question"`
	History  []ChatMessage `json:"history"`
}
type PlannerResponse struct {
	ToolCalls []ToolCall `json:"tool_calls"`
}
type ToolCall struct {
	ToolName  string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}
type InsuranceContext struct {
	ClaimsData      interface{}
	KnowledgeChunks []SearchResult
	Comments        []SearchResult
}
type SynthesizerTemplateData struct {
	UserQuestion    string
	History         []ChatMessage
	ClaimsData      interface{}
	KnowledgeChunks []SearchResult
	Comments        []SearchResult
}
type ActionPlan struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
type SynthesizerResponse struct {
	Actions []ActionPlan `json:"actions"`
}
type Action struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
type QueryApiResponse struct {
	Actions []Action `json:"actions"`
}
type LLMRequestBody struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
type SearchResult struct {
	Source          string                 `json:"source"`
	Text            string                 `json:"text"`
	SimilarityScore float32                `json:"similarityScore"`
	Metadata        map[string]interface{} `json:"metadata"`
}
type InsuranceHandler struct {
	queries             *insurance.Queries
	platformQuerier     repository.Querier
	httpClient          *http.Client
	embeddingServiceURL string
	plannerTemplate     *template.Template
	synthesizerTemplate *template.Template
	openAIAPIKey        string
	LLMURL              string
	logger              *slog.Logger
}
type UpdateClaimRequest struct {
	BusinessStatus string `json:"business_status"`
}
type CreateCommentRequest struct {
	CommentText string `json:"comment_text"`
}
type EmbeddingRequest struct {
	Text string `json:"text"`
}
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewInsuranceHandler(q *insurance.Queries, pq repository.Querier, apiKey string, LLMURL string, logger *slog.Logger) (*InsuranceHandler, error) {
	funcMap := template.FuncMap{
		"marshal": func(v interface{}) (string, error) {
			if v == nil {
				return "[]", nil
			}
			a, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(a), nil
		},
	}
	plannerTmpl, err := template.New("insurance_planner_prompt.tmpl").Funcs(funcMap).ParseFiles("backend/configs/apps/insurance/prompts/insurance_planner_prompt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse insurance planner template: %w", err)
	}
	synthesizerTmpl, err := template.New("synthesizer_prompt.tmpl").Funcs(funcMap).ParseFiles("backend/configs/apps/insurance/prompts/synthesizer_prompt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse insurance synthesizer template: %w", err)
	}
	return &InsuranceHandler{
		queries:             q,
		platformQuerier:     pq,
		httpClient:          &http.Client{Timeout: 30 * time.Second},
		embeddingServiceURL: "http://embedding-service:5001/embed",
		plannerTemplate:     plannerTmpl,
		synthesizerTemplate: synthesizerTmpl,
		openAIAPIKey:        apiKey,
		LLMURL:              LLMURL,
		logger:              logger.With("component", "insurance_handler"),
	}, nil
}
func (h *InsuranceHandler) HandleListClaims(c echo.Context) error {
	ctx := c.Request().Context()
	reqLogger := h.logger.With("request_id", c.Get("requestID"))
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 32)
	if limit <= 0 {
		limit = 50
	}
	page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 32)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var results interface{}
	var err error
	searchQuery := c.QueryParam("semantic_search_query")

	parseAmount := func(amountStr string) pgtype.Numeric {
		if amountStr == "" {
			return pgtype.Numeric{Valid: false}
		}
		d, err := decimal.NewFromString(amountStr)
		if err != nil {
			return pgtype.Numeric{Valid: false}
		}
		num := new(pgtype.Numeric)
		_ = num.Scan(d.String())
		return *num
	}

	if searchQuery != "" {
		embedding, embErr := h.getEmbedding(ctx, searchQuery)
		if embErr != nil {
			reqLogger.ErrorContext(ctx, "Failed to get embedding", "error", embErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process search query.")
		}
		params := insurance.ListClaimsWithVectorParams{
			Limit:            int32(limit),
			Offset:           int32(offset),
			SearchEmbedding:  pgvector.NewVector(embedding),
			ClaimID:          pgtype.Text{String: c.QueryParam("claim_id"), Valid: c.QueryParam("claim_id") != ""},
			AdjusterAssigned: pgtype.Text{String: c.QueryParam("adjuster_assigned"), Valid: c.QueryParam("adjuster_assigned") != ""},
			Status:           pgtype.Text{String: c.QueryParam("status"), Valid: c.QueryParam("status") != ""},
			PolicyNumber:     pgtype.Text{String: c.QueryParam("policy_number"), Valid: c.QueryParam("policy_number") != ""},
			MinAmount:        parseAmount(c.QueryParam("min_amount")),
			MaxAmount:        parseAmount(c.QueryParam("max_amount")),
		}
		results, err = h.queries.ListClaimsWithVector(ctx, params)
	} else {
		params := insurance.ListClaimsWithoutVectorParams{
			Limit:            int32(limit),
			Offset:           int32(offset),
			ClaimID:          pgtype.Text{String: c.QueryParam("claim_id"), Valid: c.QueryParam("claim_id") != ""},
			AdjusterAssigned: pgtype.Text{String: c.QueryParam("adjuster_assigned"), Valid: c.QueryParam("adjuster_assigned") != ""},
			Status:           pgtype.Text{String: c.QueryParam("status"), Valid: c.QueryParam("status") != ""},
			PolicyNumber:     pgtype.Text{String: c.QueryParam("policy_number"), Valid: c.QueryParam("policy_number") != ""},
			SortBy:           c.QueryParam("sort_by"),
			SortDirection:    c.QueryParam("sort_direction"),
			MinAmount:        parseAmount(c.QueryParam("min_amount")),
			MaxAmount:        parseAmount(c.QueryParam("max_amount")),
		}
		results, err = h.queries.ListClaimsWithoutVector(ctx, params)
	}
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to list insurance claims", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve claims")
	}
	var claimsCount int
	switch v := results.(type) {
	case []insurance.ListClaimsWithVectorRow:
		claimsCount = len(v)
	case []insurance.ListClaimsWithoutVectorRow:
		claimsCount = len(v)
	}
	h.logger.InfoContext(ctx, "Successfully retrieved claims list", "count", claimsCount)
	return c.JSON(http.StatusOK, results)
}
func (h *InsuranceHandler) HandleListPolicyholders(c echo.Context) error {
	ctx := c.Request().Context()
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 32)
	if limit <= 0 {
		limit = 50
	}
	page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 32)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	params := insurance.ListPolicyholdersParams{
		Limit:         int32(limit),
		Offset:        int32(offset),
		State:         pgtype.Text{String: c.QueryParam("state"), Valid: c.QueryParam("state") != ""},
		CustomerLevel: pgtype.Text{String: c.QueryParam("customer_level"), Valid: c.QueryParam("customer_level") != ""},
	}
	policyholders, err := h.queries.ListPolicyholders(ctx, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to list policyholders", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve policyholders")
	}
	h.logger.InfoContext(ctx, "Successfully retrieved policyholders list", "count", len(policyholders))
	return c.JSON(http.StatusOK, policyholders)
}
func (h *InsuranceHandler) HandleGetClaimDetails(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid claim ID format")
	}
	claimDetails, err := h.queries.GetClaimDetails(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get claim details", "error", err, "claim_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve claim details")
	}
	h.logger.InfoContext(ctx, "Successfully retrieved claim details", "claim_id", id)
	return c.JSON(http.StatusOK, claimDetails)
}
func (h *InsuranceHandler) HandleGetClaimStatusHistory(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid claim ID format")
	}
	history, err := h.queries.GetClaimStatusHistory(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get claim status history", "error", err, "claim_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve claim history")
	}
	type HistoryResponse struct {
		ID             int64           `json:"ID"`
		EventTimestamp time.Time       `json:"event_timestamp"`
		EventData      json.RawMessage `json:"event_data"`
		UserName       pgtype.Text     `json:"user_name"`
	}
	response := make([]HistoryResponse, len(history))
	for i, event := range history {
		response[i] = HistoryResponse{
			ID:             event.EventID,
			EventTimestamp: event.EventTimestamp.Time,
			EventData:      event.EventData,
			UserName:       event.UserName,
		}
	}
	return c.JSON(http.StatusOK, response)
}
func (h *InsuranceHandler) HandleUpdateClaim(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid claim ID format")
	}
	var req UpdateClaimRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	var userID int64 = 1 // Placeholder for auth
	existingItem, err := h.platformQuerier.GetItemForUpdate(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}
	var customProps map[string]interface{}
	if err := json.Unmarshal(existingItem.CustomProperties, &customProps); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse existing item properties")
	}
	oldStatus := customProps["Status"]
	customProps["Status"] = req.BusinessStatus
	updatedCustomProps, err := json.Marshal(customProps)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to serialize updated properties")
	}
	updateParams := repository.UpdateItemParams{
		ID:               id,
		Scope:            existingItem.Scope,
		Status:           existingItem.Status,
		CustomProperties: updatedCustomProps,
	}
	_, err = h.platformQuerier.UpdateItem(ctx, updateParams)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to update item", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update claim")
	}
	eventData := map[string]interface{}{"old_status": oldStatus, "new_status": req.BusinessStatus}
	eventDataJSON, _ := json.Marshal(eventData)
	eventParams := repository.CreateItemEventParams{
		ItemID:    id,
		EventType: "CLAIM_STATUS_CHANGED",
		EventData: eventDataJSON,
		CreatedBy: userID,
	}
	_, err = h.platformQuerier.CreateItemEvent(ctx, eventParams)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to create status change event", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create audit event for claim update")
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *InsuranceHandler) HandleListComments(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid claim ID format")
	}
	comments, err := h.platformQuerier.ListCommentsForItem(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to list comments", "error", err, "item_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve comments")
	}
	return c.JSON(http.StatusOK, comments)
}
func (h *InsuranceHandler) HandleCreateComment(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid claim ID format")
	}
	var req CreateCommentRequest
	if err := c.Bind(&req); err != nil || req.CommentText == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: comment_text is required")
	}
	var userID int64 = 1 // Placeholder for auth
	params := repository.CreateCommentParams{
		ItemID:  id,
		Comment: req.CommentText,
		UserID:  userID,
	}
	newComment, err := h.platformQuerier.CreateComment(ctx, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to create comment", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save comment")
	}
	embedding, err := h.getEmbedding(ctx, newComment.Comment)
	if err != nil {
		h.logger.WarnContext(ctx, "Failed to generate embedding for comment", "error", err, "comment_id", newComment.ID)
	} else {
		updateEmbeddingParams := repository.SetCommentEmbeddingParams{
			ID:        newComment.ID,
			Embedding: pgvector.NewVector(embedding),
		}
		err = h.platformQuerier.SetCommentEmbedding(ctx, updateEmbeddingParams)
		if err != nil {
			h.logger.ErrorContext(ctx, "Failed to save embedding for comment", "error", err, "comment_id", newComment.ID)
		}
	}
	return c.JSON(http.StatusCreated, newComment)
}
func (h *InsuranceHandler) getEmbedding(ctx context.Context, textToEmbed string) ([]float32, error) {
	reqBody, err := json.Marshal(EmbeddingRequest{Text: textToEmbed})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", h.embeddingServiceURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
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
func (h *InsuranceHandler) callLLM(ctx context.Context, prompt string, useJSONMode bool) (string, error) {
	h.logger.InfoContext(ctx, "Executing LLM call", "prompt", prompt)
	apiKey := h.openAIAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI key is not configured on the handler")
	}
	payload := LLMRequestBody{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	if useJSONMode {
		payload.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", h.LLMURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var llmResponse LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}
	if len(llmResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}
	h.logger.InfoContext(ctx, "Received LLM response content", "content", llmResponse.Choices[0].Message.Content)
	return llmResponse.Choices[0].Message.Content, nil
}

// --- RAG Handler ---
func (h *InsuranceHandler) HandleInsuranceQuery(c echo.Context) error {
	ctx := c.Request().Context()
	var req InsuranceQueryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}
	if req.Question == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "field 'question' is required")
	}
	plan, err := h.getExecutionPlan(ctx, req.Question, req.History)
	if err != nil {
		h.logger.ErrorContext(ctx, "RAG Error: Failed to get execution plan", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error planning query")
	}
	contextData, err := h.getContextFromPlan(ctx, plan)
	if err != nil {
		h.logger.ErrorContext(ctx, "RAG Error: Failed to execute plan", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error executing plan")
	}
	finalApiResponse, err := h.synthesizeAnswer(ctx, c, req.Question, req.History, contextData)
	if err != nil {
		h.logger.ErrorContext(ctx, "RAG Error: Failed to synthesize answer", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error synthesizing answer")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"answer": finalApiResponse})
}
func (h *InsuranceHandler) getExecutionPlan(ctx context.Context, question string, history []ChatMessage) ([]ToolCall, error) {
	type PlannerTemplateData struct {
		UserQuestion string
		History      []ChatMessage
	}
	templateData := PlannerTemplateData{
		UserQuestion: question,
		History:      history,
	}
	var promptBuffer bytes.Buffer
	if err := h.plannerTemplate.Execute(&promptBuffer, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute planner template: %w", err)
	}
	llmResponseContent, err := h.callLLM(ctx, promptBuffer.String(), true)
	if err != nil {
		return nil, err
	}
	cleanedJSON := strings.TrimPrefix(strings.TrimSpace(llmResponseContent), "```json")
	cleanedJSON = strings.TrimSuffix(cleanedJSON, "```")
	var plannerResponse PlannerResponse
	if err := json.Unmarshal([]byte(cleanedJSON), &plannerResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool call plan from LLM: %w. Raw content: %s", err, llmResponseContent)
	}
	return plannerResponse.ToolCalls, nil
}
func (h *InsuranceHandler) getContextFromPlan(ctx context.Context, plan []ToolCall) (*InsuranceContext, error) {
	var insuranceCtx InsuranceContext
	reqLogger := h.logger.With("plan_execution", true)

	for _, toolCall := range plan {
		switch toolCall.ToolName {
		case "get_claims_data":
			var claimsData interface{}
			var err error
			// Helper functions for safe argument parsing
			getStringArg := func(key string) string {
				if val, ok := toolCall.Arguments[key]; ok {
					if strVal, ok := val.(string); ok {
						return strVal
					}
				}
				return ""
			}
			parseAmount := func(amountVal interface{}) pgtype.Numeric {
				if amountVal == nil {
					return pgtype.Numeric{Valid: false}
				}
				var amountStr string
				switch v := amountVal.(type) {
				case string:
					amountStr = v
				case float64:
					amountStr = fmt.Sprintf("%.2f", v)
				default:
					return pgtype.Numeric{Valid: false}
				}
				if amountStr == "" {
					return pgtype.Numeric{Valid: false}
				}
				d, err := decimal.NewFromString(amountStr)
				if err != nil {
					return pgtype.Numeric{Valid: false}
				}
				num := new(pgtype.Numeric)
				_ = num.Scan(d.String())
				return *num
			}

			searchQuery := getStringArg("semantic_search_query")
			if searchQuery != "" {
				embedding, embErr := h.getEmbedding(ctx, searchQuery)
				if embErr != nil {
					reqLogger.ErrorContext(ctx, "Failed to get embedding", "error", embErr)
					continue
				}
				params := insurance.ListClaimsWithVectorParams{
					Limit:            100,
					Offset:           0,
					SearchEmbedding:  pgvector.NewVector(embedding),
					ClaimID:          pgtype.Text{String: getStringArg("claim_id"), Valid: getStringArg("claim_id") != ""},
					AdjusterAssigned: pgtype.Text{String: getStringArg("adjuster_assigned"), Valid: getStringArg("adjuster_assigned") != ""},
					Status:           pgtype.Text{String: getStringArg("status"), Valid: getStringArg("status") != ""},
					PolicyNumber:     pgtype.Text{String: getStringArg("policy_number"), Valid: getStringArg("policy_number") != ""},
					MinAmount:        parseAmount(toolCall.Arguments["min_amount"]),
					MaxAmount:        parseAmount(toolCall.Arguments["max_amount"]),
				}
				claims, vectorErr := h.queries.ListClaimsWithVector(ctx, params)
				claimsData = claims
				err = vectorErr
			} else {
				params := insurance.ListClaimsWithoutVectorParams{
					Limit:            100,
					Offset:           0,
					ClaimID:          pgtype.Text{String: getStringArg("claim_id"), Valid: getStringArg("claim_id") != ""},
					AdjusterAssigned: pgtype.Text{String: getStringArg("adjuster_assigned"), Valid: getStringArg("adjuster_assigned") != ""},
					Status:           pgtype.Text{String: getStringArg("status"), Valid: getStringArg("status") != ""},
					PolicyNumber:     pgtype.Text{String: getStringArg("policy_number"), Valid: getStringArg("policy_number") != ""},
					SortBy:           getStringArg("sort_by"),
					SortDirection:    getStringArg("sort_direction"),
					MinAmount:        parseAmount(toolCall.Arguments["min_amount"]),
					MaxAmount:        parseAmount(toolCall.Arguments["max_amount"]),
				}
				claims, nonVectorErr := h.queries.ListClaimsWithoutVector(ctx, params)
				claimsData = claims
				err = nonVectorErr
			}
			if err != nil {
				reqLogger.ErrorContext(ctx, "Failed to execute 'get_claims_data' tool", "error", err)
				continue
			}

			var claimsCount int
			switch v := claimsData.(type) {
			case []insurance.ListClaimsWithVectorRow:
				claimsCount = len(v)
			case []insurance.ListClaimsWithoutVectorRow:
				claimsCount = len(v)
			}
			insuranceCtx.ClaimsData = claimsData
			reqLogger.InfoContext(ctx, "Executed tool: get_claims_data", "results_found", claimsCount)

		case "search_knowledge_base":
			getStringArg := func(key string) string {
				if val, ok := toolCall.Arguments[key]; ok {
					if strVal, ok := val.(string); ok {
						return strVal
					}
				}
				return ""
			}
			searchQuery := getStringArg("search_query")
			if searchQuery == "" {
				reqLogger.WarnContext(ctx, "Missing 'search_query' argument for search_knowledge_base")
				continue
			}
			embedding, err := h.getEmbedding(ctx, searchQuery)
			if err != nil {
				reqLogger.ErrorContext(ctx, "Failed to get embedding", "error", err)
				continue
			}
			pgVec := pgvector.NewVector(embedding)

			knowledgeChunks, err1 := h.queries.SearchKnowledgeChunks(ctx, insurance.SearchKnowledgeChunksParams{
				Embedding: pgVec,
				Limit:     5,
			})
			if err1 != nil {
				reqLogger.ErrorContext(ctx, "Failed to search knowledge chunks", "error", err1)
				continue // Use continue to skip to the next tool call on error
			}

			var enrichedResults []SearchResult
			for _, chunk := range knowledgeChunks {
				sourceText, _ := chunk.Source.(string)
				textValue, _ := chunk.Text.(string)
				score, _ := chunk.SimilarityScore.(float64)
				var metadata map[string]interface{}
				if rawJSON, ok := chunk.StructuredMetadata.([]byte); ok && rawJSON != nil {
					_ = json.Unmarshal(rawJSON, &metadata)
				}

				enrichedResult := SearchResult{
					Source:          sourceText,
					Text:            textValue,
					SimilarityScore: float32(score),
					Metadata:        metadata,
				}

				// Fetch and merge header data if a document_id is present
				if enrichedResult.Metadata != nil {
					if docID, ok := enrichedResult.Metadata["document_id"].(string); ok && docID != "" {
						headerMetadataJSON, err := h.queries.GetDocumentHeader(ctx, docID)
						if err != nil {
							reqLogger.WarnContext(ctx, "Could not fetch document header", "doc_id", docID, "error", err)
						} else if headerMetadataJSON != nil {
							if rawJSON, ok := headerMetadataJSON.([]byte); ok {
								var headerMetadata map[string]interface{}
								if err := json.Unmarshal(rawJSON, &headerMetadata); err == nil {
									// Ensure metadata map is initialized
									if enrichedResult.Metadata == nil {
										enrichedResult.Metadata = make(map[string]interface{})
									}
									// Merge header properties into the chunk's metadata
									for key, value := range headerMetadata {
										enrichedResult.Metadata[key] = value
									}
								}
							}
						}
					}
				}
				enrichedResults = append(enrichedResults, enrichedResult)
			}
			insuranceCtx.KnowledgeChunks = append(insuranceCtx.KnowledgeChunks, enrichedResults...)

		case "search_comments":
			getStringArg := func(key string) string {
				if val, ok := toolCall.Arguments[key]; ok {
					if strVal, ok := val.(string); ok {
						return strVal
					}
				}
				return ""
			}
			searchQuery := getStringArg("search_query")
			if searchQuery == "" {
				reqLogger.WarnContext(ctx, "Missing 'search_query' argument for search_comments")
				continue
			}
			embedding, err := h.getEmbedding(ctx, searchQuery)
			if err != nil {
				reqLogger.ErrorContext(ctx, "Failed to get embedding", "error", err)
				continue
			}
			pgVec := pgvector.NewVector(embedding)

			comments, err2 := h.queries.SearchComments(ctx, insurance.SearchCommentsParams{
				Embedding: pgVec,
				Limit:     10,
			})
			if err2 != nil {
				reqLogger.ErrorContext(ctx, "Failed to search comments", "error", err2)
			} else {
				var commentResults []SearchResult
				for _, comment := range comments {
					score, _ := comment.SimilarityScore.(float64)

					commentMetadata := make(map[string]interface{})
					if comment.ClaimID.Valid {
						commentMetadata["claim_id"] = comment.ClaimID.String
					}

					commentResults = append(commentResults, SearchResult{
						Source:          comment.Source,
						Text:            comment.Text,
						SimilarityScore: float32(score),
						Metadata:        commentMetadata,
					})
				}
				insuranceCtx.Comments = commentResults
			}
		}
	}
	return &insuranceCtx, nil
}

func (h *InsuranceHandler) synthesizeAnswer(ctx context.Context, c echo.Context, question string, history []ChatMessage, context *InsuranceContext) (QueryApiResponse, error) {
	h.logger.InfoContext(ctx, "Synthesizing final answer from hybrid context...")
	templateData := SynthesizerTemplateData{
		UserQuestion:    question,
		History:         history,
		ClaimsData:      context.ClaimsData,
		KnowledgeChunks: context.KnowledgeChunks,
		Comments:        context.Comments,
	}
	var promptBuffer bytes.Buffer
	if err := h.synthesizerTemplate.Execute(&promptBuffer, templateData); err != nil {
		return QueryApiResponse{}, fmt.Errorf("failed to execute synthesizer template: %w", err)
	}
	llmResponseContent, err := h.callLLM(ctx, promptBuffer.String(), true)
	if err != nil {
		return QueryApiResponse{}, err
	}
	var synthResponse SynthesizerResponse
	if err := json.Unmarshal([]byte(llmResponseContent), &synthResponse); err != nil {
		return QueryApiResponse{}, fmt.Errorf("failed to unmarshal synthesizer response from LLM: %w. Raw content: %s", err, llmResponseContent)
	}
	var finalApiResponse QueryApiResponse
	if synthResponse.Actions == nil {
		return finalApiResponse, nil
	}
	for _, plannedAction := range synthResponse.Actions {
		finalAction := Action{Type: plannedAction.Type}
		switch plannedAction.Type {
		case "text_response":
			finalAction.Payload = plannedAction.Payload
		case "render_table":
			if wantsTable, ok := plannedAction.Payload.(bool); ok && wantsTable {
				finalAction.Payload = context.ClaimsData
			}
		case "open_detail_drawer":
			if wantsDrawer, ok := plannedAction.Payload.(bool); ok && wantsDrawer {
				var claimID int64
				if claims, ok := context.ClaimsData.([]insurance.ListClaimsWithVectorRow); ok && len(claims) == 1 {
					claimID = claims[0].ID
				} else if claims, ok := context.ClaimsData.([]insurance.ListClaimsWithoutVectorRow); ok && len(claims) == 1 {
					claimID = claims[0].ID
				}
				if claimID > 0 {
					claimDetails, err := h.queries.GetClaimDetails(ctx, claimID)
					if err != nil {
						reqLogger := h.logger.With("request_id", c.Get("requestID"))
						reqLogger.ErrorContext(ctx, "Failed to get claim details for drawer action", "error", err, "claim_id", claimID)
						continue
					}
					finalAction.Payload = claimDetails
				}
			}
		}
		if finalAction.Payload != nil {
			finalApiResponse.Actions = append(finalApiResponse.Actions, finalAction)
		}
	}
	return finalApiResponse, nil
}
