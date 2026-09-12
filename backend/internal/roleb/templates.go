package roleb

import (
	"bytes"
	"html/template"
	"strings"
	texttemplate "text/template"
)

type ScenarioTemplate struct {
	ID          string
	Difficulty  string
	Subject     string
	Greeting    string
	Body        string
	CTALabel    string
	Landing     LandingConfig
	SafetyNotes []string
}

type RenderedEmail struct {
	Subject string `json:"subject"`
	HTML    string `json:"emailHtml"`
	Text    string `json:"emailText"`
}

var ScenarioTemplates = []ScenarioTemplate{
	{
		ID:         "event_followup",
		Difficulty: "medium",
		Subject:    "【提醒】{{.Topic}}後續問卷與資料下載",
		Greeting:   "Hi {{.DisplayName}},",
		Body:       "感謝參加本次 {{.Topic}}。請於下方連結填寫問卷並下載簡報。",
		CTALabel:   "前往問卷與資料下載",
		Landing: LandingConfig{
			Title:       "活動後續問卷",
			Brand:       "Demo Corp Training (測試品牌)",
			Description: "請填寫測試欄位以完成演練。此頁不會傳送任何欄位值。",
			CTALabel:    "送出測試表單",
		},
	},
	{
		ID:         "training_reminder",
		Difficulty: "low",
		Subject:    "【教育訓練提醒】{{.Department}} 資安課程確認",
		Greeting:   "Hi {{.DisplayName}},",
		Body:       "本週有一堂 {{.Topic}} 內部課程，請確認測試頁面上的出席資訊。",
		CTALabel:   "確認課程資訊",
		Landing: LandingConfig{
			Title:       "教育訓練確認",
			Brand:       "Demo Corp Learning Lab (測試品牌)",
			Description: "這是安全演練用頁面，僅記錄互動事件。",
			CTALabel:    "送出測試確認",
		},
	},
	{
		ID:         "benefit_update",
		Difficulty: "medium",
		Subject:    "【一般通知】員工福利方案資訊更新",
		Greeting:   "Hi {{.DisplayName}},",
		Body:       "HR 測試品牌正在整理一般福利資訊，請前往演練頁查看摘要。",
		CTALabel:   "查看福利資訊摘要",
		Landing: LandingConfig{
			Title:       "福利資訊摘要（測試）",
			Brand:       "Demo Corp People Ops Sandbox",
			Description: "頁面使用虛構測試品牌，不要求真實個資、密碼或金融資料。",
			CTALabel:    "完成測試閱讀",
		},
	},
	{
		ID:         "saas_security_notice",
		Difficulty: "high",
		Subject:    "【測試通知】內部 SaaS 安全設定檢查",
		Greeting:   "Hi {{.DisplayName}},",
		Body:       "這是一封受控演練通知，請前往測試頁確認安全提醒。請勿輸入真實密碼。",
		CTALabel:   "查看安全提醒",
		Landing: LandingConfig{
			Title:       "SaaS 安全提醒（測試）",
			Brand:       "Demo Corp Security Sandbox",
			Description: "本頁是演練沙盒，不模仿真實登入頁，也不收集憑證。",
			CTALabel:    "我已閱讀提醒",
		},
	},
}

func RenderEmail(t ScenarioTemplate, data map[string]string) (RenderedEmail, error) {
	if data == nil {
		data = map[string]string{}
	}
	dataWithDefaults := map[string]string{
		"DisplayName": valueOr(data["DisplayName"], "Demo User"),
		"Department":  valueOr(data["Department"], "Demo Team"),
		"Topic":       valueOr(data["Topic"], "AI 資安研討會"),
	}

	subject, err := renderText(t.Subject, dataWithDefaults)
	if err != nil {
		return RenderedEmail{}, err
	}
	greeting, err := renderText(t.Greeting, dataWithDefaults)
	if err != nil {
		return RenderedEmail{}, err
	}
	body, err := renderText(t.Body, dataWithDefaults)
	if err != nil {
		return RenderedEmail{}, err
	}

	text := strings.Join([]string{greeting, "", body, "", t.CTALabel + ": {{landingUrl}}", "", "Demo Corp 演練沙盒（測試品牌）"}, "\n")
	return RenderedEmail{
		Subject: subject,
		HTML: `<html><body>` +
			`<p>` + template.HTMLEscapeString(greeting) + `</p>` +
			`<p>` + template.HTMLEscapeString(body) + `</p>` +
			`<p><a href="{{landingUrl}}">` + template.HTMLEscapeString(t.CTALabel) + `</a></p>` +
			`<p>Demo Corp 演練沙盒（測試品牌）</p>` +
			`</body></html>`,
		Text: text,
	}, nil
}

func renderText(tmpl string, data map[string]string) (string, error) {
	parsed, err := texttemplate.New("text").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
