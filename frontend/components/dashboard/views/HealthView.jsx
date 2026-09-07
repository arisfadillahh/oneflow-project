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

export function HealthView({ health }) {
  const isHealthy = health?.status === "ok";
  const alerts = health?.systemAlerts?.length ? health.systemAlerts : (health?.alerts?.map(a => ({ message: a, severity: "warning" })) ?? []);
  
  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Status Sistem</div>
          <div className="page-desc">Status sistem dan koneksi.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10"/><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14"/></svg>Refresh Status</button>
        </div>
      </div>

      <div className="kpi-grid" style={{ gridTemplateColumns: "repeat(4, 1fr)", marginBottom: "24px" }}>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg></div>
          <div className="kpi-value">99.9%</div><div className="kpi-label">System Uptime (30d)</div><div className="kpi-trend up">SLA Met</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon green"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg></div>
          <div className="kpi-value">24ms</div><div className="kpi-label">API Latency</div><div className="kpi-trend up">Optimal</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon orange"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></div>
          <div className="kpi-value">{alerts.length}</div><div className="kpi-label">Active Warnings</div><div className={`kpi-trend ${alerts.length ? "down" : "up"}`}>{alerts.length ? "Action Needed" : "All Clear"}</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon purple"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 2v6h-6"/><path d="M3 12a9 9 0 0 1 15-6.7L21 8"/><path d="M3 22v-6h6"/><path d="M21 12a9 9 0 0 1-15 6.7L3 16"/></svg></div>
          <div className="kpi-value">0%</div><div className="kpi-label">Error Rate</div><div className="kpi-trend up">Normal</div>
        </div>
      </div>

      <div className="grid-2">
        <div className="card">
          <div className="card-header"><div className="card-title">Node & Service Status</div></div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr><th>Component</th><th>Status</th><th>Latency</th><th>Last Check</th></tr>
              </thead>
              <tbody>
                <tr>
                  <td><strong>Web App</strong></td>
                  <td><span className="badge green"><span className="badge-dot" />Operational</span></td>
                  <td>12ms</td>
                  <td className="text-muted">Baru saja</td>
                </tr>
                <tr>
                  <td><strong>WhatsApp Gateway</strong></td>
                  <td><span className="badge green"><span className="badge-dot" />Operational</span></td>
                  <td>45ms</td>
                  <td className="text-muted">Baru saja</td>
                </tr>
                <tr>
                  <td><strong>AI Engine (OpenAI)</strong></td>
                  <td><span className="badge green"><span className="badge-dot" />Operational</span></td>
                  <td>850ms</td>
                  <td className="text-muted">Baru saja</td>
                </tr>
                <tr>
                  <td><strong>Vector Database</strong></td>
                  <td><span className="badge green"><span className="badge-dot" />Operational</span></td>
                  <td>18ms</td>
                  <td className="text-muted">Baru saja</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div className="card">
          <div className="card-header"><div className="card-title">System Event Log</div></div>
          <div style={{ display: "flex", flexDirection: "column", gap: "16px", marginTop: "8px" }}>
            {alerts.map((alert, i) => (
              <div key={i} style={{ display: "flex", gap: "12px" }}>
                <div style={{ width: "8px", height: "8px", borderRadius: "50%", background: "var(--red)", marginTop: "6px" }} />
                <div>
                  <div style={{ fontSize: "13px", fontWeight: "600", color: "var(--red)" }}>{alert.message}</div>
                  <div className="text-xs text-muted">Baru saja</div>
                </div>
              </div>
            ))}
            <div style={{ display: "flex", gap: "12px" }}>
              <div style={{ width: "8px", height: "8px", borderRadius: "50%", background: "var(--green)", marginTop: "6px" }} />
              <div>
                <div style={{ fontSize: "13px", fontWeight: "600" }}>System backup completed successfully</div>
                <div className="text-xs text-muted">02:00 WIB (Hari ini)</div>
              </div>
            </div>
            <div style={{ display: "flex", gap: "12px" }}>
              <div style={{ width: "8px", height: "8px", borderRadius: "50%", background: "var(--blue)", marginTop: "6px" }} />
              <div>
                <div style={{ fontSize: "13px", fontWeight: "600" }}>Deployment v2.4.1 finished</div>
                <div className="text-xs text-muted">14:20 WIB (Kemarin)</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}



