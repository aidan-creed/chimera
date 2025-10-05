package api

import (
	"context"
	"fmt"
	"github.com/jjckrbbt/chimera/backend/internal/repository"
)

// ListParams holds the common pagination parameters
type ListParams struct {
	Limit  int32
	Offset int32
}

// ItemListFetcher the signature for any function that can fetch a list of items.
type ItemListFetcher func(ctx context.Context, db repository.DBTX, params ListParams) (interface{}, int64, error)

// ItemRegistry is the signature for any function that can fetch a list of items.
var ItemRegistry = make(map[string]ItemListFetcher)

// FetcherRegistry holds a map of the item types to their corresponding fetcher functions
type FetcherRegistry struct {
	fetchers map[string]ItemListFetcher
}

// NewFetcherRegistry creates and return a new registry
func NewFetcherRegistry() *FetcherRegistry {
	return &FetcherRegistry{
		fetchers: make(map[string]ItemListFetcher),
	}
}

// Register add a fetcher function to the registry for a given item type
func (r *FetcherRegistry) Register(itemType string, fetcher ItemListFetcher) {
	if _, exists := r.fetchers[itemType]; exists {
		panic(fmt.Sprintf("Fetcher for item type '%s' is already registered", itemType))
	}
	r.fetchers[itemType] = fetcher
}

// Get retrieves a fetcher function from the registry for  given item type
func (r *FetcherRegistry) Get(itemType string) (ItemListFetcher, bool) {
	fetcher, found := r.fetchers[itemType]
	return fetcher, found
}
