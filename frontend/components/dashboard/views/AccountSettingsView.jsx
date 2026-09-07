import { Icons, toRoleLabel } from "../../../lib/dashboard-core";

function initials(name = "") {
  return String(name || "?")
    .split(" ")
    .filter(Boolean)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

export function AccountSettingsView({
  auth,
  accountProfileForm,
  setAccountProfileForm,
  accountPasswordForm,
  setAccountPasswordForm,
  saveAccountProfile,
  changeAccountPassword,
  busyKey,
}) {
  const user = auth?.user || {};
  const displayName = user.name || user.username || "User";
  const roleLabel = toRoleLabel(user.role);
  const profileBusy = busyKey === "account-profile";
  const passwordBusy = busyKey === "account-password";

  return (
    <div className="account-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Akun</div>
          <div className="page-desc">Kelola nama akun dan kata sandi.</div>
        </div>
      </div>

      <section className="account-hero">
        <div className="account-avatar-large">{initials(displayName)}</div>
        <div className="account-hero-copy">
          <h2>{displayName}</h2>
          <p>{user.username}</p>
          <div className="account-identity-meta">
            <span className="badge blue">{roleLabel}</span>
            <span className="badge gray">Account settings</span>
          </div>
        </div>
      </section>

      <div className="account-settings-grid">
        <section className="account-settings-panel">
          <div className="account-panel-head">
            <span className="account-panel-icon">{Icons.team}</span>
            <div>
              <h3>Profil Akun</h3>
              <p>Nama ini muncul di dashboard, assignment, dan audit internal.</p>
            </div>
          </div>

          <form className="account-form" onSubmit={saveAccountProfile}>
            <label className="form-group">
              <span className="form-label">Nama tampilan</span>
              <input
                className="form-input"
                value={accountProfileForm.name}
                onChange={(event) => setAccountProfileForm((current) => ({ ...current, name: event.target.value }))}
                placeholder="Nama lengkap"
                autoComplete="name"
              />
            </label>

            <label className="form-group">
              <span className="form-label">Username</span>
              <input className="form-input" value={user.username || ""} disabled />
            </label>

            <div className="account-form-actions">
              <button className="btn btn-primary" type="submit" disabled={profileBusy || !accountProfileForm.name?.trim()}>
                {profileBusy ? "Menyimpan..." : "Simpan profil"}
              </button>
            </div>
          </form>
        </section>

        <section className="account-settings-panel">
          <div className="account-panel-head">
            <span className="account-panel-icon">{Icons.lock}</span>
            <div>
              <h3>Keamanan Login</h3>
              <p>Gunakan password minimal 8 karakter dan jangan pakai ulang password lama.</p>
            </div>
          </div>

          <form className="account-form" onSubmit={changeAccountPassword}>
            <label className="form-group">
              <span className="form-label">Password saat ini</span>
              <input
                className="form-input"
                type="password"
                value={accountPasswordForm.currentPassword}
                onChange={(event) => setAccountPasswordForm((current) => ({ ...current, currentPassword: event.target.value }))}
                autoComplete="current-password"
                placeholder="Masukkan password saat ini"
              />
            </label>

            <div className="account-password-row">
              <label className="form-group">
                <span className="form-label">Password baru</span>
                <input
                  className="form-input"
                  type="password"
                  value={accountPasswordForm.newPassword}
                  onChange={(event) => setAccountPasswordForm((current) => ({ ...current, newPassword: event.target.value }))}
                  autoComplete="new-password"
                  placeholder="Minimal 8 karakter"
                />
              </label>

              <label className="form-group">
                <span className="form-label">Ulangi password baru</span>
                <input
                  className="form-input"
                  type="password"
                  value={accountPasswordForm.confirmPassword}
                  onChange={(event) => setAccountPasswordForm((current) => ({ ...current, confirmPassword: event.target.value }))}
                  autoComplete="new-password"
                  placeholder="Samakan dengan password baru"
                />
              </label>
            </div>

            <div className="account-form-actions">
              <button className="btn btn-secondary" type="submit" disabled={passwordBusy}>
                {passwordBusy ? "Mengupdate..." : "Update password"}
              </button>
            </div>
          </form>
        </section>
      </div>
    </div>
  );
}
