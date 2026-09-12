const messages = [
  {
    id: "cathay-summary",
    sender: "國泰世華銀行",
    address: "statement@notice.example.test",
    subject: "國泰世華銀行消費彙整通知（請勿直接回覆）",
    preview: "本月消費彙整通知已產生，請登入官方服務確認。",
    date: "9月12日",
    fullDate: "2026年9月12日 13:14",
    unread: true,
    starred: false,
    label: "updates",
    accent: "#2b796d",
    type: "summary"
  },
  {
    id: "toyota-rental",
    sender: "TOYOTA 租車",
    address: "reservation@travel.example.test",
    subject: "[TOYOTA租車]　感謝您預約租車。",
    preview: "感謝您的預約，行程資訊與取車提醒已整理完成。",
    date: "9月12日",
    fullDate: "2026年9月12日 11:42",
    unread: true,
    starred: true,
    label: "travel",
    accent: "#b84d42",
    type: "travel"
  },
  {
    id: "pytorchcon",
    sender: "PyTorch Foundation",
    address: "program@conference.example.test",
    subject: "🤖 Agents Have Entered the #PyTorchCon North America Program",
    preview: "Explore agentic AI sessions spanning training, robotics, and governance.",
    date: "9月12日",
    fullDate: "2026年9月12日 09:35",
    unread: true,
    starred: false,
    label: "updates",
    accent: "#e36a46",
    type: "newsletter"
  },
  {
    id: "google-download",
    sender: "Google Takeout",
    address: "no-reply@accounts.example.test",
    subject: "We've scheduled your Google data download",
    preview: "Your requested archive is being prepared. Review the activity if this wasn't you.",
    date: "9月12日",
    fullDate: "2026年9月12日 06:32",
    unread: true,
    starred: false,
    label: "updates",
    accent: "#4776d0",
    type: "simulation"
  },
  {
    id: "nintendo-direct",
    sender: "Nintendo",
    address: "news@gaming.example.test",
    subject: "【最新ソフト情報】「Nintendo Direct 2026.9.9」を公開。",
    preview: "Nintendo Switch 2 軟體最新消息與節目內容已公開。",
    date: "9月11日",
    fullDate: "2026年9月11日 16:39",
    unread: true,
    starred: false,
    label: "updates",
    accent: "#d84949",
    type: "newsletter"
  },
  {
    id: "indeed-career",
    sender: "Indeed キャリアガイド編集部",
    address: "career@jobs.example.test",
    subject: "転職準備、今から始めよう＜Indeed キャリアガイド編集部＞",
    preview: "キャリアの棚卸しから応募準備まで、今週のガイドをお届けします。",
    date: "9月11日",
    fullDate: "2026年9月11日 11:17",
    unread: false,
    starred: false,
    label: "work",
    accent: "#356ac3",
    type: "newsletter"
  },
  {
    id: "github-pr",
    sender: "mistysya",
    address: "notifications@code.example.test",
    subject: "Re: [mistysya/hackathon_2026] feat: add role B landing and event tracking (PR #3)",
    preview: "A new review comment was added to the pull request.",
    date: "9月11日",
    fullDate: "2026年9月11日 22:33",
    unread: false,
    starred: true,
    label: "work",
    accent: "#4d596a",
    type: "work"
  },
  {
    id: "openai-signin",
    sender: "OpenAI",
    address: "security@account.example.test",
    subject: "New sign-in to your OpenAI account",
    preview: "We noticed a new sign-in. If this was you, no action is needed.",
    date: "9月10日",
    fullDate: "2026年9月10日 18:56",
    unread: false,
    starred: false,
    label: "updates",
    accent: "#2e8373",
    type: "security"
  },
  {
    id: "job-alert",
    sender: "My104會員中心",
    address: "jobs@career.example.test",
    subject: "你關注的公司刊登新職務囉！",
    preview: "本週有新的技術職缺符合你設定的關注條件。",
    date: "9月10日",
    fullDate: "2026年9月10日 10:51",
    unread: false,
    starred: false,
    label: "work",
    accent: "#e09836",
    type: "work"
  },
  {
    id: "line-bank-business",
    sender: "LINE Bank連線商業銀行",
    address: "news@finance.example.test",
    subject: "公司開戶💥線上辦、審核快💥活存1.5%無上限 >>",
    preview: "企業金融服務方案與本月活動資訊。",
    date: "9月10日",
    fullDate: "2026年9月10日 09:54",
    unread: false,
    starred: false,
    label: "updates",
    accent: "#42a77a",
    type: "newsletter"
  },
  {
    id: "linux-education",
    sender: "Linux Foundation Education",
    address: "education@learning.example.test",
    subject: "Learn it. Build it. Prove it. 3 days only.",
    preview: "Bundle a certification with an annual learning subscription.",
    date: "9月9日",
    fullDate: "2026年9月9日 10:03",
    unread: false,
    starred: false,
    label: "updates",
    accent: "#31435d",
    type: "newsletter"
  },
  {
    id: "mega-transfer",
    sender: "兆豐銀行",
    address: "notice@banking.example.test",
    subject: "兆豐銀行全國性繳費【    　轉出交易　  】通知",
    preview: "系統已產生一則交易通知；本展示不包含任何真實交易資料。",
    date: "9月8日",
    fullDate: "2026年9月8日 13:14",
    unread: false,
    starred: false,
    label: "updates",
    accent: "#b18a35",
    type: "summary"
  },
  {
    id: "cathay-statement",
    sender: "國泰世華銀行",
    address: "statement@notice.example.test",
    subject: "國泰世華銀行綜合對帳單",
    preview: "您的電子綜合對帳單已產生；請由官方應用程式查看。",
    date: "9月7日",
    fullDate: "2026年9月7日 10:47",
    unread: false,
    starred: false,
    label: "updates",
    accent: "#2b796d",
    type: "summary"
  }
];

