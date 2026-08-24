const state = { cases: [], selected: null, detail: null, filter: "" };
const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];
const statusNames = {
  monitoring: "监测中",
  remediation: "整改中",
  ready_for_review: "待独立复核",
  review_rejected: "复核退回",
  acceptance_frozen: "验收已冻结"
};
const eventNames = {
  "case.created": "案件建档与基线锁定",
  "monitoring.recorded": "现场监测证据入账",
  "remediation.retested": "整改复测入账",
  "acceptance.reviewed": "独立复核决定",
  "acceptance.frozen": "验收冻结并签发凭据"
};

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers || {}) }
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error?.message || "请求失败 (" + response.status + ")");
  return payload.data;
}

function commandMeta(role, version = 0) {
  return {
    actor: $("#actor").value.trim(),
    role,
    expectedVersion: version,
    idempotencyKey: crypto.randomUUID()
  };
}

function showToast(message, error = false) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.className = "toast visible" + (error ? " error" : "");
  clearTimeout(showToast.timer);
  showToast.timer = setTimeout(() => toast.classList.remove("visible"), 3600);
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>'"]/g, char => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;"
  })[char]);
}

function formatTime(value) {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit"
  }).format(new Date(value));
}

function localInput(date) {
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

async function loadCases(selectID = state.selected) {
  state.cases = await api("/api/cases");
  renderCases();
  $("#ledgerState").textContent = "本地账本 · " + state.cases.length + " 案";
  if (selectID && state.cases.some(item => item.id === selectID)) await selectCase(selectID);
  else if (!state.selected && state.cases.length) await selectCase(state.cases[0].id);
}

function renderCases() {
  const query = state.filter.toLowerCase();
  const items = state.cases.filter(item => (item.name + " " + item.siteCode).toLowerCase().includes(query));
  $("#caseList").innerHTML = items.length
    ? items.map(item =>
      '<article class="case-card ' + (item.id === state.selected ? "active" : "") + '" data-case-id="' + escapeHTML(item.id) + '" tabindex="0" role="button" aria-label="查看 ' + escapeHTML(item.name) + '">' +
      '<div class="case-card-top"><strong>' + escapeHTML(item.siteCode) + '</strong><span class="status-pill" data-tone="' + statusTone(item.status) + '">' + (statusNames[item.status] || item.status) + '</span></div>' +
      '<p>' + escapeHTML(item.name) + '</p><small>' + escapeHTML(item.habitatType) + ' · v' + item.version + '</small></article>'
    ).join("")
    : '<p class="event-body">没有匹配的案件。</p>';
  $$(".case-card").forEach(card => {
    const open = () => selectCase(card.dataset.caseId);
    card.addEventListener("click", open);
    card.addEventListener("keydown", event => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        open();
      }
    });
  });
}

async function selectCase(id) {
  state.selected = id;
  state.detail = await api("/api/cases/" + encodeURIComponent(id));
  renderCases();
  renderDetail();
}

function statusTone(status) {
  if (status === "remediation" || status === "review_rejected") return "alert";
  if (status === "acceptance_frozen") return "dark";
  return "sea";
}

function renderDetail() {
  const item = state.detail.case;
  $("#emptyState").hidden = true;
  $("#caseDetail").hidden = false;
  $("#detailSite").textContent = item.siteCode;
  $("#detailHabitat").textContent = item.habitatType;
  $("#detailName").textContent = item.name;
  $("#detailBaseline").textContent = "基线版本 " + item.baselineVersion + " · 案件版本 v" + item.version;
  $("#detailStatus").textContent = statusNames[item.status] || item.status;
  $("#detailStatus").dataset.tone = statusTone(item.status);
  const open = item.remediations.filter(action => action.status === "open");
  const passed = item.monitoring.filter(record => record.status === "pass" || record.status === "retest_pass");
  $("#metrics").innerHTML =
    metric("基线指标", Object.keys(item.baseline).length) +
    metric("监测证据", item.monitoring.length) +
    metric("待整改", open.length) +
    metric("通过记录", passed.length);
  renderRemediations(open);
  renderTimeline(state.detail.timeline);
  renderBaseline(item.baseline);
  $("#openMonitoring").disabled = item.status === "acceptance_frozen";
  $("#openAcceptance").disabled = item.status === "acceptance_frozen";
}

function metric(label, value) {
  return '<div class="metric"><small>' + label + '</small><strong>' + value + '</strong></div>';
}

function renderRemediations(open) {
  $("#remediationBand").hidden = open.length === 0;
  $("#remediationList").innerHTML = open.map(action =>
    '<article class="remediation-item"><div><strong>' + escapeHTML(action.issueType.toUpperCase()) + ' 偏差</strong>' +
    '<small>责任人 ' + escapeHTML(action.owner) + ' · 截止 ' + formatTime(action.dueAt) + '</small></div>' +
    '<p>' + escapeHTML(action.action) + '</p><button class="secondary" type="button" data-retest="' + escapeHTML(action.id) + '">提交复测</button></article>'
  ).join("");
  $$("[data-retest]").forEach(button => {
    button.addEventListener("click", () => openRetest(button.dataset.retest));
  });
}

