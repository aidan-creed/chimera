package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

// ItemHandler is a generic handler for the 'items' resource.
type ItemHandler struct {
	queries  repository.Querier
	db       repository.DBTX
	logger   *slog.Logger
	registry *FetcherRegistry
}

// NewItemHandler creates a new instance of the ItemHandler.
func NewItemHandler(q repository.Querier, db repository.DBTX, logger *slog.Logger, registry *FetcherRegistry) *ItemHandler {
	return &ItemHandler{
		queries:  q,
		db:       db,
		logger:   logger.With("component", "item_handler"),
		registry: registry,
	}
}

// --- Request & Response Structs ---

// PaginatedItemsResponse defines the structure for paginated item lists.
type PaginatedItemsResponse struct {
	TotalCount int64       `json:"total_count"`
	Data       interface{} `json:"data"`
}

// CreateItemRequest defines the structure for creating a new generic item.
type CreateItemRequest struct {
	ItemType         string          `json:"item_type"`
	Scope            string          `json:"scope"`
	BusinessKey      string          `json:"business_key"`
	Status           string          `json:"status"`
	CustomProperties json.RawMessage `json:"custom_properties"`
}

// UpdateItemRequest defines the structure for updating an item's mutable fields.
type UpdateItemRequest struct {
	Scope            *string         `json:"scope,omitempty"`
	Status           *string         `json:"status,omitempty"`
	CustomProperties json.RawMessage `json:"custom_properties,omitempty"`
}

// --- Handlers ---

// HandleGetItems retrieves a list of items, filtered by item_type.
func (h *ItemHandler) HandleGetItems(c echo.Context) error {
	ctx := c.Request().Context()
	itemType := c.QueryParam("item_type")
	businessKey := c.QueryParam("business_key")

	if itemType == "" {
		h.logger.WarnContext(ctx, "HandleGetItems called without required 'item_type' query parameter")
		return echo.NewHTTPError(http.StatusBadRequest, "Query parameter 'item_type' is required")
	}

	if businessKey != "" {
		h.logger.WarnContext(ctx, "Attempted lookup by business_key, which is not yet implemented", "business_key", businessKey)
		return c.JSON(http.StatusNotImplemented, "Lookup by business_key not yet implemented")
	}

	fetcher, ok := h.registry.Get(itemType)
	if !ok {
		h.logger.WarnContext(ctx, "HandleGetItems called with unsupported 'item_type'", "item_type", itemType)
		return echo.NewHTTPError(http.StatusBadRequest, "Unsupported 'item_type'"+itemType)
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	params := ListParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	items, totalCount, err := fetcher(ctx, h.db, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to fech items", "item_type", itemType, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve items")
	}

	response := PaginatedItemsResponse{
		TotalCount: totalCount,
		Data:       items,
	}

	return c.JSON(http.StatusOK, response)
}

// HandleCreateItem creates a new item in the database.
func (h *ItemHandler) HandleCreateItem(c echo.Context) error {
	ctx := c.Request().Context()
	var req CreateItemRequest
	if err := c.Bind(&req); err != nil {
		h.logger.WarnContext(ctx, "Failed to bind request body for creating item", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	params := repository.CreateItemParams{
		ItemType:         repository.ItemType(req.ItemType),
		Scope:            pgtype.Text{String: req.Scope, Valid: req.Scope != ""},
		BusinessKey:      pgtype.Text{String: req.BusinessKey, Valid: req.BusinessKey != ""},
		Status:           repository.ItemStatus(req.Status),
		CustomProperties: []byte(req.CustomProperties),
	}

	newItem, err := h.queries.CreateItem(ctx, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to create item in database", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create item")
	}

	h.logger.InfoContext(ctx, "Successfully created new item", "item_id", newItem.ID, "item_type", newItem.ItemType)
	return c.JSON(http.StatusCreated, newItem)
}

// HandleUpdateItem updates an existing item's mutable fields.
func (h *ItemHandler) HandleUpdateItem(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.WarnContext(ctx, "Invalid item ID format provided to update handler", "error", err, "id_param", c.Param("id"))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid item ID format")
	}

	var req UpdateItemRequest
	if err := c.Bind(&req); err != nil {
		h.logger.WarnContext(ctx, "Failed to bind request body for updating item", "error", err, "item_id", id)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	existingItem, err := h.queries.GetItemForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.WarnContext(ctx, "Attempted to update a non-existent item", "item_id", id)
			return echo.NewHTTPError(http.StatusNotFound, "Item not found")
		}
		h.logger.ErrorContext(ctx, "Failed to retrieve item for update", "error", err, "item_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve item for update")
	}

	params := repository.UpdateItemParams{
		ID:               id,
		Scope:            existingItem.Scope,
		Status:           existingItem.Status,
		CustomProperties: existingItem.CustomProperties,
	}

	if req.Scope != nil {
		params.Scope = pgtype.Text{String: *req.Scope, Valid: true}
	}
	if req.Status != nil {
		params.Status = repository.ItemStatus(*req.Status)
	}
	if req.CustomProperties != nil {
		// A real implementation would merge JSONB fields, but for now we overwrite.
		params.CustomProperties = []byte(req.CustomProperties)
	}

	updatedItem, err := h.queries.UpdateItem(ctx, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to update item in database", "error", err, "item_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update item")
	}

	h.logger.InfoContext(ctx, "Successfully updated item", "item_id", updatedItem.ID)
	return c.JSON(http.StatusOK, updatedItem)
}

// HandleGetHistory retrieves the event history for a specific item.
func (h *ItemHandler) HandleGetHistory(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.WarnContext(ctx, "Invalid item ID format for history lookup", "error", err, "id_param", c.Param("id"))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid item ID format")
	}

	history, err := h.queries.GetEventsForItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.WarnContext(ctx, "No history found for item", "item_id", id)
			return c.JSON(http.StatusOK, []interface{}{})
		}
		h.logger.ErrorContext(ctx, "Failed to retrieve item history", "error", err, "item_id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve item history")
	}

	return c.JSON(http.StatusOK, history)
}