const labelNames = {
  work: "工作",
  travel: "旅行",
  updates: "最新消息"
};

const state = {
  query: "",
  unreadOnly: false,
  folder: "inbox",
  label: null,
  selectedId: null
};

const mailList = document.querySelector("#mailList");
const emptyState = document.querySelector("#emptyState");
const inboxPanel = document.querySelector("#inboxPanel");
const messagePanel = document.querySelector("#messagePanel");
const searchInput = document.querySelector("#searchInput");
const rangeLabel = document.querySelector("#rangeLabel");
const folderTitle = document.querySelector("#folderTitle");
const filterButton = document.querySelector("#filterButton");
const selectAll = document.querySelector("#selectAll");
const sidebar = document.querySelector("#sidebar");
const menuButton = document.querySelector("#menuButton");
const toast = document.querySelector("#toast");
let toastTimer;

function visibleMessages() {
  const query = state.query.trim().toLocaleLowerCase();
  return messages.filter((message) => {
    const matchesQuery = !query || [message.sender, message.subject, message.preview]
      .join(" ")
      .toLocaleLowerCase()
      .includes(query);
    const matchesUnread = !state.unreadOnly || message.unread;
    const matchesFolder = state.folder === "inbox" || (state.folder === "starred" && message.starred);
    const matchesLabel = !state.label || message.label === state.label;
    return matchesQuery && matchesUnread && matchesFolder && matchesLabel;
  });
}

function renderList() {
  const visible = visibleMessages();
  mailList.replaceChildren(...visible.map(createMessageRow));
  emptyState.hidden = visible.length !== 0;
  rangeLabel.textContent = visible.length ? `1–${visible.length}，共 ${visible.length} 封` : "0 封";
  document.querySelector("#inboxCount").textContent = messages.filter((message) => message.unread).length;
  selectAll.checked = false;
}

