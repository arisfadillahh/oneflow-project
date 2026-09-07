import { useEffect, useMemo, useRef, useState } from "react";
import { DetailRow, EmptyState, Notice, SectionCard, StatCard, StatusPill } from "../ui";
import {
  apiBase,
  wsBase,
  waBase,
  roleNav,
  viewMeta,
  fallbackOpenAIModels,
  analyticsRangeOptions,
  composerEmojis,
  normalizeModelOptions,
  defaultSettingsDraft,
  simulationScenarios,
  createPlaygroundWelcome,
  buildPlaygroundHistory,
  Icons,
  formatDateTime,
  formatTime,
  formatNumber,
  formatPercent,
  formatCurrencyIDR,
  formatCostIDR,
  truncateText,
  fileToBase64,
  mediaKindFromMime,
  isEditingFormControl,
  toLabel,
  toRoleLabel,
  roleInitials,
  renderChatMessage,
  renderMessageMedia,
  renderChatBlock,
  renderInlineRichText,
  getInitials,
  getAvatarColor,
  settingsApiToDraft,
  dashboardBadge,
  activeViewStorageKey,
  waModalDismissedStorageKey,
  viewRouteSegments,
  segmentViewIds,
  canonicalViewId,
  segmentForView,
  pathForView,
  viewFromPath,
  requestedDashboardPath,
  defaultViewForRole,
  isViewAllowedForRole,
  storedViewForRole
} from "../../../lib/dashboard-core";

function ToggleRow({ label, helper, checked, onChange }) {
  return (
    <label className="settings-toggle-row">
      <span>
        <strong>{label}</strong>
        <small>{helper}</small>
      </span>
      <span className="toggle">
        <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
        <span className="toggle-slider" />
      </span>
    </label>
  );
}