function renderTimeline(events) {
  $("#eventCount").textContent = events.length + " 个账本事件";
  $("#timeline").innerHTML = [...events].reverse().map(event => {
    const data = event.data || {};
    const record = data.record;
    const hit = record?.ruleHit;
    const tone = event.type.includes("frozen")
      ? "dark"
      : (record?.status?.includes("required") || record?.status?.includes("failed") || data.review?.decision === "rejected" ? "alert" : "sea");
    const hitHTML = hit
      ? '<div class="rule-hit"><strong>' + escapeHTML(hit.riskLevel.toUpperCase()) + '</strong> ' + escapeHTML(hit.explanation) + '<br>' + escapeHTML(hit.suggestion) + '</div>'
      : "";
    return '<li class="timeline-item" data-tone="' + tone + '"><div class="event-head"><strong>' +
      (eventNames[event.type] || event.type) + '</strong><time>' + formatTime(event.occurredAt) +
      '</time></div><div class="event-body">' + eventDescription(event.type, data) + '<br>执行人：' +
      escapeHTML(event.actor) + ' · 版本 v' + event.version + '</div>' + hitHTML + '</li>';
  }).join("");
}

function eventDescription(type, data) {
  if (type === "case.created") return "基线版本 " + escapeHTML(data.case?.baselineVersion);
  if (type === "monitoring.recorded") {
    return escapeHTML(data.record?.indicator) + " = " + data.record?.observedValue + " " +
      escapeHTML(data.record?.unit) + "；" + escapeHTML(data.record?.evidenceNote);
  }
  if (type === "remediation.retested") {
    return escapeHTML(data.record?.indicator) + " 复测为 " + data.record?.observedValue + " " +
      escapeHTML(data.record?.unit) + "；" + (data.closed ? "整改已关闭" : "仍需整改");
  }
  if (type === "acceptance.reviewed") {
    return escapeHTML(data.review?.decision) + "；" + escapeHTML(data.review?.reviewNote);
  }
  if (type === "acceptance.frozen") {
    const code = data.certificate?.credentialCode || "";
    return "凭据 " + escapeHTML(code) + ' · <a href="/certificates/' +
      encodeURIComponent(code) + '" target="_blank" rel="noopener">查看可验证凭据</a>';
  }
  return "事件已入账";
}

function renderBaseline(baseline) {
  const items = Object.values(baseline).sort((a, b) => a.indicator.localeCompare(b.indicator, "zh-CN"));
  $("#baselineBody").innerHTML = items.map(item =>
    "<tr><td><strong>" + escapeHTML(item.indicator) + "</strong></td><td>" + item.minimum +
    "</td><td>" + item.maximum + "</td><td>" + escapeHTML(item.unit) + "</td></tr>"
  ).join("");
}

function addBaselineRow(values = {}) {
  const row = document.createElement("div");
  row.className = "baseline-row";
  row.innerHTML =
    '<input aria-label="指标名称" name="indicator" required placeholder="指标名称" value="' + escapeHTML(values.indicator || "") + '">' +
    '<input aria-label="允许下限" name="minimum" type="number" step="any" required placeholder="下限" value="' + (values.minimum ?? "") + '">' +
    '<input aria-label="允许上限" name="maximum" type="number" step="any" required placeholder="上限" value="' + (values.maximum ?? "") + '">' +
    '<input aria-label="指标单位" name="unit" required placeholder="单位" value="' + escapeHTML(values.unit || "") + '">' +
    '<button class="icon-button" type="button" aria-label="删除指标">×</button>';
  $(".icon-button", row).addEventListener("click", () => {
    if ($$(".baseline-row").length > 1) row.remove();
  });
  $("#baselineRows").append(row);
}

function openDialog(id) { $(id).showModal(); }
function closeDialog(dialog) { dialog.close(); }

function openMonitoring() {
  const item = state.detail.case;
  $("#indicatorSelect").innerHTML = Object.values(item.baseline).map(range =>
    '<option value="' + escapeHTML(range.indicator) + '" data-unit="' + escapeHTML(range.unit) + '">' +
    escapeHTML(range.indicator) + " · " + range.minimum + "–" + range.maximum + " " + escapeHTML(range.unit) + "</option>"
  ).join("");
  const now = new Date();
  $("[name=capturedAt]", $("#monitoringForm")).value = localInput(now);
  const due = new Date(now);
  due.setDate(due.getDate() + 7);
  $("[name=remediationDueAt]", $("#monitoringForm")).value = localInput(due);
  openDialog("#monitoringDialog");
}

function openRetest(actionID) {
  const action = state.detail.case.remediations.find(item => item.id === actionID);
  const original = state.detail.case.monitoring.find(item => item.id === action.monitoringID);
  const form = $("#retestForm");
  form.elements.actionId.value = actionID;
  $("#retestContext").innerHTML =
    "<strong>" + escapeHTML(original.indicator) + " · 原观测 " + original.observedValue + " " + escapeHTML(original.unit) +
    "</strong><span>目标范围 " + original.expectedRange.minimum + "–" + original.expectedRange.maximum + " " +
    escapeHTML(original.unit) + " · 责任人 " + escapeHTML(action.owner) + "</span>";
  $("#actor").value = action.owner;
  openDialog("#retestDialog");
}

