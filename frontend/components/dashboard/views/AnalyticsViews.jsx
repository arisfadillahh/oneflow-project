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

export function AnalyticsView({ role, analytics, analyticsRange, setAnalyticsRange, aiRuns, inbox, contacts, agents }) {
  return (
    <PerformanceDashboardView
      role={role}
      analytics={analytics}
      analyticsRange={analyticsRange}
      setAnalyticsRange={setAnalyticsRange}
      aiRuns={aiRuns}
      inbox={inbox}
      contacts={contacts}
      agents={agents}
      mode="analytics"
    />
  );
}

const analyticsDayFormatter = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", timeZone: "UTC" });

function utcDayKey(date) {
  return date.toISOString().slice(0, 10);
}

function todayUtcDate() {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
}

function addUtcDays(date, days) {
  const next = new Date(date);
  next.setUTCDate(next.getUTCDate() + days);
  return next;
}

function analyticsItemDateKey(item) {
  const raw = item?.date || item?.day || item?.createdAt || item?.created_at;
  if (!raw) return "";
  const parsed = new Date(raw);
  if (!Number.isFinite(parsed.getTime())) return String(raw).slice(0, 10);
  return utcDayKey(new Date(Date.UTC(parsed.getUTCFullYear(), parsed.getUTCMonth(), parsed.getUTCDate())));
}

function buildSevenDayWindow(items, anchorDate, mapValue) {
  const byDate = new Map();
  items.forEach((item) => {
    const key = analyticsItemDateKey(item);
    if (!key) return;
    byDate.set(key, item);
  });
  return Array.from({ length: 7 }, (_, index) => {
    const date = addUtcDays(anchorDate, index - 6);
    const key = utcDayKey(date);
    const item = byDate.get(key) || {};
    return {
      ...mapValue(item),
      date: key,
      label: analyticsDayFormatter.format(date),
      isToday: key === utcDayKey(todayUtcDate()),
    };
  });
}

function ChartWindowControls({ anchorDate, setAnchorDate }) {
  const today = todayUtcDate();
  const canGoNext = anchorDate.getTime() < today.getTime();
  return (
    <div className="chart-window-controls">
      <button type="button" className="chart-window-btn" onClick={() => setAnchorDate((current) => addUtcDays(current, -1))} aria-label="Lihat hari sebelumnya">
        <span aria-hidden="true">‹</span>
      </button>
      <span>{analyticsDayFormatter.format(addUtcDays(anchorDate, -6))} - {analyticsDayFormatter.format(anchorDate)}</span>
      <button type="button" className="chart-window-btn" onClick={() => setAnchorDate((current) => addUtcDays(current, 1))} disabled={!canGoNext} aria-label="Lihat hari berikutnya">
        <span aria-hidden="true">›</span>
      </button>
    </div>
  );
}

function PerformanceTrendChart({ items = [], rangeLabel }) {
  const [anchorDate, setAnchorDate] = useState(() => todayUtcDate());
  const visiblePoints = buildSevenDayWindow(items, anchorDate, (item) => ({
    aiReplies: Number(item.aiReplies) || 0,
    humanReplies: Number(item.humanReplies) || 0,
    inbound: Number(item.inbound) || 0,
  }));
  const maxTotal = Math.max(1, ...visiblePoints.map((point) => point.aiReplies + point.humanReplies));

  return (
    <div className="card">
      <div className="card-header">
        <div><div className="card-title">AI vs Human Ratio</div></div>
        <ChartWindowControls anchorDate={anchorDate} setAnchorDate={setAnchorDate} />
      </div>
      <div className="chart-area" style={{ height: "200px" }}>
        <div className="chart-bars analytics-seven-day-bars" style={{ gap: "10px" }}>
          {visiblePoints.map((point, index) => {
            const aiHeight = point.aiReplies ? Math.max(3, Math.round((point.aiReplies / maxTotal) * 100)) : 0;
            const humanHeight = point.humanReplies ? Math.max(3, Math.round((point.humanReplies / maxTotal) * 100)) : 0;
            return (
              <div key={`${point.date}-${index}`} className={`analytics-stacked-day ${point.isToday ? "is-today" : ""}`}>
                <div style={{ flex: 1, width: "100%", display: "flex", flexDirection: "column", justifyContent: "flex-end", gap: "2px" }}>
                  <div title={`Human ${formatNumber(point.humanReplies)}`} style={{ height: `${humanHeight}%`, background: "var(--orange)", opacity: 0.8, borderRadius: "2px 2px 0 0", minHeight: point.humanReplies ? "3px" : 0 }} />
                  <div title={`AI ${formatNumber(point.aiReplies)}`} style={{ height: `${aiHeight}%`, background: "var(--blue)", opacity: 0.75, borderRadius: "2px 2px 0 0", minHeight: point.aiReplies ? "3px" : 0 }} />
                </div>
                <div style={{ fontSize: "10px", color: "var(--gray-400)" }}>{String(point.label || "-").slice(0, 5)}</div>
              </div>
            );
          })}
        </div>
      </div>
      <div className="flex-between mt-8">
        <div className="flex gap-16 text-sm">
          <div className="flex-center gap-4"><div style={{ width: "10px", height: "10px", borderRadius: "50%", background: "var(--blue)" }} /> AI Replies</div>
          <div className="flex-center gap-4"><div style={{ width: "10px", height: "10px", borderRadius: "50%", background: "var(--orange)" }} /> Human Replies</div>
        </div>
        <div className="text-xs text-muted">{rangeLabel}</div>
      </div>
    </div>
  );
}

