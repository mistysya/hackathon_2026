package campaign

import (
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

func TestLifecycleServiceTrimsInputAndMapsRepositoryErrors(t *testing.T) {
	repository := &fakeLifecycleRepository{approved: lifecycleExpectedCampaign()}
	service := NewLifecycleService(repository)
	got, err := service.Approve(context.Background(), " c_1 ", " hr@example.test ")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if got.CampaignID != "c_1" || repository.approveID != "c_1" || repository.approvedBy != "hr@example.test" {
		t.Fatalf("approve result/call = %+v / %+v", got, repository)
	}
	if _, err := service.Reject(context.Background(), " ", "reason"); !errors.Is(err, ErrLifecycleInvalidInput) {
		t.Fatalf("blank ID error = %v, want ErrLifecycleInvalidInput", err)
	}
	if _, err := service.Approve(context.Background(), "c_1", " "); !errors.Is(err, ErrLifecycleInvalidInput) {
		t.Fatalf("blank approver error = %v, want ErrLifecycleInvalidInput", err)
	}
	if _, err := service.Reject(context.Background(), "c_1", " "); !errors.Is(err, ErrLifecycleInvalidInput) {
		t.Fatalf("blank reason error = %v, want ErrLifecycleInvalidInput", err)
	}
	for name, repositoryError := range map[string]error{"not found": store.ErrNotFound, "conflict": store.ErrConflict} {
		t.Run(name, func(t *testing.T) {
			service := NewLifecycleService(&fakeLifecycleRepository{approveErr: repositoryError})
			_, err := service.Approve(context.Background(), "c_1", "hr@example.test")
			if name == "not found" && !errors.Is(err, ErrCampaignNotFound) {
				t.Fatalf("error = %v, want campaign not found", err)
			}
			if name == "conflict" && !errors.Is(err, ErrCampaignConflict) {
				t.Fatalf("error = %v, want campaign conflict", err)
			}
		})
	}
}

func TestLifecycleRoutesFrozenHTTPContract(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	makeHandler := func(approveErr, rejectErr error) http.Handler {
		repository := &fakeLifecycleRepository{approved: lifecycleExpectedCampaign(), rejected: lifecycleExpectedCampaign(), approveErr: approveErr, rejectErr: rejectErr}
		return httpapi.NewRouter(logger, NewLifecycleRoutes(NewLifecycleService(repository), logger))
	}
	cases := []struct {
		name, method, path, body, contentType string
		handler                               http.Handler
		status                                int
		code                                  string
	}{
		{name: "approve", method: http.MethodPost, path: "/campaigns/c_1/approve", body: `{"approvedBy":"hr@example.test"}`, contentType: "application/json", handler: makeHandler(nil, nil), status: http.StatusOK},
		{name: "reject", method: http.MethodPost, path: "/campaigns/c_1/reject", body: `{"reason":"Needs review"}`, contentType: "application/json", handler: makeHandler(nil, nil), status: http.StatusOK},
		{name: "content type", method: http.MethodPost, path: "/campaigns/c_1/approve", body: `{"approvedBy":"hr@example.test"}`, contentType: "text/plain", handler: makeHandler(nil, nil), status: http.StatusBadRequest, code: "invalid_content_type"},
		{name: "unknown field", method: http.MethodPost, path: "/campaigns/c_1/approve", body: `{"approvedBy":"hr@example.test","extra":true}`, contentType: "application/json", handler: makeHandler(nil, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "trailing JSON", method: http.MethodPost, path: "/campaigns/c_1/reject", body: `{"reason":"Needs review"} {}`, contentType: "application/json", handler: makeHandler(nil, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "empty field", method: http.MethodPost, path: "/campaigns/c_1/reject", body: `{"reason":" "}`, contentType: "application/json", handler: makeHandler(nil, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "missing", method: http.MethodPost, path: "/campaigns/c_1/approve", body: `{"approvedBy":"hr@example.test"}`, contentType: "application/json", handler: makeHandler(store.ErrNotFound, nil), status: http.StatusNotFound, code: "campaign_not_found"},
		{name: "conflict", method: http.MethodPost, path: "/campaigns/c_1/reject", body: `{"reason":"Needs review"}`, contentType: "application/json", handler: makeHandler(nil, store.ErrConflict), status: http.StatusConflict, code: "campaign_conflict"},
		{name: "repository failure", method: http.MethodPost, path: "/campaigns/c_1/approve", body: `{"approvedBy":"hr@example.test"}`, contentType: "application/json", handler: makeHandler(errors.New("database down"), nil), status: http.StatusInternalServerError, code: "internal_error"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			test.handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q", response.Header().Get("Content-Type"))
			}
			if test.code != "" {
				var envelope struct {
					Error struct{ Code, RequestID string }
				}
				if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
					t.Fatal(err)
				}
				if envelope.Error.Code != test.code || envelope.Error.RequestID == "" {
					t.Fatalf("error envelope = %+v", envelope.Error)
				}
			}
		})
	}
}

func lifecycleExpectedCampaign() domain.GeneratedCampaign {
	return domain.GeneratedCampaign{CampaignID: "c_1", Status: domain.CampaignStatusApproved, SafetyChecks: []domain.SafetyCheck{}}
}

type fakeLifecycleRepository struct {
	approved, rejected        domain.GeneratedCampaign
	approveErr, rejectErr     error
	approveID, approvedBy     string
	rejectID, rejectionReason string
}

func (repository *fakeLifecycleRepository) ApproveCampaign(_ context.Context, campaignID, approvedBy string) (domain.GeneratedCampaign, error) {
	repository.approveID, repository.approvedBy = campaignID, approvedBy
	return repository.approved, repository.approveErr
}

func (repository *fakeLifecycleRepository) RejectCampaign(_ context.Context, campaignID, reason string) (domain.GeneratedCampaign, error) {
	repository.rejectID, repository.rejectionReason = campaignID, reason
	return repository.rejected, repository.rejectErr
}
