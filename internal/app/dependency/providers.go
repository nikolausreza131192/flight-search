// Package dependency is the composition root helpers: it declares the provider
// functions that adapt generic infrastructure and the use case to the concrete
// types required by the object graph. The graph itself is assembled by Wire in
// wire_gen.go.
package dependency

import (
	"bookcabin/config"
	"bookcabin/internal/usecase"
	"bookcabin/internal/util/cache"
)

// provideResultCache adapts the generic TTL cache to the use case's ResultCache
// port for the concrete SearchResponse value type.
func provideResultCache(cfg config.Config) usecase.ResultCache {
	return cache.New[usecase.SearchResponse](cfg.CacheTTL)
}

// provideService builds the search use case from its injected ports.
func provideService(c usecase.ResultCache, repo usecase.SearchRepository, cfg config.Config) *usecase.SearchUseCase {
	return usecase.New(c, repo, usecase.Options{CacheTTL: cfg.CacheTTL})
}