function DailyVolumeChart({ items = [] }) {
  const [anchorDate, setAnchorDate] = useState(() => todayUtcDate());
  const visiblePoints = buildSevenDayWindow(items, anchorDate, (item) => ({
    value: Number(item.inbound) || Number(item.opened) || 0,
  }));
  const maxValue = Math.max(1, ...visiblePoints.map((point) => point.value));
  return (
    <div className="card">
      <div className="card-header">
        <div><div className="card-title">Volume Percakapan Harian</div></div>
        <ChartWindowControls anchorDate={anchorDate} setAnchorDate={setAnchorDate} />
      </div>
      <div className="chart-area" style={{ height: "200px" }}>
        <div className="chart-bars analytics-seven-day-bars">
          {visiblePoints.map((point, index) => (
            <div
              key={`${point.date}-${index}`}
              className={`chart-bar analytics-day-bar ${point.isToday ? "is-today" : ""}`}
              title={`${point.label}: ${formatNumber(point.value)}`}
              style={{ height: `${point.value ? Math.max(4, Math.round((point.value / maxValue) * 100)) : 0}%` }}
            />
          ))}
        </div>
      </div>
      <div className="flex-between mt-8 text-xs text-muted">
        <span>{visiblePoints[0]?.label || "-"}</span>
        <span>{visiblePoints[Math.floor((visiblePoints.length - 1) / 2)]?.label || "-"}</span>
        <span>{visiblePoints[visiblePoints.length - 1]?.label || "-"}</span>
      </div>
    </div>
  );
}

function ResponseTimeDistribution({ aiRuns = [] }) {
  const buckets = [
    { label: "5s", max: 5000 },
    { label: "10s", max: 10000 },
    { label: "15s", max: 15000 },
    { label: "20s", max: 20000 },
    { label: "25s", max: 25000 },
    { label: "30s", max: 30000 },
    { label: "35s", max: 35000 },
    { label: "40s", max: 40000 },
    { label: "45s", max: 45000 },
    { label: "50s", max: 50000 },
    { label: "55s", max: 55000 },
    { label: "60s", max: 60000 },
    { label: "65s", max: 65000 },
    { label: "70s", max: Infinity },
  ].map((bucket) => ({ ...bucket, count: 0 }));
  aiRuns.forEach((run) => {
    const latency = Number(run.latencyMS) || 0;
    if (!latency) return;
    const bucket = buckets.find((item) => latency <= item.max) || buckets[buckets.length - 1];
    bucket.count += 1;
  });
  const total = buckets.reduce((sum, bucket) => sum + bucket.count, 0);
  const maxCount = Math.max(1, ...buckets.map((bucket) => bucket.count));
  return (
    <div className="card">
      <div className="card-header">
        <div><div className="card-title">Response Time Distribution</div><div className="card-subtitle">Distribusi waktu respons AI (dalam detik)</div></div>
      </div>
      {total ? (
        <div className="chart-area" style={{ height: "140px" }}>
          <div className="chart-bars" style={{ gap: "6px" }}>
            {buckets.map((bucket, index) => {
              const tone = index < 3 ? "var(--green)" : index < 8 ? "var(--blue)" : "var(--orange)";
              return (
                <div key={bucket.label} style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", gap: "4px", height: "100%" }}>
                  <div style={{ flex: 1, width: "100%", display: "flex", alignItems: "flex-end" }}>
                    <div title={`${bucket.label}: ${formatNumber(bucket.count)}`} style={{ height: `${bucket.count ? Math.max(4, Math.round((bucket.count / maxCount) * 100)) : 0}%`, width: "100%", background: tone, opacity: 0.8, borderRadius: "3px 3px 0 0", transition: "height 0.5s ease" }} />
                  </div>
                  <div style={{ fontSize: "9px", color: "var(--gray-400)" }}>{bucket.label}</div>
                </div>
              );
            })}
          </div>
        </div>
      ) : (
        <EmptyState title="Belum ada latency" copy="Distribusi muncul setelah ai_runs menyimpan latency_ms." compact />
      )}
    </div>
  );
}

