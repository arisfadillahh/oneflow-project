import { Icons, toRoleLabel } from "../../../lib/dashboard-core";
import { supportedLocales, t } from "../../../lib/i18n";

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
  locale = "id",
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
          <div className="page-title">{t(locale, "account.title")}</div>
          <div className="page-desc">{t(locale, "account.description")}</div>
        </div>
      </div>

      <section className="account-hero">
        <div className="account-avatar-large">{initials(displayName)}</div>
        <div className="account-hero-copy">
          <h2>{displayName}</h2>
          <p>{user.username}</p>
          <div className="account-identity-meta">
            <span className="badge blue">{roleLabel}</span>
            <span className="badge gray">{t(locale, "account.settings")}</span>
          </div>
        </div>
      </section>

      <div className="account-settings-grid">
        <section className="account-settings-panel">
          <div className="account-panel-head">
            <span className="account-panel-icon">{Icons.team}</span>
            <div>
              <h3>{t(locale, "account.profile")}</h3>
              <p>{t(locale, "account.profileHelp")}</p>
            </div>
          </div>

          <form className="account-form" onSubmit={saveAccountProfile}>
            <label className="form-group">
              <span className="form-label">{t(locale, "account.displayName")}</span>
              <input
                className="form-input"
                value={accountProfileForm.name}
                onChange={(event) => setAccountProfileForm((current) => ({ ...current, name: event.target.value }))}
                placeholder={t(locale, "account.fullName")}
                autoComplete="name"
              />
            </label>

            <label className="form-group">
              <span className="form-label">{t(locale, "account.language")}</span>
              <select
                className="form-input"
                value={accountProfileForm.preferredLocale || "id"}
                onChange={(event) => setAccountProfileForm((current) => ({ ...current, preferredLocale: event.target.value }))}
              >
                {supportedLocales.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
              </select>
              <span className="form-help">{t(locale, "account.languageHelp")}</span>
            </label>

            <label className="form-group">
              <span className="form-label">Username</span>
              <input className="form-input" value={user.username || ""} disabled />
            </label>

            <div className="account-form-actions">
              <button className="btn btn-primary" type="submit" disabled={profileBusy || !accountProfileForm.name?.trim()}>
                {profileBusy ? t(locale, "account.saving") : t(locale, "account.save")}
              </button>
            </div>
          </form>
        </section>

        <section className="account-settings-panel">
          <div className="account-panel-head">
            <span className="account-panel-icon">{Icons.lock}</span>
            <div>
              <h3>{t(locale, "account.security")}</h3>
              <p>{t(locale, "account.securityHelp")}</p>
            </div>
          </div>

          <form className="account-form" onSubmit={changeAccountPassword}>
            <label className="form-group">
              <span className="form-label">{t(locale, "account.currentPassword")}</span>
              <input
                className="form-input"
                type="password"
                value={accountPasswordForm.currentPassword}
                onChange={(event) => setAccountPasswordForm((current) => ({ ...current, currentPassword: event.target.value }))}
                autoComplete="current-password"
                placeholder={t(locale, "account.passwordPlaceholder")}
              />
            </label>

            <div className="account-password-row">
              <label className="form-group">
                <span className="form-label">{t(locale, "account.newPassword")}</span>
                <input
                  className="form-input"
                  type="password"
                  value={accountPasswordForm.newPassword}
                  onChange={(event) => setAccountPasswordForm((current) => ({ ...current, newPassword: event.target.value }))}
                  autoComplete="new-password"
                  placeholder={t(locale, "account.minimum")}
                />
              </label>

              <label className="form-group">
                <span className="form-label">{t(locale, "account.repeatPassword")}</span>
                <input
                  className="form-input"
                  type="password"
                  value={accountPasswordForm.confirmPassword}
                  onChange={(event) => setAccountPasswordForm((current) => ({ ...current, confirmPassword: event.target.value }))}
                  autoComplete="new-password"
                  placeholder={t(locale, "account.match")}
                />
              </label>
            </div>

            <div className="account-form-actions">
              <button className="btn btn-secondary" type="submit" disabled={passwordBusy}>
                {passwordBusy ? t(locale, "account.updating") : t(locale, "account.updatePassword")}
              </button>
            </div>
          </form>
        </section>
      </div>
    </div>
  );
}
