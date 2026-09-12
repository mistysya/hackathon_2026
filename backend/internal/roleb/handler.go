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
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; --accent: #2563eb; --accent-soft: #eff6ff; --ink: #172033; }
    * { box-sizing: border-box; }
    body { margin: 0; min-height: 100vh; background: radial-gradient(circle at top left, var(--accent-soft), #f8fafc 42%); color: var(--ink); }
    body.theme-developer { --accent: #7c3aed; --accent-soft: #f3e8ff; }
    body.theme-event { --accent: #0284c7; --accent-soft: #e0f2fe; }
    body.theme-people { --accent: #db2777; --accent-soft: #fce7f3; }
    main { max-width: 920px; margin: 54px auto; padding: 0 22px; }
    .shell { overflow: hidden; background: rgba(255,255,255,.96); border: 1px solid #dbe3ef; border-radius: 22px; box-shadow: 0 24px 70px rgba(23,32,51,.13); }
    .topbar { display: flex; align-items: center; justify-content: space-between; padding: 18px 26px; color: #fff; background: #101827; }
    .portal { display: flex; align-items: center; gap: 12px; font-weight: 760; }
    .portal-mark { display: grid; place-items: center; width: 34px; height: 34px; border-radius: 10px; background: var(--accent); font-size: 15px; }
    .simulation { padding: 6px 10px; border: 1px solid rgba(255,255,255,.35); border-radius: 999px; font-size: 11px; font-weight: 800; letter-spacing: .08em; }
    .content { display: grid; grid-template-columns: 1.08fr .92fr; gap: 0; }
    .intro, .form-panel { padding: 38px; }
    .intro { border-right: 1px solid #e3e8f0; }
    .brand { color: var(--accent); font-weight: 800; letter-spacing: .08em; text-transform: uppercase; font-size: 12px; }
    h1 { margin: 12px 0 16px; max-width: 600px; font-size: clamp(29px, 4vw, 42px); line-height: 1.08; letter-spacing: -.025em; }
    h2 { margin: 0 0 8px; font-size: 21px; }
    .description { color: #475569; font-size: 17px; line-height: 1.65; }
    .audience { margin-top: 28px; padding: 15px 17px; border-radius: 14px; background: var(--accent-soft); }
    .audience span { display: block; margin-bottom: 4px; color: #64748b; font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .08em; }
    .form-panel { background: #fbfcfe; }
    label { display: block; margin: 18px 0 8px; font-size: 14px; font-weight: 700; }
    input { width: 100%; padding: 13px 14px; border: 1px solid #b9c5d7; border-radius: 10px; background: #fff; font-size: 15px; }
    button { width: 100%; margin-top: 22px; padding: 13px 18px; border: 0; border-radius: 10px; background: var(--accent); color: #fff; font-size: 15px; font-weight: 800; cursor: pointer; }
    button:focus, input:focus { outline: 3px solid color-mix(in srgb, var(--accent) 30%, transparent); outline-offset: 2px; }
    .notice { grid-column: 1 / -1; margin: 0 38px 38px; padding: 20px; border-radius: 14px; background: #eefbf2; border: 1px solid #a9dfb9; }
    .safety { margin-top: 22px; color: #64748b; font-size: 12px; line-height: 1.5; }
    .muted { color: #667085; }
    @media (max-width: 720px) { .content { grid-template-columns: 1fr; } .intro { border-right: 0; border-bottom: 1px solid #e3e8f0; } .intro, .form-panel { padding: 28px; } .notice { margin: 0 28px 28px; } }
  </style>
</head>
<body class="theme-{{.Theme}}">
<main>
  <section class="shell">
    <header class="topbar">
      <div class="portal"><span class="portal-mark">S</span>{{.PortalLabel}}</div>
      <span class="simulation">SIMULATION</span>
    </header>
    <div class="content">
      <section class="intro">
        <div class="brand">{{.Brand}}</div>
        <h1>{{.Title}}</h1>
        <p class="description">{{.Description}}</p>
        {{if .Audience}}<div class="audience"><span>Requested for</span><strong>{{.Audience}}</strong></div>{{end}}
        <div class="safety"><strong>Safe demo:</strong> these fields have no name attributes, and their values are never transmitted or stored. Only interaction events are recorded.</div>
      </section>
      <section class="form-panel">
        <h2>{{.FormHeading}}</h2>
        <p class="muted">Use fictional test values only.</p>
        <form id="dummy-form" autocomplete="off" novalidate>
          <label for="demo-user">{{.PrimaryLabel}}</label>
          <input id="demo-user" type="text" placeholder="{{.PrimaryPlaceholder}}">
          <label for="demo-note">{{.SecondaryLabel}}</label>
          <input id="demo-note" type="text" placeholder="{{.SecondaryPlaceholder}}">
          <button type="submit">{{.CTALabel}}</button>
        </form>
      </section>
      <section id="reveal" class="notice" hidden>
        <h2>{{.EducationTitle}}</h2>
        <p>You interacted with a simulated social-engineering page. In a real workflow, verify the sender, destination, and request through a trusted channel before continuing.</p>
        <p class="muted">This exercise records only clicked, form_attempted, and training_viewed events. It never stores field values.</p>
      </section>
    </div>
  </section>
</main>
<script>
(function () {
  const token = {{.Token}};
  const endpoint = {{.EventEndpoint}};
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