function minutesSince(dateString) {
  if (!dateString) return 0;
  const value = new Date(dateString).getTime();
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.floor((Date.now() - value) / 60000));
}

function knowledgeGapLabel(run) {
  const reason = String(run?.escalationReason || "").trim();
  if (reason) return reason.length > 72 ? `${reason.slice(0, 72)}...` : reason;
  const metadata = run?.retrievalMetadata?.retrieval ?? run?.retrievalMetadata ?? {};
  const queryType = metadata?.orchestrator?.queryType || metadata?.generation?.queryType || metadata?.intentAnalyzer?.queryType;
  if (queryType) return `Low confidence: ${toLabel(String(queryType))}`;
  if (Number(run?.confidenceScore) > 0 && Number(run?.confidenceScore) < 0.55) return "Low confidence answer";
  return "Needs knowledge review";
}

function groupKnowledgeGaps(aiRuns) {
  const grouped = new Map();
  aiRuns.forEach((run) => {
    const confidence = Number(run.confidenceScore) || 0;
    const isGap = run.decision !== "answer" || confidence < 0.55 || String(run.escalationReason || "").trim();
    if (!isGap) return;
    const label = knowledgeGapLabel(run);
    const current = grouped.get(label) ?? { label, count: 0, example: "" };
    current.count += 1;
    if (!current.example) current.example = run.questionSummary || run.escalationReason || "-";
    grouped.set(label, current);
  });
  return [...grouped.values()].sort((a, b) => b.count - a.count).slice(0, 5);
}

function averageLatencyFromRuns(aiRuns) {
  const values = aiRuns.map((run) => Number(run.latencyMS) || 0).filter((value) => value > 0);
  if (!values.length) return 0;
  return Math.round(values.reduce((sum, value) => sum + value, 0) / values.length);
}

function escalationReasonsFromRuns(aiRuns) {
  const grouped = new Map();
  aiRuns.forEach((run) => {
    const reason = String(run.escalationReason || "").trim();
    if (!reason) return;
    grouped.set(reason, (grouped.get(reason) || 0) + 1);
  });
  return [...grouped.entries()]
    .map(([reason, count]) => ({ reason, count }))
    .sort((a, b) => b.count - a.count || a.reason.localeCompare(b.reason))
    .slice(0, 5);
}