function createMessageRow(message) {
  const row = document.createElement("div");
  row.className = `mail-row${message.unread ? " is-unread" : ""}${message.isNew ? " is-new" : ""}`;
  row.dataset.id = message.id;
  row.setAttribute("role", "listitem");
  row.tabIndex = 0;

  const checkWrap = document.createElement("label");
  checkWrap.className = "mail-check";
  checkWrap.title = "選取";
  const checkbox = document.createElement("input");
  checkbox.type = "checkbox";
  checkbox.setAttribute("aria-label", `選取 ${message.subject}`);
  checkbox.addEventListener("click", (event) => event.stopPropagation());
  checkWrap.append(checkbox);
  checkWrap.addEventListener("click", (event) => event.stopPropagation());

  const star = document.createElement("button");
  star.className = `star-button${message.starred ? " is-starred" : ""}`;
  star.type = "button";
  star.textContent = message.starred ? "★" : "☆";
  star.setAttribute("aria-label", message.starred ? "移除星號" : "加上星號");
  star.addEventListener("click", (event) => {
    event.stopPropagation();
    message.starred = !message.starred;
    renderList();
    showToast(message.starred ? "已加上星號" : "已移除星號");
  });

  const sender = document.createElement("span");
  sender.className = "mail-sender";
  sender.textContent = message.sender;

  const copy = document.createElement("span");
  copy.className = "mail-copy";
  const subject = document.createElement("span");
  subject.className = "mail-subject";
  subject.textContent = message.subject;
  const preview = document.createElement("span");
  preview.className = "mail-preview";
  preview.textContent = `— ${message.preview}`;
  copy.append(subject, preview);

  const date = document.createElement("time");
  date.className = "mail-date";
  date.textContent = message.date;

  row.append(checkWrap, star, sender, copy, date);
  row.addEventListener("click", () => openMessage(message.id));
  row.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openMessage(message.id);
    }
  });
  return row;
}

