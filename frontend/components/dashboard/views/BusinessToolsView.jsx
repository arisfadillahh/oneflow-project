import { useEffect, useMemo, useState } from "react";
import { Icons } from "../../../lib/dashboard-core";

const toolOrder = ["prospects", "tickets", "commerce", "booking", "payments"];

function orderedTools(items = []) {
  const byKey = new Map(items.map((item) => [item.key, item]));
  return toolOrder.map((key) => byKey.get(key)).filter(Boolean);
}

function isInstalled(item) {
  return Boolean(item?.installed ?? item?.enabled);
}

function toolStatusLabel(item) {
  if (item.status === "planned") return "Disiapkan";
  return isInstalled(item) ? "Terpasang" : "Belum dipasang";
}

function toolBadgeClass(item) {
  if (item.status === "planned") return "orange";
  return isInstalled(item) ? "green" : "gray";
}

function toolAiSummary(item, installed) {
  if (item.status === "planned") return "AI belum memakai alat ini.";
  if (!installed) return "Pasang dulu sebelum AI bisa memakai data alat ini.";
  if ((item.aiMode || "off") === "off") return "AI belum diizinkan memakai alat ini.";
  if (item.aiMode === "read") return "AI boleh membaca data alat ini saat chat.";
  if (item.aiMode === "draft") return "AI boleh membaca data dan menyiapkan catatan untuk dicek admin.";
  if (item.aiMode === "action") return "AI boleh membaca data dan membantu proses operasional yang diizinkan.";
  return "AI mengikuti izin akses yang dipilih.";
}

function aiModeLabel(mode) {
  if (mode === "read") return "Baca data";
  if (mode === "draft") return "Siapkan draf";
  if (mode === "action") return "Bantu otomatis";
  return "Belum aktif";
}

function fallbackAiModes(item) {
  return item.aiModes?.length ? item.aiModes : [
    { mode: "off", label: "Belum dipakai AI", description: "AI tidak memakai data alat ini." },
    { mode: "read", label: "Jawab dari data", description: "AI hanya membaca data alat ini." },
  ];
}

function friendlyAiModeOptionLabel(option) {
  if (option.mode === "off") return "Belum dipakai AI";
  if (option.mode === "read") return "Jawab dari data";
  if (option.mode === "draft") return "Siapkan draf untuk admin";
  if (option.mode === "action") return "Bantu otomatis";
  return option.label;
}

function toolIcon(item) {
  return Icons[item.key === "booking" ? "calendar" : item.key === "commerce" ? "products" : item.key === "prospects" ? "analytics" : item.key === "tickets" ? "ticket" : "wallet"];
}

function aiBadgeClass(item) {
  if (!isInstalled(item) || (item.aiMode || "off") === "off") return "gray";
  if (item.aiMode === "action") return "orange";
  return "blue";
}

function notificationRuleDraft(rule) {
  return {
    triggerKey: rule?.triggerKey || "",
    isEnabled: rule?.isEnabled !== false,
    templateText: rule?.templateText || "",
  };
}

function notificationRulesForTool(rules = [], toolKey = "") {
  return rules.filter((rule) => rule?.pluginKey === toolKey);
}

function variableAllowedForTool(variable, toolKey) {
  const group = variable?.group || "";
  if (["Bisnis", "Customer", "Event", "Percakapan"].includes(group)) return true;
  if (toolKey === "commerce") return group === "Pesanan" || group === "Plugin";
  if (toolKey === "booking") return group === "Booking" || group === "Plugin";
  return group === "Plugin";
}

function detailFlowItems(item) {
  return item.activationImpact?.length ? item.activationImpact : [
    "Admin memasang alat dan mengisi data bisnisnya.",
    "AI baru memakai data jika izinnya dinyalakan.",
    "Aksi edit dan koreksi tetap tersedia dari dashboard admin.",
  ];
}

function aiModeDescription(item) {
  const mode = item.aiMode || "off";
  return fallbackAiModes(item).find((entry) => entry.mode === mode)?.description || "Pilih izin AI untuk alat ini.";
}

