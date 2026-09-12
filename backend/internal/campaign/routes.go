package campaign

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
)

type Routes struct {
	service *Service
	logger  *slog.Logger
}

func NewRoutes(service *Service, logger *slog.Logger) *Routes {
	return &Routes{service: service, logger: logger}
}

func (routes *Routes) RegisterRoutes(router chi.Router) {
	router.Post("/campaigns/generate", routes.generate)
	router.Get("/campaigns/{id}", routes.get)
}

type generateRequest struct {
	EmployeeID string `json:"employeeId"`
}

func (routes *Routes) generate(writer http.ResponseWriter, request *http.Request) {
	if !isJSON(request.Header.Get("Content-Type")) {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_content_type", "content type must be application/json")
		return
	}
	var input generateRequest
	if err := decodeStrictJSON(request.Body, &input); err != nil || strings.TrimSpace(input.EmployeeID) == "" {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "request must contain employeeId")
		return
	}
	campaign, err := routes.service.Generate(request.Context(), strings.TrimSpace(input.EmployeeID))
	if err != nil {
		routes.writeServiceError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusCreated, campaign)
}

func (routes *Routes) get(writer http.ResponseWriter, request *http.Request) {
	campaign, err := routes.service.Get(request.Context(), chi.URLParam(request, "id"))
	if err != nil {
		routes.writeServiceError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, campaign)
}

func (routes *Routes) writeServiceError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, ErrEmployeeNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "employee_not_found", "employee not found")
	case errors.Is(err, ErrCampaignNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "campaign_not_found", "campaign not found")
	case errors.Is(err, ErrProfileRequired):
		httpapi.WriteError(writer, request, http.StatusConflict, "profile_required", "employee profile is required")
	case errors.Is(err, ErrGenerationFailed):
		httpapi.WriteError(writer, request, http.StatusBadGateway, "generation_failed", "campaign generation failed")
	default:
		if routes.logger != nil {
			routes.logger.Error("campaign request failed", "error", err)
		}
		httpapi.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func decodeStrictJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

var _ httpapi.RouteRegistrar = (*Routes)(nil)
