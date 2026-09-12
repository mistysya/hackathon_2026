# Role B：信件、落地頁與事件追蹤實作說明

本分支提供成員 B 的可串接後端切面，對齊 `docs/api-contract.md` 的 frozen contract。

## 已提供

- `GET /landing/{token}`：以 Go `html/template` 產生安全演練頁。
  - 頁面載入只送 `clicked`。
  - Dummy Form submit 只送 `form_attempted`，不送欄位值。
  - 顯示揭露／教育內容後送 `training_viewed`。
  - 不在 landing 頁送 `opened`，保留給信件預覽畫面觸發。
- `POST /events`：接受 `{ "token": "...", "eventType": "clicked" }`，以 token 反查 target 並 `INSERT OR IGNORE`，確保同一 target 同一事件只記一次。
- `POST /campaigns/{id}/simulate`：approved campaign 產生 32-hex tracking token，建立 target，回傳 `/landing/{token}`。
- `GET /reports/{campaignId}`：依 distinct target 彙整漏斗並回傳事件時間軸。

信件與情境內容由 Member A campaign generator 單一管理；Role B 使用 persisted `landing_config_json` 渲染受控頁面，避免維護第二套模板。

## 驗證與整合

```bash
cd backend
go test ./...
```

Role B 的 Handler 實作 `httpapi.RouteRegistrar`，不直接擁有 `cmd/api` composition root。最終由 backend integration branch 將 Role B Handler 傳入共用 Router，並共用 Middleware、SQLite lifecycle 與 Go embedded schema。

## 安全邊界

- Token 為不可猜隨機字串，不含姓名、email 或員工 ID。
- 表單欄位不含 `name` 屬性，JS 只提交 event type。
- `metadata_json` 固定 `{}`，不保存輸入內容。
- Landing page 明確標示測試品牌與資安演練沙盒，不仿冒真實登入頁。
