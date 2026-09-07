import { Icons, toLabel } from "../../../lib/dashboard-core";

export function SectionCard({ title, subtitle, children }) {
  return (
    <section className="card surface-card">
      <div className="card-header panel-header">
        <div>
          <h2 className="card-title">{title}</h2>
          {subtitle ? <p className="card-subtitle">{subtitle}</p> : null}
        </div>
      </div>
      {children}
    </section>
  );
}

export function StatCard({ label, value, meta }) {
  return (
    <article className="kpi-card stat-card">
      <span className="kpi-label stat-label">{label}</span>
      <strong className="kpi-value stat-value">{value}</strong>
      <span className="kpi-trend stat-meta">{meta}</span>
    </article>
  );
}

export function DetailRow({ label, value, multiline = false }) {
  return (
    <div className={`detail-row ${multiline ? "multiline" : ""}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

export function StatusPill({ value, kind }) {
  return <span className={`badge pill ${kind} ${String(value).replace(/_/g, "-")}`}>{toLabel(String(value))}</span>;
}

export function EmptyState({ title, copy, compact = false, action = null }) {
  return (
    <div className={`empty-state ${compact ? "compact" : ""}`}>
      <strong>{title}</strong>
      <p>{copy}</p>
      {action ? <div className="empty-state-action">{action}</div> : null}
    </div>
  );
}

export function Notice({ tone, title, text }) {
  return (
    <div className={`notice ${tone}`}>
      <span className="notice-icon">{tone === "danger" ? Icons.alert : tone === "warn" ? Icons.alert : Icons.check}</span>
      <div>
        <strong>{title}</strong>
        <p>{text}</p>
      </div>
    </div>
  );
}