function openMessage(id, updateHash = true) {
  const message = messages.find((item) => item.id === id);
  if (!message) return;

  state.selectedId = id;
  message.unread = false;
  document.querySelector("#messageSubject").textContent = message.subject;
  document.querySelector("#messageSender").textContent = message.sender;
  document.querySelector("#senderAddress").textContent = `<${message.address}>`;
  document.querySelector("#messageDate").textContent = message.fullDate;
  document.querySelector("#messageLabel").textContent = labelNames[message.label] || "收件匣";

  const avatar = document.querySelector("#senderAvatar");
  avatar.textContent = avatarLetter(message.sender);
  avatar.style.background = message.accent;

  const messageStar = document.querySelector(".message-star");
  messageStar.textContent = message.starred ? "★" : "☆";
  messageStar.classList.toggle("is-starred", message.starred);
  messageStar.onclick = () => {
    message.starred = !message.starred;
    openMessage(message.id, false);
    showToast(message.starred ? "已加上星號" : "已移除星號");
  };

  const body = document.querySelector("#messageBody");
  body.innerHTML = buildBody(message);
  bindMessageActions(message);
  notifyCampaignOpened(message);

  inboxPanel.hidden = true;
  messagePanel.hidden = false;
  if (updateHash) history.pushState({ id }, "", `#mail=${encodeURIComponent(id)}`);
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function avatarLetter(sender) {
  const cleaned = sender.replace(/[^A-Za-z0-9\u3400-\u9fff\u3040-\u30ff]/g, "");
  return cleaned.charAt(0).toUpperCase() || "M";
}

function buildBody(message) {
  if (message.type === "generated-campaign") {
    return `
      <div class="generated-email">
        <iframe id="generatedCampaignFrame" title="${escapeHtml(message.subject)}" sandbox="allow-popups allow-popups-to-escape-sandbox"></iframe>
        <p>這封郵件由安全演練管理台產生，郵件內容顯示於隔離的預覽框架中。</p>
      </div>`;
  }

  if (message.type === "simulation") {
    return `
      <div class="email-card">
        <div class="email-card__header">
          <small>Account archive</small>
          <h2>Your data download is being prepared</h2>
        </div>
        <div class="email-card__body">
          <p>Hello,</p>
          <p>An archive of your account data was recently scheduled. The download will be available for a limited time after processing is complete.</p>
          <div class="email-card__detail">
            <span>Request</span><strong>Account data archive</strong>
            <span>Device</span><strong>Desktop browser</strong>
            <span>Status</span><strong>Processing</strong>
          </div>
          <p>If you did not request this archive, review the activity now.</p>
          <button class="email-cta" id="reviewActivity" type="button">Review activity</button>
          <p class="email-footer">This synthetic notification was sent to a demonstration inbox. The button does not visit an external site or request credentials.</p>
        </div>
      </div>
      <div id="trainingReveal"></div>`;
  }

  if (message.type === "travel") {
    return `
      <p>您好，</p>
      <p>感謝您預約租車服務。您的測試行程已保留，請在取車前確認所需證件與營業時間。</p>
      <div class="plain-notice">
        <strong>預約狀態：已確認</strong><br>
        取車地點：範例車站門市<br>
        預約編號：DEMO-2026-0912
      </div>
      <p>這封展示郵件未包含真實姓名、門市、日期、價格或預約資訊。</p>
      <p class="email-signoff">TOYOTA 租車服務團隊</p>`;
  }

  if (message.type === "work") {
    return `
      <p>Hi,</p>
      <p>${escapeHtml(message.preview)}</p>
      <div class="plain-notice">
        <strong>本週更新</strong><br>
        團隊已新增一項待確認事項。請從您平常使用的官方工作平台開啟通知，不要直接使用郵件中的未知連結。
      </div>
      <p>Thanks,<br>${escapeHtml(message.sender)}</p>`;
  }

  if (message.type === "security") {
    return `
      <div class="email-card">
        <div class="email-card__header" style="background:linear-gradient(135deg,#247263,#3f9a87)">
          <small>Security notification</small>
          <h2>New sign-in detected</h2>
        </div>
        <div class="email-card__body">
          <p>We noticed a new sign-in to your account from a demonstration device.</p>
          <div class="email-card__detail"><span>Device</span><strong>Demo Browser</strong><span>Location</span><strong>Example City</strong></div>
          <p>If this wasn't you, open the official application directly and review your active sessions.</p>
          <p class="email-footer">No real account, location, IP address, or security link is included.</p>
        </div>
      </div>`;
  }

  if (message.type === "summary") {
    return `
      <p>親愛的客戶您好：</p>
      <p>${escapeHtml(message.preview)}</p>
      <div class="plain-notice">
        為保護您的資料，本展示不顯示姓名、帳號、卡號、金額、交易明細或對帳單連結。查詢財務資訊時，請直接開啟金融機構的官方應用程式。
      </div>
      <p class="email-signoff">此為系統自動發送的合成通知，請勿直接回覆。</p>`;
  }

  return `
    <p>Hello,</p>
    <h2>${escapeHtml(message.subject)}</h2>
    <p>${escapeHtml(message.preview)}</p>
    <p>Here are this week's selected updates, sessions, and learning resources. Visit the publisher's official website directly to learn more.</p>
    <div class="plain-notice">This is synthetic newsletter copy created for the mailbox presentation. Original email content and tracking links were not copied.</div>
    <p class="email-signoff">— ${escapeHtml(message.sender)}</p>`;
}

function bindMessageActions(message) {
  const generatedFrame = document.querySelector("#generatedCampaignFrame");
  if (generatedFrame) {
    const emailHtml = message.emailHtml.replaceAll("{{landingUrl}}", message.landingUrl);
    generatedFrame.srcdoc = addExternalLinkTarget(emailHtml);
  }

  const reviewButton = document.querySelector("#reviewActivity");
  if (!reviewButton) return;
  reviewButton.addEventListener("click", () => {
    document.querySelector("#trainingReveal").innerHTML = `
      <section class="training-reveal" aria-live="polite">
        <div class="training-reveal__top">
          <div class="training-reveal__icon">!</div>
          <div><h3>這是一封安全意識演練郵件</h3><p>你已安全完成「點擊」階段，沒有任何資料被送出。</p></div>
        </div>
        <div class="training-reveal__body">
          <p><strong>在點擊前，可以注意這些線索：</strong></p>
          <ul class="red-flags">
            <li>寄件網域是 <code>example.test</code>，而不是服務的官方網域。</li>
            <li>訊息利用帳戶資料外洩的急迫感，引導你立刻採取行動。</li>
            <li>更安全的做法是自行開啟官方網站或應用程式確認活動。</li>
          </ul>
          <div class="event-progress" aria-label="演練進度">
            <span class="is-complete">開啟郵件</span><span class="is-complete">點擊連結</span><span>嘗試輸入</span><span>完成學習</span>
          </div>
        </div>
      </section>`;
    reviewButton.disabled = true;
    reviewButton.textContent = "Activity reviewed";
    showToast("已記錄模擬點擊事件（僅限本機展示）");
    document.querySelector("#trainingReveal").scrollIntoView({ behavior: "smooth", block: "center" });
  });
}

function addExternalLinkTarget(emailHtml) {
  const base = '<base target="_blank">';
  if (/<head(?:\s[^>]*)?>/i.test(emailHtml)) {
    return emailHtml.replace(/<head(\s[^>]*)?>/i, (head) => `${head}${base}`);
  }
  if (/<html(?:\s[^>]*)?>/i.test(emailHtml)) {
    return emailHtml.replace(/<html(\s[^>]*)?>/i, (html) => `${html}<head>${base}</head>`);
  }
  return `<!doctype html><html><head>${base}</head><body>${emailHtml}</body></html>`;
}

function notifyCampaignOpened(message) {
  if (!message.isCampaign || message.openedEventSent) return;
  window.clearTimeout(message.openedRetryTimer);
  if (!message.deliverySource || message.deliverySource.closed) return;
  message.deliverySource.postMessage({
    type: "simsafe:mailbox-event",
    campaignId: message.campaignId,
    token: message.token,
    eventType: "opened"
  }, message.deliveryOrigin);
  // Only an ACK after a successful API write completes delivery. The API deduplicates retries.
  message.openedRetryTimer = window.setTimeout(() => notifyCampaignOpened(message), 2000);
}

function receiveEventAcknowledgement(event) {
  if (event.data?.type !== "simsafe:event-ack" || event.data.eventType !== "opened") return;
  const message = messages.find((item) => item.isCampaign && item.campaignId === event.data.campaignId && item.token === event.data.token);
  if (!message || event.source !== message.deliverySource || event.origin !== message.deliveryOrigin) return;
  message.openedEventSent = true;
  window.clearTimeout(message.openedRetryTimer);
}

function escapeHtml(value) {
  return value.replace(/[&<>'"]/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "'": "&#039;",
    '"': "&quot;"
  })[character]);
}

