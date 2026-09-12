package campaign

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/jsonutil"
)

type LifecycleRoutes struct {
	service *LifecycleService
	logger  *slog.Logger
}

func NewLifecycleRoutes(service *LifecycleService, logger *slog.Logger) *LifecycleRoutes {
	return &LifecycleRoutes{service: service, logger: logger}
}

func (routes *LifecycleRoutes) RegisterRoutes(router chi.Router) {
	router.Post("/campaigns/{id}/approve", routes.approve)
	router.Post("/campaigns/{id}/reject", routes.reject)
}

type approveRequest struct {
	ApprovedBy string `json:"approvedBy"`
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (routes *LifecycleRoutes) approve(writer http.ResponseWriter, request *http.Request) {
	if !isJSON(request.Header.Get("Content-Type")) {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_content_type", "content type must be application/json")
		return
	}
	var input approveRequest
	if err := jsonutil.DecodeStrict(request.Body, &input); err != nil || strings.TrimSpace(input.ApprovedBy) == "" {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "request must contain approvedBy")
		return
	}
	campaign, err := routes.service.Approve(request.Context(), chi.URLParam(request, "id"), input.ApprovedBy)
	if err != nil {
		routes.writeLifecycleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, campaign)
}

func (routes *LifecycleRoutes) reject(writer http.ResponseWriter, request *http.Request) {
	if !isJSON(request.Header.Get("Content-Type")) {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_content_type", "content type must be application/json")
		return
	}
	var input rejectRequest
	if err := jsonutil.DecodeStrict(request.Body, &input); err != nil || strings.TrimSpace(input.Reason) == "" {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "request must contain reason")
		return
	}
	campaign, err := routes.service.Reject(request.Context(), chi.URLParam(request, "id"), input.Reason)
	if err != nil {
		routes.writeLifecycleError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, campaign)
}

func (routes *LifecycleRoutes) writeLifecycleError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, ErrCampaignNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "campaign_not_found", "campaign not found")
	case errors.Is(err, ErrLifecycleInvalidInput):
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "request contains an empty required field")
	case errors.Is(err, ErrCampaignConflict):
		httpapi.WriteError(writer, request, http.StatusConflict, "campaign_conflict", "campaign cannot transition")
	default:
		if routes.logger != nil {
			routes.logger.Error("campaign lifecycle request failed", "error", err)
		}
		httpapi.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

var _ httpapi.RouteRegistrar = (*LifecycleRoutes)(nil)
