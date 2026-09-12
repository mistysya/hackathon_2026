package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const DefaultMaxBodyBytes int64 = 2 << 20

type RouteRegistrar interface {
	RegisterRoutes(chi.Router)
}

func NewRouter(logger *slog.Logger, registrars ...RouteRegistrar) http.Handler {
	router := chi.NewRouter()

	for _, registrar := range registrars {
		if registrar != nil {
			registrar.RegisterRoutes(router)
		}
	}

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, "not_found", "route not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	var handler http.Handler = router
	handler = LimitBody(DefaultMaxBodyBytes)(handler)
	handler = Recoverer(logger)(handler)
	handler = AccessLog(logger)(handler)
	handler = RequestID(handler)
	return handler
}