function closeMessage(updateHash = true) {
  state.selectedId = null;
  messagePanel.hidden = true;
  inboxPanel.hidden = false;
  renderList();
  if (updateHash) history.pushState({}, "", window.location.pathname + window.location.search);
}

function showToast(text) {
  clearTimeout(toastTimer);
  toast.textContent = text;
  toast.classList.add("is-visible");
  toastTimer = setTimeout(() => toast.classList.remove("is-visible"), 2200);
}

searchInput.addEventListener("input", () => {
  state.query = searchInput.value;
  if (state.selectedId) closeMessage();
  renderList();
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    searchInput.focus();
  }
  if (event.key === "Escape" && state.selectedId) closeMessage();
});

filterButton.addEventListener("click", () => {
  state.unreadOnly = !state.unreadOnly;
  filterButton.setAttribute("aria-pressed", String(state.unreadOnly));
  renderList();
});

selectAll.addEventListener("change", () => {
  mailList.querySelectorAll("input[type='checkbox']").forEach((checkbox) => {
    checkbox.checked = selectAll.checked;
  });
});

document.querySelector("#refreshButton").addEventListener("click", () => {
  const button = document.querySelector("#refreshButton");
  button.animate([{ transform: "rotate(0)" }, { transform: "rotate(360deg)" }], { duration: 450 });
  showToast("信箱已是最新狀態");
});

document.querySelector("#backButton").addEventListener("click", () => closeMessage());

document.querySelectorAll(".folder").forEach((button) => {
  button.addEventListener("click", () => {
    document.querySelectorAll(".folder").forEach((folder) => folder.classList.remove("is-active"));
    button.classList.add("is-active");
    state.folder = button.dataset.folder;
    state.label = null;
    folderTitle.textContent = button.querySelector("span").textContent;
    closeMessage();
    renderList();
    sidebar.classList.remove("is-open");
    menuButton.setAttribute("aria-expanded", "false");
  });
});

