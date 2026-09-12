package employee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func TestImportRouteContentTypeAndResponse(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{imported: []bool{true}}
	handler := employeeHandler(repository)

	badRequest := httptest.NewRequest(http.MethodPost, "/employees/import", strings.NewReader(""))
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, badRequest)
	assertErrorCode(t, badResponse, http.StatusBadRequest, "invalid_content_type")

	request := httptest.NewRequest(http.MethodPost, "/employees/import", strings.NewReader(
		"employee_id,display_name,email,department,title,company\nE001,Demo,demo@example.test,,,\n"))
	request.Header.Set("Content-Type", "text/csv; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("POST import status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body ImportResponse
	decodeBody(t, response.Body, &body)
	if body.Imported != 1 || body.Skipped != 0 || body.Errors == nil || len(body.Errors) != 0 {
		t.Fatalf("POST import response = %#v", body)
	}
}

func TestImportRouteMapsInvalidCSVAndMaximumBody(t *testing.T) {
	t.Parallel()
	handler := employeeHandler(&fakeRepository{})
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "csv syntax", body: "employee_id,display_name,email,department,title,company\nE001,\"Demo,demo@example.test,,,\n"},
		{name: "maximum body", body: strings.Repeat("x", int(httpapi.DefaultMaxBodyBytes)+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/employees/import", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "text/csv")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			assertErrorCode(t, response, http.StatusBadRequest, "invalid_csv")
		})
	}
}

func TestEmployeeQueryRoutes(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{
		list:    []domain.EmployeeSummary{{EmployeeID: "E001", DisplayName: "Demo", HasProfile: false}},
		details: domain.EmployeeDetails{EmployeeID: "E001", DisplayName: "Demo", Email: "demo@example.test"},
	}
	handler := employeeHandler(repository)

	listRequest := httptest.NewRequest(http.MethodGet, "/employees", nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || listResponse.Body.String() == "null\n" {
		t.Fatalf("GET employees = %d %s", listResponse.Code, listResponse.Body.String())
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/employees/E001", nil)
	detailResponse := httptest.NewRecorder()
	handler.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("GET employee = %d %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detail map[string]any
	decodeBody(t, detailResponse.Body, &detail)
	if detail["employeeId"] != "E001" || detail["profile"] != nil {
		t.Fatalf("GET employee body = %#v", detail)
	}

	repository.getErr = store.ErrNotFound
	notFoundRequest := httptest.NewRequest(http.MethodGet, "/employees/unknown", nil)
	notFoundResponse := httptest.NewRecorder()
	handler.ServeHTTP(notFoundResponse, notFoundRequest)
	assertErrorCode(t, notFoundResponse, http.StatusNotFound, "employee_not_found")

	repository.getErr = errors.New("SQL details must not be returned")
	failureRequest := httptest.NewRequest(http.MethodGet, "/employees/E001", nil)
	failureResponse := httptest.NewRecorder()
	handler.ServeHTTP(failureResponse, failureRequest)
	assertErrorCode(t, failureResponse, http.StatusInternalServerError, "internal_error")
	if strings.Contains(failureResponse.Body.String(), "SQL details") {
		t.Fatalf("error leaked repository details: %s", failureResponse.Body.String())
	}
}

func employeeHandler(repository *fakeRepository) http.Handler {
	return httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), NewRoutes(NewService(repository), nil))
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeBody(t, response.Body, &body)
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q", body.Error.Code, code)
	}
}

func decodeBody(t *testing.T, body *bytes.Buffer, value any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(value); err != nil {
		t.Fatalf("decode response: %v; body = %q", err, body.String())
	}
}

// Keep context imported as part of compile-time confirmation that fakeRepository
// continues to satisfy the full repository port used by the service.
var _ interface {
	ImportEmployees(context.Context, []domain.Employee) ([]bool, error)
} = (*fakeRepository)(nil)
