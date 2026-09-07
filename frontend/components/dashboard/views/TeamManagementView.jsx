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

function organizationRoleLabel(role) {
  if (role === "org_owner") return "Primary Admin";
  if (role === "org_admin") return "Org Admin";
  if (role === "supervisor") return "Supervisor";
  if (role === "operator") return "Agent";
  return toLabel(role);
}

function memberStatusLabel(status) {
  if (status === "active") return "Aktif";
  if (status === "pending") return "Menunggu";
  if (status === "expired") return "Kedaluwarsa";
  if (status === "disabled") return "Nonaktif";
  return status ? toLabel(status) : "-";
}

export function TeamManagementView({
  agents = [],
  members = [],
  invites = [],
  inviteForm,
  setInviteForm,
  createInvite,
  revokeInvite,
  updateMember,
  removeMember,
  agentForm,
  setAgentForm,
  submitAgent,
  resetAgentPassword,
  canManageAgentAccounts = false,
  busyKey
}) {
  const [teamSearch, setTeamSearch] = useState("");
  const filteredAgents = useMemo(() => {
    const query = teamSearch.trim().toLowerCase();
    if (!query) return agents;
    return agents.filter((agent) => {
      const haystack = [
        agent.name,
        agent.username,
        agent.phone,
        agent.role,
        toRoleLabel(agent.role),
      ].join(" ").toLowerCase();
      return haystack.includes(query);
    });
  }, [agents, teamSearch]);

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Tim</div>
          <div className="page-desc">Tambah anggota tim dan atur aksesnya.</div>
        </div>
        {canManageAgentAccounts ? (
          <div className="page-actions">
            <button className="btn btn-primary" onClick={() => setAgentForm({ id: "", name: "", username: "", role: "operator", password: "" })}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>Tambah Akun Tim
            </button>
          </div>
        ) : null}
      </div>

      <div className="card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="card-title">Akses Organisasi</div>
            <div className="card-subtitle">Daftar user yang punya akses ke bisnis aktif beserta role organisasinya.</div>
          </div>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Anggota</th>
                <th>Role</th>
                <th>Status</th>
                <th>Bergabung</th>
                <th>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => (
                <tr key={member.id}>
                  <td>
                    <strong>{member.name || member.username}</strong>
                    <div className="text-xs text-muted">@{member.username}</div>
                  </td>
                  <td>
                    <select
                      className="form-select"
                      value={member.organizationRole || "operator"}
                      disabled={member.organizationRole === "org_owner" || busyKey === `team-member-${member.id}`}
                      onChange={(event) => updateMember?.(member.id, event.target.value, member.status !== "disabled")}
                    >
                      <option value="org_admin">Org Admin</option>
                      <option value="supervisor">Supervisor</option>
                      <option value="operator">Agent</option>
                      <option value="org_owner">Primary Admin</option>
                    </select>
                  </td>
                  <td><span className={`badge ${member.status === "active" ? "green" : "gray"}`}>{memberStatusLabel(member.status)}</span></td>
                  <td className="text-muted text-sm">{formatDateTime(member.joinedAt)}</td>
                  <td>
                    <button className="btn btn-danger btn-sm" type="button" disabled={member.organizationRole === "org_owner" || busyKey === `team-member-${member.id}`} onClick={() => removeMember?.(member.id)}>Nonaktifkan</button>
                  </td>
                </tr>
              ))}
              {!members.length ? (
                <tr><td colSpan={5} style={{ textAlign: "center", padding: "24px", color: "var(--gray-500)" }}>Belum ada anggota di organisasi ini.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="card-title">Undangan Tim</div>
            <div className="card-subtitle">Tambahkan user baru lewat email atau nomor WhatsApp. Undangan berlaku 7 hari.</div>
          </div>
        </div>
        <form className="inline-form" onSubmit={createInvite}>
          <input className="form-input" placeholder="email@company.com" value={inviteForm?.email || ""} onChange={(event) => setInviteForm?.({ ...(inviteForm || {}), email: event.target.value })} />
          <input className="form-input" placeholder="Nomor WhatsApp" value={inviteForm?.phone || ""} onChange={(event) => setInviteForm?.({ ...(inviteForm || {}), phone: event.target.value })} />
          <select className="form-select" value={inviteForm?.role || "operator"} onChange={(event) => setInviteForm?.({ ...(inviteForm || {}), role: event.target.value })}>
            <option value="org_admin">Org Admin</option>
            <option value="supervisor">Supervisor</option>
            <option value="operator">Agent</option>
          </select>
          <button className="btn btn-primary" type="submit" disabled={busyKey === "team-invite"}>Kirim Undangan</button>
        </form>
        <div className="table-wrap">
          <table>
            <thead>
              <tr><th>Tujuan</th><th>Role</th><th>Status</th><th>Berlaku Sampai</th><th>Aksi</th></tr>
            </thead>
            <tbody>
              {invites.map((invite) => (
                <tr key={invite.id}>
                  <td>{invite.email || invite.phone || "-"}</td>
                  <td><span className="badge blue">{organizationRoleLabel(invite.role)}</span></td>
                  <td><span className={`badge ${invite.status === "pending" ? "orange" : "gray"}`}>{memberStatusLabel(invite.status)}</span></td>
                  <td className="text-muted text-sm">{formatDateTime(invite.expiresAt)}</td>
                  <td><button className="btn btn-secondary btn-sm" type="button" disabled={invite.status !== "pending" || busyKey === `team-invite-${invite.id}`} onClick={() => revokeInvite?.(invite.id)}>Batalkan</button></td>
                </tr>
              ))}
              {!invites.length ? (
                <tr><td colSpan={5} style={{ textAlign: "center", padding: "24px", color: "var(--gray-500)" }}>Belum ada undangan.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="kpi-grid" style={{ gridTemplateColumns: "repeat(4, 1fr)", marginBottom: "24px" }}>
        <div className="kpi-card">
          <div className="kpi-icon blue"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg></div>
          <div className="kpi-value">{agents.length}</div><div className="kpi-label">Total Anggota</div><div className="kpi-trend up">Semua Role</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon green"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="m12 16 .01 0"/></svg></div>
          <div className="kpi-value">{agents.filter(a => a.isActive !== false).length}</div><div className="kpi-label">Akun Aktif</div><div className="kpi-trend up">Normal</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon orange"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg></div>
          <div className="kpi-value">{agents.filter(a => a.role === "operator").length}</div><div className="kpi-label">Agent</div><div className="kpi-trend up">Aktif melayani</div>
        </div>
        <div className="kpi-card">
          <div className="kpi-icon red"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></div>
          <div className="kpi-value">{agents.filter(a => a.role === "admin" || a.role === "super_admin").length}</div><div className="kpi-label">Admin</div><div className="kpi-trend neutral">Pengelola akses</div>
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <div className="card-title">Direktori Tim</div>
          <div className="searchbox team-searchbox">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            <input type="text" placeholder="Cari nama anggota..." value={teamSearch} onChange={(event) => setTeamSearch(event.target.value)} />
          </div>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Nama / Username</th>
                <th>Role Akses</th>
                <th>Status Akun</th>
                <th>Terakhir Login</th>
                <th>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {filteredAgents.map((a, i) => (
                <tr key={i}>
                  <td>
                    <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
                      <div className="avatar avatar-sm blue">{a.name ? a.name[0].toUpperCase() : a.username[0].toUpperCase()}</div>
                      <div>
                        <strong>{a.name || "Unknown"}</strong>
                        <div className="text-xs text-muted">@{a.username}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span className={`badge ${a.role === 'admin' ? 'red' : a.role === 'super_admin' ? 'purple' : 'blue'}`}>
                      {toRoleLabel(a.role)}
                    </span>
                  </td>
                  <td>
                    {a.isActive !== false ? <span className="badge green">Aktif</span> : <span className="badge gray">Nonaktif</span>}
                  </td>
                  <td className="text-muted text-sm">{formatDateTime(a.updatedAt || a.createdAt)}</td>
                  <td>
                    <div style={{ display: "flex", gap: "6px" }}>
                      {canManageAgentAccounts ? (
                        <>
                          <button className="btn btn-secondary btn-sm" onClick={() => setAgentForm({ ...a, name: a.name || "", username: a.username || "", phone: a.phone || "", role: a.role || "operator", isActive: a.isActive !== false })}>Edit</button>
                          <button className="btn btn-secondary btn-sm" disabled={busyKey === `agent-reset-${a.id}`} onClick={() => resetAgentPassword?.(a.id, a.username)}>Reset Password</button>
                          <button className="btn btn-danger btn-sm" disabled={busyKey === "agent" || a.isActive === false} onClick={() => setAgentForm({ ...a, name: a.name || "", username: a.username || "", phone: a.phone || "", role: a.role || "operator", isActive: false })}>Cabut Akses</button>
                        </>
                      ) : (
                        <span className="text-xs text-muted">Kelola lewat role organisasi</span>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              {!filteredAgents.length && (
                <tr>
                  <td colSpan={5} style={{ textAlign: "center", padding: "32px", color: "var(--gray-500)" }}>{agents.length ? "Tidak ada anggota yang cocok." : "Belum ada anggota tim."}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {agentForm && (
        <div className="modal-backdrop">
          <div className="modal-card" style={{ width: "400px" }}>
            <div className="modal-header">
              <div className="modal-title">{agentForm.id ? "Edit Anggota" : "Tambah Anggota"}</div>
              <button type="button" className="icon-btn" onClick={() => setAgentForm(null)}>{Icons.close}</button>
            </div>
            <div className="modal-body" style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div>
                <label className="form-label">Nama Lengkap</label>
                <input className="form-input" value={agentForm.name} onChange={(e) => setAgentForm({ ...agentForm, name: e.target.value })} placeholder="Budi Santoso" />
              </div>
              <div>
                <label className="form-label">Username</label>
                <input className="form-input" value={agentForm.username} onChange={(e) => setAgentForm({ ...agentForm, username: e.target.value })} placeholder="budi.s" disabled={!!agentForm.id} />
              </div>
              <div>
                <label className="form-label">Role Akses</label>
                <select className="form-select" value={agentForm.role} onChange={(e) => setAgentForm({ ...agentForm, role: e.target.value })}>
                  <option value="operator">Agent (Akses Inbox & Chat)</option>
                  <option value="admin">Admin (Akses Setting & Tim)</option>
                  <option value="super_admin">Super Admin (Akses Penuh)</option>
                </select>
              </div>
              {agentForm.id ? (
                <label className="form-checkbox-row">
                  <input type="checkbox" checked={agentForm.isActive !== false} onChange={(e) => setAgentForm({ ...agentForm, isActive: e.target.checked })} />
                  <span>
                    <strong>{agentForm.isActive !== false ? "Akun aktif" : "Akun nonaktif"}</strong>
                    <small>{agentForm.isActive !== false ? "User masih bisa login dan menerima assignment." : "User tidak bisa dipakai untuk akses operasional."}</small>
                  </span>
                </label>
              ) : null}
              {!agentForm.id && (
                <div>
                  <label className="form-label">Password Sementara</label>
                  <input type="password" className="form-input" value={agentForm.password || ""} onChange={(e) => setAgentForm({ ...agentForm, password: e.target.value })} placeholder="Buatkan password..." />
                </div>
              )}
            </div>
            <div className="modal-footer" style={{ marginTop: "24px", display: "flex", justifyContent: "flex-end", gap: "12px" }}>
              <button type="button" className="btn btn-secondary" onClick={() => setAgentForm(null)}>Batal</button>
              <button type="button" className="btn btn-primary" onClick={submitAgent} disabled={!agentForm.name || !agentForm.username || busyKey === "agent"}>Simpan</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
