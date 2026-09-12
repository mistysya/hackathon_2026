# Mailbox demo

1. Start API: `cd backend && docker compose up --build api`
2. Start admin: `cd frontend && npm install && VITE_USE_FIXTURES=false npm run dev`
3. Start mailbox: `python3 -m http.server 4173 --directory presentation`
4. In admin: import employee → enrich → generate → **開啟員工信箱 ↗**.
5. Click **核准 Campaign**, then **模擬寄送 →**.
6. The generated campaign appears as email #14; open it to record `opened`.
