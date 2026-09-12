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

Vite 會將前端的 `/api/*` 代理至 `http://localhost:8080/*`。API 路徑、資料型別及事件語意依 `docs/api-contract.md`：CSV import、enrich、generate、approve/reject、simulate、events、reports。

```bash
npm run build
```
