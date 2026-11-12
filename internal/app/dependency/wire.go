//go:build wireinject
// +build wireinject

package dependency

import (
	"net/http"

	"github.com/google/wire"

	"bookcabin/config"
	"bookcabin/internal/app/http/handler"
	"bookcabin/internal/app/http/route"
	"bookcabin/internal/repository"
	"bookcabin/internal/usecase"
)

// InitHandler assembles the full dependency graph and returns the HTTP router.
// The concrete implementations (airline providers, search repository, cache,
// use case service and handler) are bound to the use case ports via Wire.
func InitHandler(cfg config.Config) (http.Handler, error) {
	wire.Build(
		repository.NewProviders,
		repository.NewSearchRepo,
		wire.Bind(new(usecase.SearchRepository), new(*repository.SearchRepo)),
		provideResultCache,
		provideService,
		wire.Bind(new(usecase.SearchItf), new(*usecase.SearchUseCase)),
		handler.New,
		route.New,
	)
	return nil, nil
}
