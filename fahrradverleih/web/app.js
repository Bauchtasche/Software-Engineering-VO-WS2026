// Fahrradverleih – Frontend. Spricht nur mit der Go-API unter /api.

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

// ---------- API ----------
async function api(method, path, body) {
  const opts = { method, headers: {} };
  if (body instanceof FormData) {
    opts.body = body;
  } else if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const res = await fetch("/api" + path, opts);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Fehler " + res.status);
  return data;
}

// ---------- Formatierung ----------
const euro = (cents) => (cents / 100).toLocaleString("de-AT", { style: "currency", currency: "EUR" });
const fmtDate = (iso) =>
  new Date(iso).toLocaleString("de-AT", { weekday: "short", day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
const fmtTime = (iso) => new Date(iso).toLocaleTimeString("de-AT", { hour: "2-digit", minute: "2-digit" });
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

// Date → Wert für <input type="datetime-local"> (lokale Zeit)
function toLocalInput(d) {
  const pad = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
const fromLocalInput = (v) => (v ? new Date(v) : null);

function toast(msg, isError = false) {
  const t = $("#toast");
  t.textContent = msg;
  t.className = "toast" + (isError ? " error" : "");
  t.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => (t.hidden = true), isError ? 5000 : 2500);
}

const state = { bikes: [], persons: [], rentals: [], tariffs: [] };

// ---------- Tabs ----------
function showTab(name) {
  $$(".tabs button").forEach((b) => b.classList.toggle("active", b.dataset.tab === name));
  $$(".tab").forEach((s) => s.classList.toggle("active", s.id === "tab-" + name));
  refresh();
}
$$(".tabs button").forEach((b) => b.addEventListener("click", () => showTab(b.dataset.tab)));
document.addEventListener("click", (e) => {
  const g = e.target.closest("[data-goto]");
  if (g) { e.preventDefault(); showTab(g.dataset.goto); }
});

// ---------- Laden ----------
async function refresh() {
  try {
    const [dash, bikes, persons, rentals] = await Promise.all([
      api("GET", "/dashboard"), api("GET", "/bikes"), api("GET", "/persons"), api("GET", "/rentals"),
    ]);
    Object.assign(state, { bikes, persons, rentals });
    renderDashboard(dash);
    renderBikes();
    renderPersons();
    renderRentals();
    fillPersonSelect();
    loadAvailableBikes();
  } catch (err) {
    toast(err.message, true);
  }
}

// ---------- Dashboard ----------
function renderDashboard(d) {
  $("#d-available").textContent = d.bikesAvailable;
  $("#d-total").textContent = d.bikesTotal;
  $("#d-rented").textContent = d.bikesRented;
  $("#d-overdue").textContent = d.overdue;
  $("#d-reserved").textContent = d.reserved;
  $("#d-persons").textContent = d.persons;
  $("#d-revenue").textContent = euro(d.revenueCents);
  $("#mini-overdue").classList.toggle("alert", d.overdue > 0);

  const item = (r, when, btn) => `
    <li><div><strong>${esc(r.personName)}</strong><div class="meta">${esc(r.bikeLabel)} · ${when}</div></div>${btn}</li>`;
  const returnBtn = (r) => `<button class="small" data-return="${r.id}">Rückgabe</button>`;
  const empty = (t) => `<li class="empty">${t}</li>`;

  $("#d-overdue-list").innerHTML = d.overdueList.length
    ? d.overdueList.map((r) => item(r, "seit " + fmtDate(r.end), returnBtn(r))).join("")
    : empty("Nichts überfällig 👍");
  $("#d-due-list").innerHTML = d.dueToday.length
    ? d.dueToday.map((r) => item(r, "bis " + fmtTime(r.end), returnBtn(r))).join("")
    : empty("Heute keine Rückgaben fällig");
  $("#d-upcoming-list").innerHTML = d.upcoming.length
    ? d.upcoming.map((r) => item(r, "ab " + fmtDate(r.start), `<span class="badge reserviert">reserviert</span>`)).join("")
    : empty("Keine Reservierungen");
}

// ---------- Fahrräder ----------
function renderBikes() {
  const q = $("#bike-search").value.toLowerCase();
  const rows = state.bikes.filter((s) => `${s.brand} ${s.model} ${s.type} ${s.frameNo}`.toLowerCase().includes(q));
  $("#bike-table").innerHTML = rows.map((s) => `
    <tr>
      <td class="num">${s.id}</td>
      <td><strong>${esc(s.brand)}</strong> ${esc(s.model)}<span class="sub">Rahmennr. ${esc(s.frameNo)}</span></td>
      <td>${esc(s.type)}<span class="sub">${euro(s.tariff.hourlyCents)}/h · max. ${euro(s.tariff.dailyCapCents)}/Tag</span></td>
      <td class="num">${esc(s.frameSize)}</td>
      <td><span class="badge ${s.status}">${s.status}</span></td>
      <td><button class="small danger" data-del-bike="${s.id}">Löschen</button></td>
    </tr>`).join("") || `<tr><td colspan="6" class="hint">Keine Fahrräder gefunden.</td></tr>`;
}
$("#bike-search").addEventListener("input", renderBikes);

$("#bike-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const f = e.target;
  try {
    await api("POST", "/bikes", {
      brand: f.brand.value, model: f.model.value, type: f.type.value, frameSize: f.frameSize.value, frameNo: f.frameNo.value,
    });
    toast("Fahrrad angelegt");
    f.brand.value = ""; f.model.value = ""; f.frameNo.value = "";
    refresh();
  } catch (err) { toast(err.message, true); }
});

// ---------- Personen ----------
function renderPersons() {
  const q = $("#person-search").value.toLowerCase();
  const rows = state.persons.filter((p) =>
    `${p.firstName} ${p.lastName} ${p.document.number} ${p.email}`.toLowerCase().includes(q));
  $("#person-table").innerHTML = rows.map((p) => `
    <tr>
      <td><strong>${esc(p.firstName)} ${esc(p.lastName)}</strong></td>
      <td>${esc(p.phone)}<span class="sub">${esc(p.email)}</span></td>
      <td>${esc(p.document.type)}<span class="sub">${esc(p.document.number)}</span></td>
      <td>${p.document.hasFile
        ? `<a href="/api/persons/${p.id}/document" target="_blank" rel="noopener">ansehen</a>`
        : `<button class="small" data-upload="${p.id}">hochladen</button>`}</td>
      <td><button class="small danger" data-del-person="${p.id}">Löschen</button></td>
    </tr>`).join("") || `<tr><td colspan="5" class="hint">Keine Personen gefunden.</td></tr>`;
}
$("#person-search").addEventListener("input", renderPersons);

$("#person-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const f = e.target;
  const file = f.docFile.files[0];
  if (file && file.size > 5 * 1024 * 1024) return toast("Datei ist größer als 5 MB", true);
  try {
    const p = await api("POST", "/persons", {
      firstName: f.firstName.value, lastName: f.lastName.value, phone: f.phone.value, email: f.email.value,
      document: { type: f.docType.value, number: f.docNumber.value },
    });
    if (file) await uploadDocument(p.id, file);
    toast(`${p.firstName} ${p.lastName} angelegt`);
    f.reset();
    refresh();
  } catch (err) { toast(err.message, true); }
});