async function submitCreate(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const data = new FormData(form);
  const rows = $$(".baseline-row", form).map(row => ({
    indicator: $("[name=indicator]", row).value,
    minimum: Number($("[name=minimum]", row).value),
    maximum: Number($("[name=maximum]", row).value),
    unit: $("[name=unit]", row).value
  }));
  const body = {
    ...commandMeta("monitor"),
    name: data.get("name"),
    siteCode: data.get("siteCode"),
    habitatType: data.get("habitatType"),
    baseline: rows
  };
  await submitAction(form, () => api("/api/cases", {
    method: "POST", body: JSON.stringify(body)
  }), "案件已建立，基线已锁定");
}

async function submitMonitoring(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const data = new FormData(form);
  const option = $("#indicatorSelect").selectedOptions[0];
  const item = state.detail.case;
  const body = {
    ...commandMeta("monitor", item.version),
    indicator: data.get("indicator"),
    observedValue: Number(data.get("observedValue")),
    unit: option.dataset.unit,
    evidenceNote: data.get("evidenceNote"),
    capturedBy: $("#actor").value.trim(),
    capturedAt: new Date(data.get("capturedAt")).toISOString(),
    remediationOwner: data.get("remediationOwner"),
    remediationDueAt: new Date(data.get("remediationDueAt")).toISOString()
  };
  await submitAction(form, () => api("/api/cases/" + item.id + "/monitoring", {
    method: "POST", body: JSON.stringify(body)
  }), "监测证据已判定并入账");
}

async function submitRetest(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const data = new FormData(form);
  const item = state.detail.case;
  const action = item.remediations.find(value => value.id === data.get("actionId"));
  const body = {
    ...commandMeta("remediator", item.version),
    owner: action.owner,
    observedValue: Number(data.get("observedValue")),
    evidenceNote: data.get("evidenceNote")
  };
  await submitAction(form, () => api("/api/cases/" + item.id + "/remediations/" + action.id + "/retest", {
    method: "POST", body: JSON.stringify(body)
  }), "复测已入账，整改状态已更新");
}

async function submitAcceptance(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const data = new FormData(form);
  const item = state.detail.case;
  const body = {
    ...commandMeta("reviewer", item.version),
    reviewer: $("#actor").value.trim(),
    decision: data.get("decision"),
    reviewNote: data.get("reviewNote")
  };
  const message = data.get("decision") === "accepted" ? "验收已冻结，放行凭据已签发" : "复核退回决定已入账";
  await submitAction(form, () => api("/api/cases/" + item.id + "/acceptance", {
    method: "POST", body: JSON.stringify(body)
  }), message);
}

async function submitAction(form, operation, success) {
  const button = $("button[type=submit]", form);
  button.disabled = true;
  try {
    const result = await operation();
    closeDialog(form.closest("dialog"));
    form.reset();
    await loadCases(result.case.id);
    showToast(success);
  } catch (error) {
    showToast(error.message, true);
  } finally {
    button.disabled = false;
  }
}

function bind() {
  $("#openCreate").addEventListener("click", () => openDialog("#createDialog"));
  $$("[data-open-create]").forEach(button => button.addEventListener("click", () => openDialog("#createDialog")));
  $$("[data-close]").forEach(button => {
    button.addEventListener("click", () => closeDialog(button.closest("dialog")));
  });
  $("#addBaseline").addEventListener("click", () => addBaselineRow());
  $("#openMonitoring").addEventListener("click", openMonitoring);
  $("#openAcceptance").addEventListener("click", () => {
    const item = state.detail.case;
    const open = item.remediations.filter(action => action.status === "open").length;
    $("#reviewWarning").textContent = open
      ? "当前仍有 " + open + " 项整改未关闭，系统会阻止冻结。"
      : "所有整改已闭环。复核员不得与任何证据采集人相同，验收通过后结果不可变。";
    openDialog("#acceptanceDialog");
  });
  $("#caseSearch").addEventListener("input", event => {
    state.filter = event.target.value;
    renderCases();
  });
  $$(".tab").forEach(tab => tab.addEventListener("click", () => {
    $$(".tab").forEach(item => {
      item.classList.toggle("active", item === tab);
      item.setAttribute("aria-selected", item === tab);
    });
    $("#evidencePanel").hidden = tab.dataset.tab !== "evidence";
    $("#baselinePanel").hidden = tab.dataset.tab !== "baseline";
  }));
  $("#createForm").addEventListener("submit", submitCreate);
  $("#monitoringForm").addEventListener("submit", submitMonitoring);
  $("#retestForm").addEventListener("submit", submitRetest);
  $("#acceptanceForm").addEventListener("submit", submitAcceptance);
}

addBaselineRow({ indicator: "植被覆盖率", minimum: 70, maximum: 100, unit: "%" });
bind();
loadCases().catch(error => showToast(error.message, true));
