package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/jsonutil"
)

type Routes struct {
	service *Service
	logger  *slog.Logger
}

func NewRoutes(service *Service, logger *slog.Logger) *Routes {
	if logger == nil {
		logger = slog.Default()
	}
	return &Routes{service: service, logger: logger}
}

func (routes *Routes) RegisterRoutes(router chi.Router) {
	router.Post("/employees/{id}/enrich", routes.enrich)
}

func (routes *Routes) enrich(writer http.ResponseWriter, request *http.Request) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_content_type", "content type must be application/json")
		return
	}
	if err := decodeEmptyObject(request.Body); err != nil {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "request body must be an empty JSON object")
		return
	}
	if routes.service == nil {
		httpapi.WriteError(writer, request, http.StatusBadGateway, "enrichment_failed", "profile enrichment failed")
		return
	}

	profile, err := routes.service.Enrich(request.Context(), chi.URLParam(request, "id"))
	if err != nil {
		switch {
		case errors.Is(err, ErrEmployeeNotFound):
			httpapi.WriteError(writer, request, http.StatusNotFound, "employee_not_found", "employee not found")
		case errors.Is(err, ErrEnrichmentFailed):
			httpapi.WriteError(writer, request, http.StatusBadGateway, "enrichment_failed", "profile enrichment failed")
		default:
			routes.logger.Error("enrich employee profile", "error", err, "employee_id", chi.URLParam(request, "id"))
			httpapi.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(profile)
}

func decodeEmptyObject(body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("body is not an object")
	}
	var payload struct{}
	return jsonutil.DecodeStrict(bytes.NewReader(trimmed), &payload)
}