async function uploadDocument(personId, file) {
  const fd = new FormData();
  fd.append("file", file);
  return api("POST", `/persons/${personId}/document`, fd);
}

// Upload für bestehende Person
let uploadTarget = null;
$("#hidden-upload").addEventListener("change", async (e) => {
  const file = e.target.files[0];
  e.target.value = "";
  if (!file || !uploadTarget) return;
  try {
    await uploadDocument(uploadTarget, file);
    toast("Ausweis hochgeladen");
    refresh();
  } catch (err) { toast(err.message, true); }
});

// ---------- Ausleihen ----------
function fillPersonSelect() {
  const sel = $("#rental-form").personId;
  const prev = sel.value;
  sel.innerHTML = `<option value="">– Person wählen –</option>` + state.persons.map((p) =>
    `<option value="${p.id}">${esc(p.lastName)}, ${esc(p.firstName)} (${esc(p.document.type)} ${esc(p.document.number)})</option>`).join("");
  sel.value = prev;
}

function period() {
  const f = $("#rental-form");
  const start = fromLocalInput(f.start.value);
  const end = fromLocalInput(f.end.value);
  if (!start || !end || end <= start) return null;
  return { start: start.toISOString(), end: end.toISOString() };
}

// Nur Fahrräder anbieten, die im gewählten Zeitraum frei sind.
async function loadAvailableBikes() {
  const sel = $("#rental-form").bikeId;
  const p = period();
  if (!p) {
    sel.innerHTML = `<option value="">– zuerst Zeitraum wählen –</option>`;
    $("#bike-count").textContent = "";
    return updateQuote();
  }
  try {
    const bikes = await api("GET", `/bikes/available?start=${encodeURIComponent(p.start)}&end=${encodeURIComponent(p.end)}`);
    const prev = sel.value;
    sel.innerHTML = `<option value="">– Fahrrad wählen –</option>` + bikes.map((s) =>
      `<option value="${s.id}">${esc(s.brand)} ${esc(s.model)} · ${esc(s.type)} · Größe ${esc(s.frameSize)} · ${euro(s.tariff.hourlyCents)}/h</option>`).join("");
    if (bikes.some((s) => String(s.id) === prev)) sel.value = prev;
    $("#bike-count").textContent = `(${bikes.length} frei im Zeitraum)`;
  } catch (err) {
    $("#bike-count").textContent = "";
  }
  updateQuote();
}