export function PerformanceDashboardView({ role, analytics, analyticsRange, setAnalyticsRange, summary = null, aiRuns = [], inbox = [], contacts = [], agents = [], mode = "dashboard" }) {
  const isSuperAdmin = role === "super_admin";
  const isAnalyticsPage = mode === "analytics";
  const inbound = Number(analytics?.inboundCount) || 0;
  const escalated = Number(analytics?.escalatedCount) || 0;
  const aiAnswered = Number(analytics?.aiAnsweredCount) || 0;
  const analyticsAvgLatency = Number(analytics?.avgAiLatencyMs);
  const avgLatency = Number.isFinite(analyticsAvgLatency) ? analyticsAvgLatency : averageLatencyFromRuns(aiRuns);
  const series = Array.isArray(analytics?.series) ? analytics.series : [];
  const agentPerformance = Array.isArray(analytics?.agentPerformance) ? analytics.agentPerformance : [];
  const escalationReasons = (Array.isArray(analytics?.escalationReasons) && analytics.escalationReasons.length)
    ? analytics.escalationReasons
    : escalationReasonsFromRuns(aiRuns);
  const totalEscalationReasons = escalationReasons.reduce((sum, item) => sum + (Number(item.count) || 0), 0);
  const rangeOptions = [
    { key: "daily", label: "1D" },
    { key: "weekly", label: "1W" },
    { key: "monthly", label: "1M" },
  ];
  const crm = summary?.crm || {};

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">{isAnalyticsPage ? "Laporan" : "Laporan Performa"}</div>
          <div className="page-desc">Ringkasan percakapan dan performa AI.</div>
        </div>
        <div className="page-actions">
          <div style={{ display: "flex", gap: "4px", background: "var(--gray-100)", borderRadius: "8px", padding: "4px" }}>
            {rangeOptions.map((opt) => (
              <button
                key={opt.key}
                type="button"
                className={`btn btn-ghost btn-sm ${analyticsRange === opt.key ? "btn-primary" : ""}`}
                style={analyticsRange === opt.key ? { background: "var(--blue)", color: "white" } : undefined}
                onClick={() => setAnalyticsRange(opt.key)}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="kpi-grid" style={{ gridTemplateColumns: "repeat(5, 1fr)" }}>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg></div>
          <div className="kpi-value">{formatNumber(inbound)}</div>
          <div className="kpi-label">Total Chats</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon green"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m12 3-1.9 5.1L5 10l5.1 1.9L12 17l1.9-5.1L19 10l-5.1-1.9L12 3Z"/></svg></div>
          <div className="kpi-value">
            {formatNumber(aiAnswered)}
          </div>
          <div className="kpi-label">AI Answers</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon orange"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><path d="M16 12l-4-4-4 4M12 8v8"/></svg></div>
          <div className="kpi-value">{formatNumber(escalated)}</div>
          <div className="kpi-label">Escalations</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon purple"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></div>
          <div className="kpi-value">{Math.round(avgLatency)}ms</div>
          <div className="kpi-label">Avg Response</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg></div>
          <div className="kpi-value">{formatNumber(contacts?.length || 0)}</div>
          <div className="kpi-label">Active Contacts</div>
        </div>
      </div>

      <div className="grid-2 mb-24">
        <DailyVolumeChart items={series} />
        <PerformanceTrendChart items={series} rangeLabel={analyticsRange} />
      </div>

      {!isAnalyticsPage ? (
        <div className="card mb-24">
          <div className="card-header">
            <div>
              <div className="card-title">CRM Hari Ini</div>
              <div className="card-subtitle">Peluang penjualan yang perlu ditindaklanjuti.</div>
            </div>
          </div>
          <div className="deals-summary-grid dashboard-crm-summary-grid">
            <div className="kpi-card deals-kpi-card">
              <div className="kpi-icon blue" />
              <div><div className="kpi-value">{formatNumber(crm.openDeals || 0)}</div><div className="kpi-label">Open Deal</div><div className="kpi-meta">{formatCurrencyIDR(crm.openDealValue || 0)}</div></div>
            </div>
            <div className="kpi-card deals-kpi-card">
              <div className="kpi-icon purple" />
              <div><div className="kpi-value">{formatCurrencyIDR(crm.weightedDealValue || 0)}</div><div className="kpi-label">Forecast</div><div className="kpi-meta">Weighted by stage</div></div>
            </div>
            <div className="kpi-card deals-kpi-card">
              <div className="kpi-icon orange" />
              <div><div className="kpi-value">{formatNumber((crm.overdueFollowUps || 0) + (crm.overdueDealTasks || 0))}</div><div className="kpi-label">Overdue Follow-up</div><div className="kpi-meta">Contact + deal task</div></div>
            </div>
            <div className="kpi-card deals-kpi-card">
              <div className="kpi-icon red" />
              <div><div className="kpi-value">{formatNumber(crm.pendingConversations || 0)}</div><div className="kpi-label">Pending Chat</div><div className="kpi-meta">Butuh agent</div></div>
            </div>
            <div className="kpi-card deals-kpi-card">
              <div className="kpi-icon green" />
              <div><div className="kpi-value">{truncateText(crm.topOwner?.name || "-", 18)}</div><div className="kpi-label">Top Owner</div><div className="kpi-meta">{formatNumber(crm.topOwner?.openDeals || 0)} open deal</div></div>
            </div>
          </div>
        </div>
      ) : null}

      <div className="grid-2 mb-24">
        <div className="card">
          <div className="card-header"><div className="card-title">Escalation Reasons</div></div>
          {escalationReasons.length ? (
            <div className="table-wrap">
              <table>
                <thead><tr><th>Alasan</th><th>Count</th><th>%</th><th>Trend</th></tr></thead>
                <tbody>
                  {escalationReasons.map((item, index) => {
                    const count = Number(item.count) || 0;
                    const pct = totalEscalationReasons ? Math.round((count / totalEscalationReasons) * 100) : 0;
                    return (
                      <tr key={`${item.reason}-${index}`}>
                        <td>{item.reason}</td>
                        <td><strong>{formatNumber(count)}</strong></td>
                        <td>
                          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                            <div style={{ flex: 1, height: "5px", background: "var(--gray-100)", borderRadius: "3px" }}>
                              <div style={{ height: "100%", width: `${pct}%`, background: "var(--orange)", borderRadius: "3px" }} />
                            </div>
                            <span className="text-sm">{pct}%</span>
                          </div>
                        </td>
                        <td><span className="kpi-trend neutral" style={{ position: "static" }}>Data</span></td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState title="Belum ada data eskalasi" copy="Reason akan muncul setelah AI menyimpan escalation_reason." compact />
          )}
        </div>

        <div className="card">
          <div className="card-header"><div className="card-title">Team Performance</div></div>
          <div className="table-wrap">
            <table>
              <thead><tr><th>Agent</th><th>Replies</th><th>Assigned</th><th>Resolved</th><th>Role</th></tr></thead>
              <tbody>
                {agentPerformance.length ? agentPerformance.map((a) => (
                  <tr key={a.id || a.username || a.name}>
                    <td><strong>{a.name}</strong></td>
                    <td>{formatNumber(a.replies)}</td>
                    <td>{formatNumber(a.assignedOpen)}</td>
                    <td>{formatNumber(a.resolved)}</td>
                    <td><span className={`badge ${a.role === "admin" ? "purple" : a.role === "operator" ? "green" : "blue"}`}>{toRoleLabel(a.role)}</span></td>
                  </tr>
                )) : (
                  <tr>
                    <td colSpan="5">
                      <EmptyState title="Belum ada data agent" copy="Performa agent akan muncul setelah ada reply human atau resolved event." compact />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <ResponseTimeDistribution aiRuns={aiRuns} />
    </>
  );
}


export function OwnerOverviewView({ analytics, summary, wallet, purchases, health, creditAdjustments = [] }) {
  const requested = purchases.filter((item) => ["requested", "pending"].includes(item.paymentStatus));
  const healthAlerts = health?.systemAlerts?.length ? health.systemAlerts : (health?.alerts?.map(a => ({ message: a, severity: "warning" })) ?? []);
  const recentBillingActivity = useMemo(() => (
    [...purchases]
      .sort((a, b) => new Date(b.createdAt || 0).getTime() - new Date(a.createdAt || 0).getTime())
      .slice(0, 5)
  ), [purchases]);
  const netCreditAdjustments = useMemo(() => creditAdjustments.reduce((total, item) => {
    const amount = Number(item.amount ?? 0) || 0;
    if (String(item.adjustmentType || "").startsWith("subtract_")) return total - amount;
    if (String(item.adjustmentType || "").startsWith("add_")) return total + amount;
    return total;
  }, 0), [creditAdjustments]);
  const ownerUsage = useMemo(() => {
    const toNumber = (value) => {
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : 0;
    };
    const monthlyCredits = toNumber(wallet?.monthlyCreditsRemaining ?? analytics?.monthlyRemaining);
    const additionalCredits = toNumber(wallet?.additionalCreditsRemaining ?? analytics?.additionalRemaining);
    const thirtyDaysAgo = Date.now() - (30 * 24 * 60 * 60 * 1000);
    const confirmedSales = purchases.reduce((total, item) => {
      if (!["confirmed", "paid"].includes(item.paymentStatus)) return total;
      const purchaseTime = new Date(item.confirmedAt || item.createdAt || 0).getTime();
      if (!Number.isFinite(purchaseTime) || purchaseTime < thirtyDaysAgo) return total;
      return total + toNumber(item.price);
    }, 0);
    const sourceSeries = Array.isArray(analytics?.series) ? analytics.series : [];
    const seriesByDate = new Map(sourceSeries.filter((item) => item.date).map((item) => [item.date, item]));
    const dateFormatter = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", timeZone: "UTC" });
    const chartSeries = seriesByDate.size
      ? Array.from({ length: 30 }, (_, index) => {
          const now = new Date();
          const date = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
          date.setUTCDate(date.getUTCDate() - (29 - index));
          const key = date.toISOString().slice(0, 10);
          const item = seriesByDate.get(key);
          return {
            label: dateFormatter.format(date),
            creditsUsed: toNumber(item?.creditsUsed),
            playgroundRuns: toNumber(item?.playgroundRuns),
            liveInboundRuns: toNumber(item?.liveInboundRuns),
          };
        })
      : sourceSeries.slice(-30).map((item) => ({
          label: item.label || "-",
          creditsUsed: toNumber(item.creditsUsed),
          playgroundRuns: toNumber(item.playgroundRuns),
          liveInboundRuns: toNumber(item.liveInboundRuns),
        }));

    return {
      totalActiveCredits: monthlyCredits + additionalCredits,
      monthlyCredits,
      additionalCredits,
      confirmedSales,
      aiRuns30d: toNumber(analytics?.totalUsageRecords),
      creditsUsed30d: toNumber(analytics?.totalCreditsUsed),
      aiCost30d: toNumber(analytics?.totalCostIdr),
      playgroundRuns: toNumber(analytics?.aiPlaygroundRuns),
      liveInboundRuns: toNumber(analytics?.liveInboundRuns),
      chartSeries,
      maxChartCredits: Math.max(1, ...chartSeries.map((item) => item.creditsUsed)),
    };
  }, [analytics, wallet, purchases]);

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Ringkasan</div>
          <div className="page-desc">Ringkasan pemakaian dan tagihan.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10"/><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14"/></svg>Refresh
          </button>
          <button className="btn btn-primary">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>Tambah Kredit
          </button>
        </div>
      </div>

      <div className="kpi-grid">
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg></div>
          <div className="kpi-value">{formatNumber(ownerUsage.totalActiveCredits)}</div>
          <div className="kpi-label">Total Kredit Aktif</div>
          <div className="kpi-trend neutral">{formatNumber(ownerUsage.monthlyCredits)} monthly</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon green"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg></div>
          <div className="kpi-value">{formatCurrencyIDR(ownerUsage.confirmedSales)}</div>
          <div className="kpi-label">Kredit Terjual (30 hari)</div>
          <div className="kpi-trend neutral">Confirmed DB</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon orange"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="9" cy="21" r="1"/><circle cx="20" cy="21" r="1"/><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"/></svg></div>
          <div className="kpi-value">{requested.length}</div>
          <div className="kpi-label">Purchase Pending</div>
          <div className="kpi-trend neutral">Menunggu</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon purple"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m12 3-1.9 5.1L5 10l5.1 1.9L12 17l1.9-5.1L19 10l-5.1-1.9L12 3Z"/></svg></div>
          <div className="kpi-value">{formatNumber(ownerUsage.aiRuns30d)}</div>
          <div className="kpi-label">AI Runs (30 hari)</div>
          <div className="kpi-trend neutral">{formatNumber(ownerUsage.creditsUsed30d)} kredit</div>
        </div>
        <div className="kpi-card">
          <div className={`kpi-icon ${healthAlerts.length > 0 ? "red" : "green"}`}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></div>
          <div className="kpi-value">{healthAlerts.length}</div>
          <div className="kpi-label">{healthAlerts.length > 0 ? "Critical Alerts" : "System Normal"}</div>
          <div className={`kpi-trend ${healthAlerts.length > 0 ? "down" : "up"}`}>{healthAlerts.length > 0 ? "Action needed" : "All good"}</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 3v18"/><path d="M17 8H9.5a3.5 3.5 0 0 0 0 7H14a3 3 0 0 1 0 6H6"/></svg></div>
          <div className="kpi-value">{formatNumber(creditAdjustments.length)}</div>
          <div className="kpi-label">Credit Adjustments</div>
          <div className="kpi-trend neutral">Net {formatNumber(netCreditAdjustments)} kredit</div>
        </div>
      </div>

      <div style={{ marginBottom: "24px" }}>
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">AI Usage (30 hari)</div>
              <div className="card-subtitle">{formatNumber(ownerUsage.aiRuns30d)} run, {formatCurrencyIDR(ownerUsage.aiCost30d)} cost dari database</div>
            </div>
            <div className="flex gap-8">
              <button className="btn btn-secondary btn-sm">1D</button>
              <button className="btn btn-primary btn-sm">1M</button>
              <button className="btn btn-secondary btn-sm">3M</button>
            </div>
          </div>
          <div className="chart-area" style={{ height: "180px" }}>
            <div className="chart-bars">
              {ownerUsage.chartSeries.map((item, i) => (
                <div
                  key={`${item.label}-${i}`}
                  className="chart-bar"
                  title={`${item.label}: ${formatNumber(item.creditsUsed)} kredit`}
                  style={{
                    height: `${Math.max(3, Math.round((item.creditsUsed / ownerUsage.maxChartCredits) * 100))}%`,
                    opacity: i === ownerUsage.chartSeries.length - 1 ? 1 : undefined,
                    background: i === ownerUsage.chartSeries.length - 1 ? "var(--blue-dark)" : undefined,
                  }}
                />
              ))}
            </div>
          </div>
          <div className="flex-between mt-8 text-xs text-muted">
            <span>{ownerUsage.chartSeries[0]?.label ?? "-"}</span><span>{ownerUsage.chartSeries[Math.floor(ownerUsage.chartSeries.length / 2)]?.label ?? "-"}</span><span>{ownerUsage.chartSeries[ownerUsage.chartSeries.length - 1]?.label ?? "-"}</span>
          </div>
        </div>
      </div>

      <div className="grid-2">
        <div className="card">
          <div className="card-header">
            <div className="card-title">Recent System Alerts</div>
            {healthAlerts.length > 0 && <span className="badge red">{healthAlerts.length} Critical</span>}
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
            {healthAlerts.length ? healthAlerts.map((alert, idx) => (
              <div key={idx} style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px", background: "var(--red-light)", borderRadius: "8px", border: "1px solid var(--red-mid)" }}>
                <svg viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth="2" style={{ width: "16px", height: "16px", flexShrink: 0 }}><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: "13px", fontWeight: "600", color: "var(--red)" }}>{alert.message}</div>
                  <div className="text-xs text-muted">Baru saja</div>
                </div>
                <span className="badge red">{alert.severity?.toUpperCase() || "CRITICAL"}</span>
              </div>
            )) : (
              <div style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px", background: "var(--green-light)", borderRadius: "8px", border: "1px solid var(--green-mid)" }}>
                <svg viewBox="0 0 24 24" fill="none" stroke="#16A34A" strokeWidth="2" style={{ width: "16px", height: "16px", flexShrink: 0 }}><path d="M20 6 9 17l-5-5"/></svg>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: "13px", fontWeight: "600", color: "var(--green)" }}>System is fully operational</div>
                  <div className="text-xs text-muted">Semua servis berjalan lancar.</div>
                </div>
                <span className="badge green">Normal</span>
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">Aktivitas Billing</div>
              <div className="card-subtitle">Transaksi paket dan top-up terakhir dari Midtrans.</div>
            </div>
            <span className="badge gray">{recentBillingActivity.length} terbaru</span>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr><th>Paket</th><th>Status</th><th>Nilai</th></tr>
              </thead>
              <tbody>
                {recentBillingActivity.map((item) => (
                  <tr key={item.id}>
                    <td>
                      <strong>{item.packageName || "Paket"}</strong>
                      <div className="text-xs text-muted">{formatDateTime(item.createdAt)}</div>
                    </td>
                    <td><StatusPill value={item.paymentStatus || "pending"} kind="payment" /></td>
                    <td>{formatCurrencyIDR(item.price)}</td>
                  </tr>
                ))}
                {!recentBillingActivity.length && (
                  <tr>
                    <td colSpan={3} className="text-sm text-muted">Belum ada transaksi billing.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}