export function AISettingsView({ settingsForm, setSettingsForm, settingsDraft, setSettingsDraft, saveSettings, busyKey }) {
  const form = settingsForm ?? settingsDraft ?? defaultSettingsDraft;
  const setForm = setSettingsForm ?? setSettingsDraft ?? (() => {});
  const updateField = (key, value) => setForm({ ...form, [key]: value });
  const safetyToggleItems = [
    ["answerOnlyFromKnowledge", "Jawab hanya dari Knowledge", "AI hanya menjawab saat sumber resmi cukup."],
    ["dontBroadenTopic", "Jaga topik bisnis", "AI diarahkan balik kalau user melebar dari konteks bisnis."],
    ["forbidPromises", "Larang janji dan keputusan final", "AI tidak menjanjikan refund, approval, diskon final, stok, atau jadwal pasti tanpa data."],
    ["forbidSensitiveAnswers", "Lindungi data sensitif", "AI menahan jawaban untuk data pribadi, internal, pembayaran, atau akses akun."],
    ["requireActionConfirmation", "Minta konfirmasi sebelum aksi penting", "AI harus memastikan user setuju sebelum booking, order, cancel, refund, atau ubah data."],
    ["escalateLowConfidence", "Eskalasikan kalau confidence rendah", "Kalau knowledge kurang kuat atau risiko salah tinggi, AI panggil human."],
  ];
  const conversationToggleItems = [
    ["guideNextStep", "Arahkan ke langkah berikutnya", "AI boleh menutup dengan pertanyaan atau ajakan lanjut yang relevan."],
    ["allowClarification", "Boleh tanya klarifikasi", "AI boleh tanya balik saat informasi customer belum cukup."],
    ["conciseResponse", "Jawab ringkas", "AI diminta padat dan tidak bertele-tele."],
  ];
  const contactToggleItems = [
    ["allowAutoUpdateContactName", "Auto-update nama kontak", "Nama kontak boleh diperbarui dari percakapan."],
    ["onlyFillNameIfEmpty", "Isi nama hanya jika kosong", "Nama lama tidak ditimpa bila sudah ada."],
  ];

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Aturan AI</div>
          <div className="page-desc">Atur batas jawaban AI dan kapan diteruskan ke admin.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" type="button" onClick={() => setForm(defaultSettingsDraft)}>Reset Default</button>
          <button className="btn btn-primary" onClick={saveSettings} disabled={busyKey === "settings"}>Simpan Perubahan</button>
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 340px", gap: "20px" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          <div className="card">
            <div className="card-title mb-16">Prompt Utama (System Prompt)</div>
            <div className="alert blue mb-16">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
              <span>Prompt ini digunakan AI sebagai instruksi utama saat menjawab pelanggan via WhatsApp.</span>
            </div>
            <textarea 
              className="form-textarea" 
              rows="8" 
              style={{ fontFamily: "monospace", fontSize: "13px" }}
              value={form.systemPrompt}
              onChange={(e) => updateField("systemPrompt", e.target.value)}
              placeholder="System prompt..."
            />
          </div>

          <div className="card">
            <div className="card-title mb-16">Kapan AI panggil admin</div>
            <textarea 
              className="form-textarea" 
              rows="5" 
              style={{ fontFamily: "monospace", fontSize: "13px" }}
              value={form.escalationPrompt}
              onChange={(e) => updateField("escalationPrompt", e.target.value)}
              placeholder="Contoh: refund, komplain pembayaran, perubahan order, atau data belum pasti."
            />
          </div>

          <div className="card">
            <div className="card-title mb-16">Pesan Tunggu (Hold Message)</div>
            <textarea 
              className="form-textarea" 
              rows="3" 
              style={{ fontFamily: "monospace", fontSize: "13px" }}
              value={form.fallbackWaitingMessage}
              onChange={(e) => updateField("fallbackWaitingMessage", e.target.value)}
              placeholder="Pesan tunggu eskalasi..."
            />
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          <div className="card">
            <div className="card-title mb-16">Batas aman jawaban</div>
            <div className="settings-toggle-list">
              {safetyToggleItems.map(([key, label, helper]) => (
                <ToggleRow key={key} label={label} helper={helper} checked={form[key] !== false} onChange={(value) => updateField(key, value)} />
              ))}
            </div>
          </div>

          <div className="card">
            <div className="card-title mb-16">Gaya percakapan</div>
            <div className="settings-toggle-list">
              {conversationToggleItems.map(([key, label, helper]) => (
                <ToggleRow key={key} label={label} helper={helper} checked={form[key] !== false} onChange={(value) => updateField(key, value)} />
              ))}
            </div>
          </div>

          <div className="card">
            <div className="card-title mb-16">Data Kontak</div>
            <div className="settings-toggle-list">
              {contactToggleItems.map(([key, label, helper]) => (
                <ToggleRow key={key} label={label} helper={helper} checked={key === "allowAutoUpdateContactName" ? Boolean(form[key]) : form[key] !== false} onChange={(value) => updateField(key, value)} />
              ))}
            </div>
          </div>

          <div className="card">
            <div className="card-title mb-16">Kontrol Klarifikasi</div>
            <div className="mb-16">
              <label className="form-label text-sm">Maksimum klarifikasi</label>
              <select className="form-select" value={form.maxClarificationCount} onChange={(e) => updateField("maxClarificationCount", Number(e.target.value))} disabled={form.allowClarification === false}>
                <option value="0">0 kali (langsung eskalasi)</option>
                <option value="1">1 kali</option>
                <option value="2">2 kali</option>
                <option value="3">3 kali</option>
              </select>
            </div>
            <div className="text-xs text-muted">Dipakai hanya saat "Boleh tanya klarifikasi" aktif.</div>
          </div>

          <div className="card">
            <div className="card-title mb-16">Ringkasan</div>
            {[
              ["Safety", safetyToggleItems.filter(([key]) => form[key] !== false).length, safetyToggleItems.length],
              ["Conversation", conversationToggleItems.filter(([key]) => form[key] !== false).length, conversationToggleItems.length],
              ["Kontak", contactToggleItems.filter(([key]) => key === "allowAutoUpdateContactName" ? Boolean(form[key]) : form[key] !== false).length, contactToggleItems.length],
            ].map(([label, active, total]) => (
              <div className="flex-between mb-16" key={label}>
                <span className="text-sm">{label}</span>
                <span className="badge blue">{active}/{total} aktif</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
