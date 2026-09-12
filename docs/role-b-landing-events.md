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
- 4 個受控情境信件模板：`event_followup`、`training_reminder`、`benefit_update`、`saas_security_notice`。

## 本機啟動

```bash
cd backend
go test ./...
go run ./cmd/api
```

預設使用 `backend/schema.sql` 初始化 SQLite，DB DSN 可用環境變數覆寫：

```bash
DB_DSN='file:app.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)' go run ./cmd/api
```

## 安全邊界

- Token 為不可猜隨機字串，不含姓名、email 或員工 ID。
- 表單欄位不含 `name` 屬性，JS 只提交 event type。
- `metadata_json` 固定 `{}`，不保存輸入內容。
- Landing page 明確標示測試品牌與資安演練沙盒，不仿冒真實登入頁。
