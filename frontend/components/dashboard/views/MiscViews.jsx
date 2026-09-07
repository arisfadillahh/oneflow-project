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

export function RoadmapView({ title, copy, requiredItems }) {
  return (
    <div className="page-stack">
      <SectionCard title={title} subtitle={copy}>
        <ul className="flat-list">
          {requiredItems.map((item) => <li key={item}>{item}</li>)}
        </ul>
      </SectionCard>
    </div>
  );
}

export function SeriesChart({ items, valueKeys }) {
  const maxValue = Math.max(1, ...items.flatMap((item) => valueKeys.map(({ key }) => Number(item[key]) || 0)));
  return (
    <div className="chart-stack">
      {items.length ? items.map((item) => (
        <div key={item.label} className="chart-row">
          <div className="chart-label">{item.label}</div>
          <div className="chart-bars">
            {valueKeys.map(({ key, label }) => (
              <div key={key} className="chart-bar-item">
                <span className="chart-bar-label">{label}</span>
                <div className="chart-bar-track">
                  <div className="chart-bar-fill" style={{ width: `${((Number(item[key]) || 0) / maxValue) * 100}%` }} />
                </div>
                <span className="chart-bar-value">{formatNumber(item[key])}</span>
              </div>
            ))}
          </div>
        </div>
      )) : <EmptyState title="Belum ada series data" copy="Window ini belum punya bucket analytics." compact />}
    </div>
  );
}