document.querySelectorAll(".labels button").forEach((button) => {
  button.addEventListener("click", () => {
    state.label = button.dataset.label;
    state.folder = "inbox";
    document.querySelectorAll(".folder").forEach((folder) => folder.classList.remove("is-active"));
    folderTitle.textContent = labelNames[state.label];
    closeMessage();
    renderList();
    sidebar.classList.remove("is-open");
  });
});

menuButton.addEventListener("click", () => {
  const open = sidebar.classList.toggle("is-open");
  menuButton.setAttribute("aria-expanded", String(open));
});

document.querySelector("#composeButton").addEventListener("click", () => {
  document.querySelector("#composeDialog").showModal();
});

document.querySelector(".compose-dialog .send-button").addEventListener("click", () => {
  showToast("演示模式：郵件未寄出");
});

window.addEventListener("popstate", () => {
  const id = new URLSearchParams(window.location.hash.slice(1)).get("mail");
  if (id) openMessage(id, false);
  else closeMessage(false);
});

function isAllowedDemoOrigin(origin) {
  if (origin === window.location.origin) return true;
  try {
    return ["localhost", "127.0.0.1"].includes(new URL(origin).hostname);
  } catch {
    return false;
  }
}

function textPreview(emailHtml) {
  const document = new DOMParser().parseFromString(emailHtml, "text/html");
  const text = document.body.textContent?.replace(/\s+/g, " ").trim() || "安全演練郵件已送達。";
  return text.length > 92 ? `${text.slice(0, 92)}…` : text;
}

function receiveGeneratedCampaign(event) {
  if (!window.opener || event.source !== window.opener || !isAllowedDemoOrigin(event.origin) || event.data?.type !== "simsafe:campaign-delivered") return;
  const { campaign, target } = event.data;
  if (!campaign || !target || typeof campaign.campaignId !== "string" || typeof campaign.subject !== "string" || typeof campaign.emailHtml !== "string") return;
  if (typeof target.token !== "string" || typeof target.landingUrl !== "string") return;
  const acknowledgeDelivery = () => event.source.postMessage({
    type: "simsafe:delivery-ack",
    campaignId: campaign.campaignId,
    token: target.token
  }, event.origin);
  const existing = messages.find((message) => message.campaignId === campaign.campaignId);
  if (existing) {
    if (existing.token === target.token) acknowledgeDelivery();
    return;
  }

  const generatedMessage = {
    id: `campaign-${campaign.campaignId}`,
    campaignId: campaign.campaignId,
    token: target.token,
    landingUrl: target.landingUrl,
    sender: campaign.landingConfig?.brand || "Security Training",
    address: "notification@campaign.example.test",
    subject: campaign.subject,
    preview: textPreview(campaign.emailHtml),
    date: "現在",
    fullDate: new Intl.DateTimeFormat("zh-TW", { dateStyle: "medium", timeStyle: "short" }).format(new Date()),
    unread: true,
    starred: false,
    label: "work",
    accent: "#0b57d0",
    type: "generated-campaign",
    emailHtml: campaign.emailHtml,
    isCampaign: true,
    isNew: true,
    deliverySource: event.source,
    deliveryOrigin: event.origin
  };

  messages.unshift(generatedMessage);
  renderList();
  acknowledgeDelivery();
  document.title = "(1) 收到新郵件 — Postbox";
  showToast("收到 1 封新郵件");
  window.setTimeout(() => {
    generatedMessage.isNew = false;
    document.querySelector(`[data-id="${generatedMessage.id}"]`)?.classList.remove("is-new");
  }, 1800);
}

window.addEventListener("message", receiveGeneratedCampaign);
window.addEventListener("message", receiveEventAcknowledgement);

renderList();
const initialId = new URLSearchParams(window.location.hash.slice(1)).get("mail");
if (initialId) openMessage(initialId, false);
// This readiness signal contains no campaign data; the admin validates origin and window identity.
window.opener?.postMessage({ type: "simsafe:mailbox-ready" }, "*");
