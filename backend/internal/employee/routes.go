package employee

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
)

// Routes maps the employee HTTP API to Service use cases.
type Routes struct {
	service *Service
	logger  *slog.Logger
}

// NewRoutes creates employee route handlers.
func NewRoutes(service *Service, logger *slog.Logger) *Routes {
	if logger == nil {
		logger = slog.Default()
	}
	return &Routes{service: service, logger: logger}
}

// RegisterRoutes registers all employee HTTP routes without owning the router's
// composition root.
func (routes *Routes) RegisterRoutes(router chi.Router) {
	router.Post("/employees/import", routes.importEmployees)
	router.Get("/employees", routes.listEmployees)
	router.Get("/employees/{id}", routes.getEmployee)
}

func (routes *Routes) importEmployees(w http.ResponseWriter, r *http.Request) {
	if !isEmployeeCSV(r.Header.Get("Content-Type")) {
		httpapi.WriteError(w, r, http.StatusBadRequest, "invalid_content_type", "content type must be text/csv")
		return
	}

	response, err := routes.service.ImportCSV(r.Context(), r.Body)
	if err != nil {
		if errors.Is(err, ErrInvalidCSV) || isMaxBytesError(err) {
			httpapi.WriteError(w, r, http.StatusBadRequest, "invalid_csv", "invalid csv")
			return
		}
		routes.internalError(w, r, "import employees", err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (routes *Routes) listEmployees(w http.ResponseWriter, r *http.Request) {
	employees, err := routes.service.ListEmployees(r.Context())
	if err != nil {
		routes.internalError(w, r, "list employees", err)
		return
	}
	writeJSON(w, http.StatusOK, employees)
}

func (routes *Routes) getEmployee(w http.ResponseWriter, r *http.Request) {
	employee, err := routes.service.GetEmployee(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			httpapi.WriteError(w, r, http.StatusNotFound, "employee_not_found", "employee not found")
			return
		}
		routes.internalError(w, r, "get employee", err)
		return
	}
	writeJSON(w, http.StatusOK, employee)
}

func (routes *Routes) internalError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	routes.logger.Error(operation+" failed", "error", err)
	httpapi.WriteError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
}

func isEmployeeCSV(contentType string) bool {
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "text/csv" {
		return false
	}
	charset, ok := parameters["charset"]
	return !ok || strings.EqualFold(charset, "utf-8")
}

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
