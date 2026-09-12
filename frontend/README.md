# Person C — 管理台與 Dashboard

```bash
cd frontend
npm install
npm run dev
```

預設 `VITE_USE_FIXTURES=true`，可獨立展示 CSV 匯入、Profile、人工核准、模擬互動與 Dashboard。

串接 Go 後端時，建立 `frontend/.env`：

```env
VITE_USE_FIXTURES=false
```

完整 Demo 建議從 repo root 執行 `bash scripts/demo.sh`；操作步驟與驗證指令見 [`../README.md`](../README.md)。

Vite 會將前端的 `/api/*`、simulate 回傳的 `/landing/*`，以及 Landing Page 使用的 `/events` 代理至 `http://localhost:8080`。API 路徑、資料型別及事件語意依 `docs/api-contract.md`：CSV import、enrich、generate、approve/reject、simulate、events、reports。

```bash
npm run build
npx playwright install chromium
npm test
```

`npm test` 自動啟動隔離的 Go API 與 Vite，測試真實整合及 fixture fallback，不會修改 Demo 資料庫。`API_PROXY_TARGET` 可覆寫 proxy 目標（預設 `http://localhost:8080`）。