async function updateQuote() {
  const f = $("#rental-form");
  const p = period();
  const box = $("#quote");
  if (!p) { box.textContent = "Ende muss nach dem Beginn liegen."; return; }
  if (!f.bikeId.value) { box.textContent = "Fahrrad wählen, um den Preis zu sehen."; return; }
  try {
    const q = await api("GET", `/quote?bikeId=${f.bikeId.value}&start=${encodeURIComponent(p.start)}&end=${encodeURIComponent(p.end)}`);
    const days = Math.floor(q.hours / 24), rest = q.hours % 24;
    const dur = (days ? `${days} Tag${days > 1 ? "e" : ""} ` : "") + (rest ? `${rest} h` : "");
    box.innerHTML = `Voraussichtlich <strong>${euro(q.priceCents)}</strong><br>
      ${dur.trim()} · ${euro(q.tariff.hourlyCents)}/h, max. ${euro(q.tariff.dailyCapCents)} pro Tag`;
  } catch (err) {
    box.textContent = err.message;
  }
}

function setDuration(hours) {
  const f = $("#rental-form");
  let start = fromLocalInput(f.start.value);
  if (!start) { start = roundedNow(); f.start.value = toLocalInput(start); }
  f.end.value = toLocalInput(new Date(start.getTime() + hours * 3600 * 1000));
  loadAvailableBikes();
}

function roundedNow() {
  const d = new Date();
  d.setSeconds(0, 0);
  return d;
}

$("#btn-now").addEventListener("click", () => {
  const f = $("#rental-form");
  const oldStart = fromLocalInput(f.start.value), oldEnd = fromLocalInput(f.end.value);
  const start = roundedNow();
  f.start.value = toLocalInput(start);
  // Dauer beibehalten, wenn schon eine gewählt war
  const dur = oldStart && oldEnd && oldEnd > oldStart ? oldEnd - oldStart : 2 * 3600 * 1000;
  f.end.value = toLocalInput(new Date(start.getTime() + dur));
  loadAvailableBikes();
});
$$("#duration-chips button").forEach((b) => b.addEventListener("click", () => setDuration(Number(b.dataset.hours))));
$("#rental-form").start.addEventListener("change", loadAvailableBikes);
$("#rental-form").end.addEventListener("change", loadAvailableBikes);
$("#rental-form").bikeId.addEventListener("change", updateQuote);

$("#rental-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const f = e.target;
  const p = period();
  if (!p) return toast("Ende muss nach dem Beginn liegen", true);
  try {
    await api("POST", "/rentals", { personId: Number(f.personId.value), bikeId: Number(f.bikeId.value), ...p });
    toast(new Date(p.start) > new Date() ? "Reservierung gespeichert" : "Ausleihe gebucht");
    f.bikeId.value = "";
    refresh();
  } catch (err) { toast(err.message, true); }
});

