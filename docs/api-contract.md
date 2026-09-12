# API 契約（Hackathon MVP 定案版）

> 狀態：**Frozen**。任何改動需三人同意。所有回應 `Content-Type: application/json`（CSV 匯入與 landing 頁除外）。
>
> 錯誤格式統一：`{ "error": { "code": "...", "message": "...", "requestId": "..." } }`

## 0. 共通約定

- 成功建立資源回 `201`；一般成功回 `200`；無回傳內容回 `204`。
- 輸入格式錯誤 `400`；資源不存在 `404`；狀態不允許（如未核准即寄送）`409`。
- MVP 無登入機制；`approvedBy` 由前端帶入。
- MVP 不驗證 event 來源（landing/前端可直接打 `POST /events`）。
- ID 型別：`employees` 內部自增 int，對外一律用業務碼 `employeeId`(string)；`campaignId` 與 `token` 為不可猜隨機字串。
- **路由參數以 chi 語法為準**：文件中的 `:id` / `:token` 對應 Go 路由 `{id}` / `{token}`，handler 用 `chi.URLParam(r, "id")` 取值。
- **所有時間欄位一律 RFC3339 UTC 字串**（例：`2026-09-12T14:03:11Z`）。DB 以 `strftime('%Y-%m-%dT%H:%M:%SZ','now')` 產生，Go repository 直接以字串進出，不做時區轉換。
- **前後端整合（定案：Vite proxy）**：後端固定 `http://localhost:8080`；前端 dev 以 Vite proxy 將 `/api` 轉發至後端，前端一律呼叫相對路徑 `/api/...`（同源，無 CORS 問題）。後端不需開 CORS。（僅在改為前後端不同 origin 部署時才啟用 CORS，非預設。）

## 1. 統一型別（TS 與 Go 必須一致）

```ts
type PublicFact = {
  fact: string;
  sourceUrl: string | null;   // 有事實但來源未知時為 null；「無事實可回報」則整個 publicFacts 為空陣列
  confidence: number | null;  // 0..1
  sourceType: "live" | "fixture" | "manual";  // 供 UI 標示 source-backed / mock，與 DB source_type 對齊
};

type EmployeeProfile = {
  employeeId: string;
  displayName: string;
  department: string;
  publicFacts: PublicFact[];
  riskSignals: string[];
  recommendedScenario: string;
};

type SafetyCheck = { rule: string; passed: boolean; detail?: string };

type CampaignStatus = "pending_review" | "approved" | "rejected" | "simulated";

type GeneratedCampaign = {
  campaignId: string;
  employeeId: string;
  templateId: string;
  difficulty: "low" | "medium" | "high";
  subject: string;
  emailHtml: string;        // 模板，可能含 {{landingUrl}} placeholder，見 §2「Landing URL 替換責任」
  landingConfig: {
    title: string;
    brand: string;        // 明確測試品牌，不擬真真實服務
    description: string;
    ctaLabel: string;
  };
  decisionReason: string;
  safetyChecks: SafetyCheck[];
  status: CampaignStatus;
  approvedBy: string | null;
  approvedAt: string | null;      // RFC3339
  rejectionReason: string | null;
};

// 對外 event request（landing/前端使用，絕不含 employeeId）
type EventRequest = { token: string; eventType: EventType };
type EventType = "opened" | "clicked" | "form_attempted" | "training_viewed";

type CampaignReport = {
  campaignId: string;
  targetCount: number;
  funnel: {
    simulated: number; opened: number; clicked: number;
    formAttempted: number; trainingViewed: number;
  };
  events: Array<{ eventType: EventType; occurredAt: string }>;  // 時間軸
};
```

對應 Go struct 需與上述欄位/JSON tag 完全一致；`EventType` 與 `CampaignStatus` 在 Service 層以允許清單驗證，不接受任意字串寫入資料庫。

## 2. Endpoints

### 員工

**`POST /employees/import`** — 匯入 CSV
- Request：`Content-Type: text/csv`，raw body，UTF-8。欄位：`employee_id,display_name,email,department,title,company`
- 去重鍵 `employee_id`，重複則 skip（不中斷）。
- `200`：`{ "imported": 5, "skipped": 2, "errors": [{ "row": 3, "reason": "duplicate employee_id" }] }`

