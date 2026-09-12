-- AI 個人化釣魚演練平台 — SQLite Schema（Hackathon MVP 定案版）
-- 原則：開機時執行，CREATE TABLE IF NOT EXISTS，不使用 migration 框架。
-- FK 一律使用業務碼 employee_id (TEXT)；campaigns/token 使用不可猜的隨機字串。

-- 連線設定注意兩種層級不同：
--   * journal_mode = WAL：資料庫「持久化」設定，設定一次即寫入 DB 檔，之後每次開啟自動沿用。
--   * foreign_keys、busy_timeout：僅對「當前連線」生效，重連即回預設(0)。
-- 因此 Go 端務必在 DSN 讓 database/sql 連線池的每條連線都帶上後兩者，例如
--   modernc.org/sqlite: "file:app.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
-- WAL 允許「多讀單寫」，但同時仍只有一個 writer；高頻寫入請靠 busy_timeout + 短 transaction 重試，不要當作已消除寫入競爭。
-- driver 統一使用 modernc.org/sqlite（純 Go、免 CGO），開工前不得更換。
-- 下面三行僅在直接以 sqlite3 CLI 套用 schema 時有意義；應用程式請改用上方 DSN。
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS employees (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id  TEXT NOT NULL UNIQUE,                 -- 去重鍵
  display_name TEXT NOT NULL,
  email        TEXT NOT NULL,
  department   TEXT,
  title        TEXT,
  company      TEXT,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- enrich 為 upsert：同一 transaction 內先 DELETE 該 employee 的 public_facts，再重新 INSERT，
-- 並 upsert employee_profiles，避免重複 enrich 累積重複事實。
CREATE TABLE IF NOT EXISTS public_facts (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id TEXT NOT NULL REFERENCES employees(employee_id),
  fact        TEXT NOT NULL,
  source_url  TEXT,                                  -- 允許 NULL = 有事實但來源未知；「無事實」則此表無列
  confidence  REAL,                                  -- 0..1
  source_type TEXT NOT NULL DEFAULT 'fixture'        -- live | fixture | manual
);
CREATE INDEX IF NOT EXISTS idx_public_facts_employee ON public_facts(employee_id);

CREATE TABLE IF NOT EXISTS employee_profiles (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  employee_id          TEXT NOT NULL UNIQUE REFERENCES employees(employee_id),  -- 一人一份，enrich 為 upsert
  risk_signals_json    TEXT NOT NULL DEFAULT '[]',
  recommended_scenario TEXT,
  generated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS campaigns (
  id                  TEXT PRIMARY KEY,              -- 隨機字串（uuid/32hex），不可猜
  employee_id         TEXT NOT NULL REFERENCES employees(employee_id),
  template_id         TEXT NOT NULL,
  status              TEXT NOT NULL DEFAULT 'pending_review',  -- pending_review|approved|rejected|simulated
  difficulty          TEXT,                          -- low|medium|high
  decision_reason     TEXT,
  subject             TEXT,                          -- email/landing 內容直接內嵌，不另建 generated_assets
  email_html          TEXT,
  landing_config_json TEXT NOT NULL DEFAULT '{}',
  safety_checks_json  TEXT NOT NULL DEFAULT '[]',
  approved_by         TEXT,
  approved_at         TEXT,
  rejection_reason    TEXT,
  created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_campaigns_employee ON campaigns(employee_id);

-- A generated campaign is stored separately as a privacy-safe mailbox item.
-- The mailbox UI may later read only delivered rows; the generated payload is
-- never reconstructed from a browser message or an external mail provider.
CREATE TABLE IF NOT EXISTS mailbox_messages (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  campaign_id    TEXT NOT NULL UNIQUE REFERENCES campaigns(id),
  employee_id    TEXT NOT NULL REFERENCES employees(employee_id),
  sender_name    TEXT NOT NULL,
  sender_address TEXT NOT NULL,
  subject        TEXT NOT NULL,
  email_html     TEXT NOT NULL,
  created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  delivered_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_mailbox_messages_employee_delivery
  ON mailbox_messages(employee_id, delivered_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS campaign_targets (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  campaign_id TEXT NOT NULL REFERENCES campaigns(id),
  employee_id TEXT NOT NULL REFERENCES employees(employee_id),
  token       TEXT NOT NULL UNIQUE,                  -- 明文隨機 token（32 hex），landing/event 以此反查
  UNIQUE(campaign_id, employee_id)                   -- 防止重複 simulate 建立同一 target
);
CREATE INDEX IF NOT EXISTS idx_targets_campaign ON campaign_targets(campaign_id);

CREATE TABLE IF NOT EXISTS tracking_events (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  campaign_id   TEXT NOT NULL REFERENCES campaigns(id),
  employee_id   TEXT NOT NULL REFERENCES employees(employee_id),
  event_type    TEXT NOT NULL,                       -- opened|clicked|form_attempted|training_viewed
  occurred_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  metadata_json TEXT NOT NULL DEFAULT '{}',          -- 不含表單原始輸入值
  UNIQUE(campaign_id, employee_id, event_type)       -- 冪等：每 target 每種 event 僅一筆，POST /events 用 INSERT OR IGNORE
);
CREATE INDEX IF NOT EXISTS idx_events_campaign ON tracking_events(campaign_id);