function renderRentals() {
  const q = $("#rental-search").value.toLowerCase();
  const st = $("#rental-filter").value;
  const order = { "überfällig": 0, aktiv: 1, reserviert: 2, "zurückgegeben": 3 };
  const rows = state.rentals
    .filter((r) => (!st || r.status === st) && `${r.personName} ${r.bikeLabel} ${r.document}`.toLowerCase().includes(q))
    .sort((a, b) => order[a.status] - order[b.status] || new Date(a.start) - new Date(b.start));

  $("#rental-table").innerHTML = rows.map((r) => {
    const price = r.finalCents != null
      ? `${euro(r.finalCents)}<span class="sub">geplant ${euro(r.plannedCents)}</span>`
      : `${euro(r.plannedCents)}<span class="sub">geplant</span>`;
    let action = "";
    if (r.status === "aktiv" || r.status === "überfällig") action = `<button class="small" data-return="${r.id}">Rückgabe</button>`;
    if (r.status === "reserviert") action = `<button class="small danger" data-cancel="${r.id}">Stornieren</button>`;
    const period = `${fmtDate(r.start)}<span class="sub">bis ${fmtDate(r.end)}${r.returnedAt ? " · zurück " + fmtDate(r.returnedAt) : ""}</span>`;
    return `<tr>
      <td><strong>${esc(r.personName)}</strong><span class="sub">${esc(r.document)}</span></td>
      <td>${esc(r.bikeLabel)}</td>
      <td class="num">${period}</td>
      <td class="num">${price}</td>
      <td><span class="badge ${r.status}">${r.status}</span></td>
      <td>${action}</td>
    </tr>`;
  }).join("") || `<tr><td colspan="6" class="hint">Keine Ausleihen.</td></tr>`;
}
$("#rental-search").addEventListener("input", renderRentals);
$("#rental-filter").addEventListener("change", renderRentals);

// ---------- Buttons in Listen (Event-Delegation) ----------
document.addEventListener("click", async (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  try {
    if (b.dataset.return) {
      const r = await api("POST", `/rentals/${b.dataset.return}/return`);
      toast(`Rückgabe gebucht – zu zahlen: ${euro(r.finalCents)}`);
    } else if (b.dataset.cancel) {
      if (!confirm("Reservierung wirklich stornieren?")) return;
      await api("DELETE", `/rentals/${b.dataset.cancel}`);
      toast("Reservierung storniert");
    } else if (b.dataset.delBike) {
      if (!confirm("Fahrrad wirklich löschen?")) return;
      await api("DELETE", `/bikes/${b.dataset.delBike}`);
      toast("Fahrrad gelöscht");
    } else if (b.dataset.delPerson) {
      if (!confirm("Person wirklich löschen?")) return;
      await api("DELETE", `/persons/${b.dataset.delPerson}`);
      toast("Person gelöscht");
    } else if (b.dataset.upload) {
      uploadTarget = Number(b.dataset.upload);
      $("#hidden-upload").click();
      return;
    } else {
      return;
    }
    refresh();
  } catch (err) { toast(err.message, true); }
});

// ---------- Start ----------
(async function init() {
  try {
    const t = await api("GET", "/tariffs");
    $("#bike-form").type.innerHTML = t.bikeTypes.map((s) => `<option>${esc(s.type)}</option>`).join("");
    $("#bike-form").frameSize.innerHTML = t.frameSizes.map((s) => `<option${s === "M" ? " selected" : ""}>${esc(s)}</option>`).join("");
    $("#person-form").docType.innerHTML = t.documentTypes.map((d) => `<option>${esc(d)}</option>`).join("");
    $("#tariff-table").innerHTML = t.bikeTypes.map((s) =>
      `<tr><td>${esc(s.type)}</td><td class="num">${euro(s.tariff.hourlyCents)}</td><td class="num">${euro(s.tariff.dailyCapCents)}</td></tr>`).join("");
  } catch (err) { toast(err.message, true); }
  // Standard: jetzt + 2 h
  const f = $("#rental-form");
  const start = roundedNow();
  f.start.value = toLocalInput(start);
  f.end.value = toLocalInput(new Date(start.getTime() + 2 * 3600 * 1000));
  refresh();
  setInterval(refresh, 60 * 1000); // Status (überfällig etc.) aktuell halten
})();
