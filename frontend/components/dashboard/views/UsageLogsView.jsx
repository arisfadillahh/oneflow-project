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

export function UsageLogsView({ role, usageLogs, billingAnalytics, inspectUsageLog, busyKey }) {
  const ownerMode = role === "owner";
  const displayedUsage = useMemo(() => {
    return usageLogs.map((log) => {
      const inputTokens = Number(log.inputTokens) || 0;
      const outputTokens = Number(log.outputTokens) || 0;
      const embeddingTokens = Number(log.embeddingTokens) || 0;
      const totalTokens = Number(log.totalTokens) || inputTokens + outputTokens + embeddingTokens;
      const runId = log.aiRunId || log.id || "";
      const contactLabel = ownerMode
        ? (log.organizationName || "Unknown company")
        : log.usageType === "playground"
        ? "AI Playground"
        : (log.contactName || log.contactPhone || (log.conversationId ? `Conversation #${String(log.conversationId).slice(0, 8)}` : "-"));

      return {
        ...log,
        modelLabel: log.modelName || log.modelUsed || "-",
        runId,
        contactLabel,
        organizationName: log.organizationName || "",
        totalTokens,
        latencyMs: Number(log.latencyMs) || 0,
        creditsUsed: Number(log.creditsUsed) || 0,
        costIdr: Number(log.costIdr) || 0,
        statusLabel: log.status === "escalated" ? "Escalated" : (log.error || log.status === "failed" ? "Failed" : "Success"),
        statusClass: log.status === "escalated" ? "orange" : (log.error || log.status === "failed" ? "red" : "green"),
      };
    });
  }, [usageLogs, ownerMode]);

  const pageTotals = useMemo(() => {
    const fallbackCost = displayedUsage.reduce((acc, log) => acc + log.costIdr, 0);
    return {
      rows: displayedUsage.length,
      tokens: displayedUsage.reduce((acc, log) => acc + log.totalTokens, 0),
      credits: displayedUsage.reduce((acc, log) => acc + log.creditsUsed, 0),
      costIdr: billingAnalytics?.totalCostIdr ?? fallbackCost,
      avgLatency: Math.round(displayedUsage.reduce((acc, log) => acc + log.latencyMs, 0) / Math.max(1, displayedUsage.filter((log) => log.latencyMs > 0).length)),
    };
  }, [displayedUsage, billingAnalytics]);
  const topOrganizations = billingAnalytics?.byOrganization ?? [];
  const topModels = billingAnalytics?.byModel ?? [];
  const daily = billingAnalytics?.daily ?? [];
  const dailyPeak = Math.max(1, ...daily.map((item) => Number(item.creditsUsed) || 0));

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">{ownerMode ? "Laporan Pemakaian" : "Log Pemakaian"}</div>
          <div className="page-desc">{ownerMode ? "Analitik agregat pemakaian AI lintas company. Data kontak dan isi percakapan tetap disembunyikan." : "Breakdown pemakaian kredit dan jejak per AI run."}</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary">Export CSV</button>
        </div>
      </div>

      <div className="kpi-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))", marginBottom: "24px" }}>
        <div className="kpi-card">
          <div className="kpi-icon purple"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m12 3-1.9 5.1L5 10l5.1 1.9L12 17l1.9-5.1L19 10l-5.1-1.9L12 3Z"/></svg></div>
          <div className="kpi-value">{formatNumber(pageTotals.rows)}</div><div className="kpi-label">{ownerMode ? "AI Runs Sample" : "Total Logs Displayed"}</div><div className="kpi-trend neutral">{ownerMode ? "Privacy-safe" : "DB"}</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></div>
          <div className="kpi-value">{formatNumber(pageTotals.tokens)}</div><div className="kpi-label">Total Tokens</div><div className="kpi-trend neutral">On Page</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon orange"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg></div>
          <div className="kpi-value">{formatNumber(pageTotals.credits)}</div><div className="kpi-label">Kredit Terpakai</div><div className="kpi-trend neutral">On Page</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon green"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></div>
          <div className="kpi-value">{pageTotals.avgLatency}ms</div><div className="kpi-label">Avg Latency</div><div className="kpi-trend neutral">{formatCurrencyIDR(pageTotals.costIdr)}</div>
        </div>
      </div>

      {ownerMode && (
        <div className="wallet-layout" style={{ marginBottom: "24px" }}>
          <main className="wallet-main">
            <div className="wallet-panel wallet-table-panel">
              <div className="wallet-panel-header">
                <div>
                  <div className="wallet-panel-title">Company Usage</div>
                  <div className="wallet-panel-subtitle">Agregat kredit dan biaya per company tanpa data kontak pelanggan.</div>
                </div>
              </div>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Company</th>
                      <th style={{ textAlign: "right" }}>AI Runs</th>
                      <th style={{ textAlign: "right" }}>Credits</th>
                      <th style={{ textAlign: "right" }}>Cost</th>
                    </tr>
                  </thead>
                  <tbody>
                    {topOrganizations.map((item) => (
                      <tr key={item.label}>
                        <td><strong>{item.label}</strong></td>
                        <td style={{ textAlign: "right" }}>{formatNumber(item.count)}</td>
                        <td style={{ textAlign: "right" }}>{formatNumber(item.creditsUsed)}</td>
                        <td style={{ textAlign: "right" }}>{formatCurrencyIDR(item.costIdr)}</td>
                      </tr>
                    ))}
                    {!topOrganizations.length && (
                      <tr><td colSpan={4} className="wallet-empty-cell">Belum ada usage agregat.</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </main>
          <aside className="wallet-side">
            <div className="wallet-panel">
              <div className="wallet-panel-title">Model Mix</div>
              <div className="wallet-panel-subtitle">Distribusi pemakaian berdasarkan model AI.</div>
              <div className="wallet-approval-list">
                {topModels.slice(0, 5).map((item) => (
                  <div className="wallet-approval-item" key={item.label}>
                    <div>
                      <strong>{item.label}</strong>
                      <span>{formatNumber(item.creditsUsed)} credits - {formatNumber(item.count)} runs</span>
                    </div>
                    <span className="badge gray">{formatCurrencyIDR(item.costIdr)}</span>
                  </div>
                ))}
                {!topModels.length && <div className="text-xs text-muted">Belum ada data model.</div>}
              </div>
            </div>
            <div className="wallet-panel">
              <div className="wallet-panel-title">14-Day Credit Trend</div>
              <div className="wallet-panel-subtitle">Tren harian agregat seluruh company.</div>
              <div className="wallet-approval-list">
                {daily.slice(-7).map((item) => (
                  <div className="wallet-approval-item" key={item.label}>
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <strong>{item.label}</strong>
                      <span>{formatNumber(item.count)} runs</span>
                      <div className="wallet-progress" style={{ marginTop: "8px" }}>
                        <div style={{ width: `${Math.max(4, Math.round(((Number(item.creditsUsed) || 0) / dailyPeak) * 100))}%` }} />
                      </div>
                    </div>
                    <span className="text-sm text-muted">{formatNumber(item.creditsUsed)} Cr</span>
                  </div>
                ))}
                {!daily.length && <div className="text-xs text-muted">Belum ada tren harian.</div>}
              </div>
            </div>
          </aside>
        </div>
      )}

      <div className="card">
        <div className="card-header">
          <div className="card-title">{ownerMode ? "Recent AI Runs (Redacted)" : "Recent AI Executions"}</div>
          <div style={{ display: "flex", gap: "8px" }}>
            <input type="text" className="form-input" placeholder={ownerMode ? "Search company or run ID..." : "Search contact or run ID..."} style={{ padding: "4px 8px", fontSize: "12px", width: "200px" }} />
          </div>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Type / Model</th>
                <th>Run ID</th>
                <th>Waktu</th>
                <th>{ownerMode ? "Company" : "Contact"}</th>
                <th style={{ textAlign: "right" }}>Tokens</th>
                <th style={{ textAlign: "right" }}>Cost (IDR)</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {displayedUsage.length ? displayedUsage.map((log) => {
                return (
                  <tr key={log.id} onClick={() => !ownerMode && inspectUsageLog && inspectUsageLog(log.id)} style={{ cursor: ownerMode ? "default" : "pointer" }}>
                    <td>
                      <div className="text-sm"><strong>{toLabel(log.usageType)}</strong></div>
                      <div className="text-xs text-muted">Model: {log.modelLabel}</div>
                    </td>
                    <td><strong className="text-muted text-sm">#{log.runId.substring(0, 8)}</strong></td>
                    <td className="text-muted text-sm">{formatDateTime(log.createdAt)}</td>
                    <td>
                      <div className="text-sm">{log.contactLabel}</div>
                      {ownerMode ? <div className="text-xs text-muted">Content redacted for privacy</div> : log.questionSummary && <div className="text-xs text-muted">{truncateText(log.questionSummary, 52)}</div>}
                    </td>
                    <td style={{ textAlign: "right", fontFamily: "monospace", fontSize: "13px" }}>
                      {formatNumber(log.totalTokens)}
                    </td>
                    <td style={{ textAlign: "right" }}>
                      <strong style={{ color: "var(--orange)" }}>{formatCurrencyIDR(log.costIdr)}</strong>
                    </td>
                    <td>
                      <span className={`badge ${log.statusClass}`}>
                        {log.statusLabel}
                      </span>
                    </td>
                  </tr>
                );
              }) : (
                <tr>
                  <td colSpan="7" style={{ textAlign: "center", padding: "32px", color: "var(--text-dim)" }}>
                    Tidak ada log penggunaan tercatat.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {displayedUsage.length > 0 && (
          <div className="card-footer flex-between">
            <span className="text-sm text-muted">Menampilkan {displayedUsage.length} logs terbaru.</span>
            <div style={{ display: "flex", gap: "4px" }}>
              <button className="btn btn-secondary btn-sm" disabled>Prev</button>
              <button className="btn btn-secondary btn-sm">Next</button>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