function aiModePlainImpact(item) {
  const mode = item.aiMode || "off";
  if (!isInstalled(item)) return "Pasang alat dulu. Setelah itu pilih apakah AI boleh membaca data, menyiapkan draf, atau membantu otomatis.";
  if (mode === "off") return "AI belum menyentuh data alat ini. Pelanggan tetap dijawab dari knowledge umum.";
  if (mode === "read") return "AI hanya menjawab info dari data ini. Tidak ada pesanan, prospek, atau booking baru yang dibuat.";
  if (mode === "draft") return "AI boleh menyiapkan draf. Admin tetap mengecek sebelum data dianggap final.";
  if (mode === "action") return "AI boleh membuat data operasional saat chat sudah jelas. Admin tetap bisa mengubah, konfirmasi, atau membatalkan dari dashboard.";
  return "AI mengikuti izin yang dipilih.";
}

export function BusinessToolsView({
  businessTools = [],
  loadState = "loading",
  canManage = false,
  busyKey = "",
  onInstallTool,
  onRemoveTool,
  onUpdateToolAI,
  onRefresh,
  onOpenTool,
  groupNotificationRules = [],
  groupNotificationVariables = [],
  canManageGroupNotifications = false,
  isNotificationGroupBound = false,
  notificationGroupName = "",
  saveGroupNotificationRule,
  sendGroupNotificationTest,
  onOpenNotificationGroup,
}) {
  const tools = useMemo(() => orderedTools(businessTools), [businessTools]);
  const [detailKey, setDetailKey] = useState("");
  const [editingNotificationKey, setEditingNotificationKey] = useState("");
  const [notificationDrafts, setNotificationDrafts] = useState({});
  const detailItem = tools.find((item) => item.key === detailKey);
  const detailNotificationRules = useMemo(
    () => notificationRulesForTool(groupNotificationRules, detailItem?.key),
    [detailItem?.key, groupNotificationRules]
  );
  const installedCount = tools.filter(isInstalled).length;
  const aiActiveCount = tools.filter((item) => isInstalled(item) && (item.aiMode || "off") !== "off").length;
  const plannedCount = tools.filter((item) => item.status === "planned").length;

  useEffect(() => {
    if (!detailItem) return undefined;
    function onKeyDown(event) {
      if (event.key === "Escape") closeDetail();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [detailItem]);

  useEffect(() => {
    setNotificationDrafts((current) => {
      const next = { ...current };
      groupNotificationRules.forEach((rule) => {
        if (!rule?.triggerKey || next[rule.triggerKey]) return;
        next[rule.triggerKey] = notificationRuleDraft(rule);
      });
      return next;
    });
  }, [groupNotificationRules]);

  function closeDetail() {
    setDetailKey("");
    setEditingNotificationKey("");
  }

  function installFromDetail(key) {
    onInstallTool?.(key);
    closeDetail();
  }

  function openToolFromDetail(viewId) {
    closeDetail();
    if (viewId) onOpenTool?.(viewId);
  }

  function removeFromDetail(key) {
    closeDetail();
    onRemoveTool?.(key);
  }

  function draftForRule(rule) {
    return notificationDrafts[rule.triggerKey] || notificationRuleDraft(rule);
  }

  function updateNotificationDraft(triggerKey, patch) {
    setNotificationDrafts((current) => ({
      ...current,
      [triggerKey]: {
        ...notificationRuleDraft(groupNotificationRules.find((rule) => rule?.triggerKey === triggerKey)),
        ...(current[triggerKey] || {}),
        ...patch,
      },
    }));
  }

  function insertNotificationVariable(triggerKey, variableKey) {
    const token = `{{${variableKey}}}`;
    const rule = groupNotificationRules.find((item) => item?.triggerKey === triggerKey) || { triggerKey };
    const current = draftForRule(rule);
    const separator = current.templateText?.endsWith(" ") || !current.templateText ? "" : " ";
    updateNotificationDraft(triggerKey, { templateText: `${current.templateText || ""}${separator}${token}` });
  }

  async function saveNotificationDraft(rule) {
    const draft = draftForRule(rule);
    await saveGroupNotificationRule?.(draft);
    setEditingNotificationKey("");
  }

  return (
    <div className="business-tools-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Alat Bisnis</div>
          <div className="page-desc">Tambahkan kemampuan AI: produk, pesanan, dan booking.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary btn-sm" type="button" onClick={onRefresh}>
            {Icons.refresh} Muat ulang
          </button>
        </div>
      </div>

      <div className="business-plugin-summary" aria-label="Ringkasan alat bisnis">
        <div>
          <strong>{installedCount}</strong>
          <span>Terpasang</span>
        </div>
        <div>
          <strong>{aiActiveCount}</strong>
          <span>Dipakai AI</span>
        </div>
        <div>
          <strong>{plannedCount}</strong>
          <span>Disiapkan</span>
        </div>
      </div>

      <section className="business-tools-marketplace" aria-label="Daftar alat bisnis">
        {loadState === "loading" && !tools.length ? (
          <div className="business-muted-panel business-tools-empty">Memuat daftar alat bisnis...</div>
        ) : null}
        {loadState === "error" && !tools.length ? (
          <div className="business-muted-panel business-tools-empty">Daftar alat bisnis belum tersedia. Muat ulang halaman atau coba lagi setelah koneksi backend pulih.</div>
        ) : null}
        {loadState === "loaded" && !tools.length ? (
          <div className="business-muted-panel business-tools-empty">Belum ada alat bisnis yang tersedia untuk organisasi ini.</div>
        ) : null}
        {tools.map((item) => {
          const installed = isInstalled(item);
          const installDisabled = !canManage || item.status !== "ready" || installed || busyKey === `business-tool-${item.key}`;
          const removeDisabled = !canManage || busyKey === `business-tool-${item.key}`;
          const toolNotificationRules = notificationRulesForTool(groupNotificationRules, item.key);
          const activeNotificationCount = toolNotificationRules.filter((rule) => rule?.isEnabled !== false).length;
          return (
            <article className={`business-tool-card ${installed ? "active" : ""}`} key={item.key}>
              <div className="business-tool-icon">{toolIcon(item)}</div>
              <div className="business-tool-card-body">
                <div className="business-tool-card-head">
                  <div>
                    <strong>{item.name}</strong>
                    <span>{item.category}</span>
                  </div>
                  <div className="business-tool-card-badges">
                    <span className={`badge ${toolBadgeClass(item)}`}>{toolStatusLabel(item)}</span>
                    <span className={`badge ${aiBadgeClass(item)}`}>{aiModeLabel(item.aiMode)}</span>
                  </div>
                </div>
                <p>{item.description}</p>
                <div className="business-tool-ai-note">
                  {Icons.robot}
                  <span>{toolAiSummary(item, installed)}</span>
                </div>
                {installed && toolNotificationRules.length ? (
                  <div className="business-tool-notification-note">
                    {Icons.whatsapp}
                    <span>{toolNotificationRules.length} jenis notif grup tersedia, {activeNotificationCount} aktif.</span>
                  </div>
                ) : null}
              </div>
              <div className="business-tool-actions">
                <button className="btn btn-secondary btn-sm" type="button" onClick={() => setDetailKey(item.key)}>
                  {installed ? "Atur" : "Detail"}
                </button>
                {installed && item.installedView ? (
                  <button className="btn btn-secondary btn-sm" type="button" onClick={() => onOpenTool?.(item.installedView)}>
                    {Icons.external} Buka
                  </button>
                ) : null}
                {installed ? (
                  <button className="btn btn-danger btn-sm" type="button" disabled={removeDisabled} onClick={() => onRemoveTool?.(item.key)}>
                    {busyKey === `business-tool-${item.key}` ? "Melepas..." : "Lepas"}
                  </button>
                ) : (
                  <button className="btn btn-primary btn-sm" type="button" disabled={installDisabled} onClick={() => onInstallTool?.(item.key)}>
                    {item.status === "planned" ? "Belum tersedia" : busyKey === `business-tool-${item.key}` ? "Memasang..." : "Pasang"}
                  </button>
                )}
              </div>
            </article>
          );
        })}
      </section>

      {detailItem ? (
        <div className="business-tool-modal-backdrop" role="presentation" onMouseDown={closeDetail}>
          <section
            className="business-tool-detail-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="business-tool-detail-title"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <div className="business-tool-detail-head">
              <div>
                <span>{detailItem.category}</span>
                <h3 id="business-tool-detail-title">{detailItem.name}</h3>
                <p>{detailItem.description}</p>
              </div>
              <div className="business-tool-detail-head-actions">
                <span className={`badge ${toolBadgeClass(detailItem)}`}>{toolStatusLabel(detailItem)}</span>
                <span className={`badge ${aiBadgeClass(detailItem)}`}>{aiModeLabel(detailItem.aiMode)}</span>
                <button className="icon-btn" type="button" aria-label="Tutup detail alat" onClick={closeDetail}>
                  {Icons.close}
                </button>
              </div>
            </div>

            <div className="business-tool-detail-grid">
              <div className="business-tool-detail-flow">
                <div className="business-tool-detail-title">Yang terjadi saat dipasang</div>
                <ul className="business-detail-list">
                  {detailFlowItems(detailItem).map((item) => (
                    <li key={item}>
                      {Icons.check}
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
              <div className="business-tool-ai-controls">
                <div>
                  <div className="business-tool-detail-title">Izin AI memakai alat</div>
                  <p>{aiModeDescription(detailItem)}</p>
                  <p>{aiModePlainImpact(detailItem)}</p>
                  {detailItem.aiWarning ? <small>{detailItem.aiWarning}</small> : null}
                </div>
                <select
                  className="form-select"
                  value={detailItem.aiMode || "off"}
                  disabled={!canManage || !isInstalled(detailItem) || busyKey === `business-tool-ai-${detailItem.key}`}
                  onChange={(event) => onUpdateToolAI?.(detailItem.key, event.target.value)}
                >
                  {fallbackAiModes(detailItem).map((mode) => (
                    <option key={mode.mode} value={mode.mode}>{friendlyAiModeOptionLabel(mode)}</option>
                  ))}
                </select>
              </div>
              <div>
                <div className="business-tool-detail-title">Fitur</div>
                <ul className="business-detail-list">
                  {(detailItem.features || []).map((feature) => (
                    <li key={feature}>
                      {Icons.check}
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <div className="business-tool-detail-title">Bisa disesuaikan</div>
                <ul className="business-detail-list">
                  {(detailItem.customizable || []).map((item) => (
                    <li key={item}>
                      {Icons.settings}
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>

            {isInstalled(detailItem) && detailNotificationRules.length ? (
              <div className="business-tool-notification-panel">
                <div className="business-tool-notification-head">
                  <div>
                    <div className="business-tool-detail-title">Notifikasi grup {detailItem.navLabel || detailItem.name}</div>
                    <p>
                      Pilih event operasional yang perlu masuk ke grup WhatsApp tim. Format pesan bisa beda untuk setiap fungsi alat.
                    </p>
                  </div>
                  <span className={`badge ${isNotificationGroupBound ? "green" : "gray"}`}>
                    {isNotificationGroupBound ? `Grup: ${notificationGroupName || "terhubung"}` : "Grup belum terhubung"}
                  </span>
                </div>
                {!isNotificationGroupBound ? (
                  <div className="business-tool-notification-warning">
                    <span>Hubungkan grup di menu WhatsApp dulu supaya test dan notifikasi otomatis bisa dikirim.</span>
                    {onOpenNotificationGroup ? (
                      <button className="btn btn-secondary btn-sm" type="button" onClick={onOpenNotificationGroup}>
                        Buka WhatsApp
                      </button>
                    ) : null}
                  </div>
                ) : null}
                <div className="wa-plugin-rule-list business-tool-notification-list">
                  {detailNotificationRules.map((rule) => {
                    const draft = draftForRule(rule);
                    const isEditing = editingNotificationKey === rule.triggerKey;
                    const isDirty = JSON.stringify(draft) !== JSON.stringify(notificationRuleDraft(rule));
                    const variables = groupNotificationVariables.filter((variable) => variableAllowedForTool(variable, detailItem.key));
                    return (
                      <div key={rule.triggerKey} className={`wa-plugin-rule business-tool-notification-rule ${draft.isEnabled ? "enabled" : ""}`}>
                        <div className="wa-plugin-rule-main">
                          <div>
                            <strong>{rule.label}</strong>
                            <span>{rule.description}</span>
                          </div>
                          <label className="switch-row compact">
                            <input
                              type="checkbox"
                              checked={draft.isEnabled}
                              onChange={(event) => updateNotificationDraft(rule.triggerKey, { isEnabled: event.target.checked })}
                              disabled={!canManageGroupNotifications || busyKey === "group-notification-rule-save"}
                            />
                            <span>{draft.isEnabled ? "Aktif" : "Mati"}</span>
                          </label>
                        </div>
                        <div className="wa-plugin-rule-actions">
                          <button className="btn btn-secondary btn-sm" type="button" onClick={() => setEditingNotificationKey(isEditing ? "" : rule.triggerKey)}>
                            {isEditing ? "Tutup" : "Edit pesan"}
                          </button>
                          <button
                            className="btn btn-secondary btn-sm"
                            type="button"
                            onClick={() => sendGroupNotificationTest?.(rule.triggerKey)}
                            disabled={!isNotificationGroupBound || !draft.isEnabled || isDirty || busyKey === "group-notification-test"}
                            title={isDirty ? "Simpan perubahan dulu" : undefined}
                          >
                            {busyKey === "group-notification-test" ? "Mengirim..." : "Kirim test"}
                          </button>
                          <button
                            className="btn btn-primary btn-sm"
                            type="button"
                            onClick={() => saveNotificationDraft(rule)}
                            disabled={!canManageGroupNotifications || !isDirty || !String(draft.templateText || "").trim() || busyKey === "group-notification-rule-save"}
                          >
                            {busyKey === "group-notification-rule-save" ? "Menyimpan..." : "Simpan"}
                          </button>
                        </div>
                        {isEditing ? (
                          <div className="wa-plugin-rule-editor">
                            <textarea
                              className="form-textarea"
                              rows="6"
                              value={draft.templateText}
                              onChange={(event) => updateNotificationDraft(rule.triggerKey, { templateText: event.target.value })}
                              disabled={!canManageGroupNotifications || busyKey === "group-notification-rule-save"}
                            />
                            <div className="wa-plugin-variable-list">
                              {variables.map((variable) => (
                                <button key={variable.key} type="button" className="agent-variable-chip" onClick={() => insertNotificationVariable(rule.triggerKey, variable.key)} disabled={!canManageGroupNotifications || busyKey === "group-notification-rule-save"} title={variable.example || variable.label}>
                                  {variable.label}
                                </button>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}

            <div className="business-tool-detail-actions">
              <button className="btn btn-secondary btn-sm" type="button" onClick={closeDetail}>
                Tutup
              </button>
              {isInstalled(detailItem) && detailItem.installedView ? (
                <>
                  <button
                    className="btn btn-danger btn-sm"
                    type="button"
                    disabled={!canManage || busyKey === `business-tool-${detailItem.key}`}
                    onClick={() => removeFromDetail(detailItem.key)}
                  >
                    {busyKey === `business-tool-${detailItem.key}` ? "Melepas..." : "Lepas alat"}
                  </button>
                  <button className="btn btn-primary btn-sm" type="button" onClick={() => openToolFromDetail(detailItem.installedView)}>
                    Buka {detailItem.navLabel || detailItem.name}
                  </button>
                </>
              ) : (
                <button
                  className="btn btn-primary btn-sm"
                  type="button"
                  disabled={!canManage || detailItem.status !== "ready" || busyKey === `business-tool-${detailItem.key}`}
                  onClick={() => installFromDetail(detailItem.key)}
                >
                  {detailItem.status === "planned" ? "Belum tersedia" : busyKey === `business-tool-${detailItem.key}` ? "Memasang..." : "Pasang alat"}
                </button>
              )}
            </div>
          </section>
        </div>
      ) : null}
    </div>
  );
}