**`GET /employees`**
- `200`：`EmployeeSummary[]` = `{ employeeId, displayName, department, title, hasProfile }`

**`GET /employees/:id`**
- `200`：`{ employeeId, displayName, email, department, title, company, profile: EmployeeProfile | null }`
- `404`：查無此員工

**`POST /employees/:id/enrich`** — 產生/更新 Profile（upsert）
- Request：`{}`（MVP 一律使用 fixture）
- `200`：`EmployeeProfile`
- `404`：查無此員工

### Campaign

**`POST /campaigns/generate`** — Agent 選情境 + 生成內容
- 前置：該員工必須已 enrich，否則 `409`（code `profile_required`）。
- Request：`{ "employeeId": "E001" }`（template 由 Scenario Agent 決定，不由前端傳）
- `201`：`GeneratedCampaign`（status = `pending_review`）
- 生成或 Schema 驗證失敗：`502`（code `generation_failed`），不建立可核准的 Campaign。

**`GET /campaigns/:id`**
- `200`：`GeneratedCampaign`
- `404`：查無此 Campaign

**`POST /campaigns/:id/approve`**
- 前置：status 必須 `pending_review`，否則 `409`。
- Request：`{ "approvedBy": "hr@example.test" }`
- `200`：`GeneratedCampaign`（status = `approved`）

**`POST /campaigns/:id/reject`**
- 前置：status 必須 `pending_review`，否則 `409`。
- Request：`{ "reason": "..." }`（存入 `rejection_reason`）
- `200`：`GeneratedCampaign`（status = `rejected`）

**`POST /campaigns/:id/simulate`** — 模擬寄送，建立 targets + token
- 前置：status 必須 `approved`，否則 `409`（驗收：未核准不可寄送）。
- `200`：
```json
{
  "campaignId": "c_9f2c...",
  "status": "simulated",
  "targets": [
    { "employeeId": "E001", "token": "9f2c8a...", "landingUrl": "/landing/9f2c8a..." }
  ]
}
```

### Landing 與事件

**事件觸發點（4 個階段，每個階段對應一種 event，避免指標重複）**

| 階段 | 誰觸發 | event |
|---|---|---|
| 管理者切到員工視角、開啟模擬信件 | 前端（信件檢視畫面） | `opened` |
| 點擊信件 CTA、進入 landing page | landing 頁載入時 | `clicked` |
| 提交 Dummy Form | landing 頁 JS | `form_attempted` |
| 顯示演練揭露／教育頁 | landing 頁 JS | `training_viewed` |

> 注意：`opened` 由**信件檢視畫面**觸發，`clicked` 才由 landing 觸發，兩者不可同時送，否則漏斗失去意義。

**Landing URL 替換責任**：`emailHtml` 內若含 `{{landingUrl}}`，於**信件顯示時**由前端以該 target 的 `landingUrl`（simulate response 提供）替換；後端不 materialize、`GET /campaigns/:id` 回傳的仍是含 placeholder 的模板。

**`GET /landing/:token`** — 由 Go `html/template` 直接回 HTML 演練頁
- `Content-Type: text/html`
- 頁面載入即以內嵌 JS 打 `clicked`；提交 Dummy Form 打 `form_attempted`；顯示教育頁時打 `training_viewed`。
- 表單欄位值**不送後端**；只記錄事件。
- `404`：token 無效

**`POST /events`**
- Request：`EventRequest` = `{ "token": "9f2c...", "eventType": "clicked" }`
- 後端以 token 反查 `campaign_targets` 取得 campaign/employee，以 `INSERT OR IGNORE` 寫入 `tracking_events`（每個 target 每種 event 僅記一次，重整/重送不灌水）。
- `204`：成功（重複送同樣視為成功）
- `404`：token 無效

### 報告

**`GET /reports/:campaignId`**
- `200`：`CampaignReport`（後端只給計數，比率由前端計算）。funnel 各項為 **distinct target 計數**（一 target 一種 event 至多算 1，與 `INSERT OR IGNORE` 一致）。
- `404`：查無此 Campaign

## 3. 狀態機

```
pending_review → approved → simulated
       └──────→ rejected
```
- 任何 `simulate` 前必須驗證 status = `approved`。
