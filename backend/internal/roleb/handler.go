package roleb

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/jsonutil"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/landing/{token}", h.GetLanding)
	r.Post("/events", h.PostEvent)
	r.Post("/campaigns/{id}/simulate", h.PostSimulate)
	r.Get("/reports/{campaignId}", h.GetReport)
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func (h *Handler) GetLanding(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.LandingPage(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeMappedError(w, r, err, "landing_not_found", "landing token not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := landingTemplate.Execute(w, data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

func (h *Handler) PostEvent(w http.ResponseWriter, r *http.Request) {
	var req domain.EventRequest
	if err := jsonutil.DecodeStrict(r.Body, &req); err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "invalid_json", "request body must contain exactly one EventRequest object")
		return
	}
	if err := h.repo.RecordEvent(r.Context(), req); err != nil {
		writeMappedError(w, r, err, "event_not_found", "tracking token not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PostSimulate(w http.ResponseWriter, r *http.Request) {
	resp, err := h.repo.SimulateCampaign(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeMappedError(w, r, err, "campaign_not_found", "campaign not found")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.repo.CampaignReport(r.Context(), chi.URLParam(r, "campaignId"))
	if err != nil {
		writeMappedError(w, r, err, "campaign_not_found", "campaign not found")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func writeMappedError(w http.ResponseWriter, r *http.Request, err error, notFoundCode, notFoundMessage string) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrInvalidToken):
		httpapi.WriteError(w, r, http.StatusNotFound, notFoundCode, notFoundMessage)
	case errors.Is(err, ErrInvalidEvent):
		httpapi.WriteError(w, r, http.StatusBadRequest, "invalid_event", "eventType must be one of opened, clicked, form_attempted, training_viewed")
	case errors.Is(err, ErrConflict):
		httpapi.WriteError(w, r, http.StatusConflict, "invalid_campaign_status", "campaign must be approved before simulate")
	default:
		httpapi.WriteError(w, r, http.StatusInternalServerError, "internal_error", "unexpected server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var landingTemplate = template.Must(template.New("landing").Parse(`<!doctype html>
<html lang="zh-Hant">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root { color-scheme: light; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; background: #f5f7fb; color: #172033; }
    main { max-width: 760px; margin: 48px auto; padding: 0 20px; }
    .card { background: #fff; border: 1px solid #d9e1f2; border-radius: 18px; box-shadow: 0 18px 50px rgba(23,32,51,.08); padding: 32px; }
    .brand { color: #3557ff; font-weight: 700; letter-spacing: .02em; text-transform: uppercase; font-size: 13px; }
    h1 { margin: 10px 0 16px; font-size: 30px; }
    label { display: block; margin: 18px 0 8px; font-weight: 600; }
    input { box-sizing: border-box; width: 100%; padding: 12px 14px; border: 1px solid #b7c4dc; border-radius: 10px; font-size: 16px; }
    button { margin-top: 22px; padding: 12px 18px; border: 0; border-radius: 999px; background: #3557ff; color: #fff; font-weight: 700; cursor: pointer; }
    button:focus, input:focus { outline: 3px solid #aab8ff; outline-offset: 2px; }
    .notice { margin-top: 24px; padding: 18px; border-radius: 14px; background: #eef8f1; border: 1px solid #afe3bd; }
    .safety { margin-top: 18px; padding: 14px; border-radius: 12px; background: #fff8e5; border: 1px solid #f2d27a; }
    .muted { color: #667085; }
  </style>
</head>
<body>
<main>
  <section class="card">
    <div class="brand">{{.Brand}}</div>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    <div class="safety">
      <strong>安全說明：</strong>此頁是 Hackathon 演練沙盒。表單欄位沒有 name 屬性，送出時不會把欄位值傳到後端；系統只記錄互動事件。
    </div>
    <form id="dummy-form" autocomplete="off" novalidate>
      <label for="demo-user">測試使用者</label>
      <input id="demo-user" type="text" placeholder="請勿輸入真實帳號或密碼">
      <label for="demo-note">測試備註</label>
      <input id="demo-note" type="text" placeholder="可留空；內容不會傳送">
      <button type="submit">{{.CTALabel}}</button>
    </form>
    <section id="reveal" class="notice" hidden>
      <h2>{{.EducationTitle}}</h2>
      <p>你剛剛與一個模擬社交工程頁面互動。真實環境中，請在輸入資料前確認寄件者、網址、需求是否合理，並避免在非受信任頁面輸入密碼、MFA 或金融資料。</p>
      <p class="muted">本次演練只記錄 clicked、form_attempted、training_viewed 等事件，不保存你在表單中的輸入值。</p>
    </section>
  </section>
</main>
<script>
(function () {
  const token = {{printf "%q" .Token}};
  const endpoint = {{printf "%q" .EventEndpoint}};
  function sendEvent(eventType) {
    return fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, eventType })
    }).catch(function () { /* Demo should not block on transient tracking failures. */ });
  }
  sendEvent("clicked");
  document.getElementById("dummy-form").addEventListener("submit", function (event) {
    event.preventDefault();
    sendEvent("form_attempted").finally(function () {
      document.getElementById("dummy-form").hidden = true;
      document.getElementById("reveal").hidden = false;
      sendEvent("training_viewed");
    });
  });
}());
</script>
</body>
</html>`))
