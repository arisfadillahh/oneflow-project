import { Fragment, useEffect, useRef, useState } from "react";
import { Icons, dashboardBadge, formatDateTime, toRoleLabel } from "../../../lib/dashboard-core";
import { navigationLabel, t } from "../../../lib/i18n";

export function LoginScreen({ error, loading, onLogin, onRegister, theme = "light" }) {
  const [credentials, setCredentials] = useState({ username: "", password: "" });
  const [mode, setMode] = useState("login");
  const [showPassword, setShowPassword] = useState(false);
  const [registration, setRegistration] = useState({
    name: "",
    username: "",
    password: "",
    phone: "",
    organizationName: "",
    organizationSlug: "",
  });

  function submitLogin(event) {
    event.preventDefault();
    onLogin(credentials.username, credentials.password);
  }

  function submitRegister(event) {
    event.preventDefault();
    onRegister?.(registration);
  }

  return (
    <main className={`login-shell theme-${theme}`} data-theme={theme}>
      <section className="left-panel">
        <div className="deco deco-1" />
        <div className="deco deco-2" />
        <div className="left-logo">
          <img className="login-brand-logo" src="/brand/oneflow-main-logo.png" alt="oneflow.id" />
        </div>

        <div className="left-body">
          <h1 className="headline">AI Chat Automation<br /><span>untuk Operasional Harian</span></h1>
          <p className="tagline">Platform AI yang membantu bisnis membalas pelanggan, mencatat pesanan, dan menjaga follow-up tetap rapi.</p>
          <div className="feat-list">
            {[
              "Sigap merespons kebutuhan dan perubahan bisnis",
              "Ramah untuk pelanggan, tim, dan operasional bisnis",
              "Rapi mengelola knowledge, inbox, dan eskalasi",
              "Profesional dengan warna, struktur, dan workflow konsisten",
            ].map((item) => (
              <div className="feat-item" key={item}>
                <div className="feat-check">{Icons.check}</div>
                <span className="feat-text">{item}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="left-bottom">
          <div className="stats-row">
            <div><div className="stat-value">24/7</div><div className="stat-label">Balasan AI</div></div>
            <div><div className="stat-value">98%</div><div className="stat-label">Response Rate</div></div>
            <div><div className="stat-value">3x</div><div className="stat-label">Follow-up Rapi</div></div>
          </div>
        </div>
      </section>

      <section className="right-panel">
        <div className="login-box">
          <img className="login-box-logo" src="/brand/oneflow-main-logo.png" alt="oneflow.id" />
          <h2 className="greeting">{mode === "login" ? "Selamat datang" : "Buat akun bisnis"}</h2>
          <p className="sub">{mode === "login" ? "Masuk ke dashboard Oneflow.id Anda" : "Daftar bisnis Anda, lalu mulai dari setup AI CS"}</p>

          <div className="auth-switch">
            <button type="button" className={mode === "login" ? "active" : ""} onClick={() => setMode("login")}>Login</button>
            <button type="button" className={mode === "register" ? "active" : ""} onClick={() => setMode("register")}>Daftar</button>
          </div>

          <div className={`error-msg ${error ? "show" : ""}`}>{error || "Username atau password salah. Silakan coba lagi."}</div>

          {mode === "login" ? (
          <form onSubmit={submitLogin}>
            <div className="form-group">
              <label className="form-label" htmlFor="login-username">Username</label>
              <div className="input-wrap">
                <span className="input-icon">{Icons.team}</span>
                <input
                  className="form-input"
                  id="login-username"
                  type="text"
                  autoComplete="username"
                  required
                  value={credentials.username}
                  onChange={(event) => setCredentials((current) => ({ ...current, username: event.target.value }))}
                  placeholder="Masukkan username"
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="login-password">
                Password <span className="forgot">Lupa password?</span>
              </label>
              <div className="input-wrap">
                <span className="input-icon">{Icons.lock}</span>
                <input
                  className="form-input"
                  id="login-password"
                  autoComplete="current-password"
                  type={showPassword ? "text" : "password"}
                  required
                  value={credentials.password}
                  onChange={(event) => setCredentials((current) => ({ ...current, password: event.target.value }))}
                  placeholder="********"
                />
                <button className="show-pwd" type="button" aria-label="Toggle password visibility" onClick={() => setShowPassword((current) => !current)}>
                  <svg viewBox="0 0 24 24" fill="none" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
                </button>
              </div>
            </div>

            <button className="btn-login" type="submit" disabled={loading}>
              <svg viewBox="0 0 24 24" fill="none" strokeWidth="2"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" /><polyline points="10 17 15 12 10 7" /><line x1="15" y1="12" x2="3" y2="12" /></svg>
              {loading ? "Memeriksa akses..." : "Masuk ke Dashboard"}
            </button>
          </form>
          ) : (
          <form onSubmit={submitRegister}>
            <div className="form-group">
              <label className="form-label" htmlFor="register-org">Nama bisnis</label>
              <input className="form-input" id="register-org" required value={registration.organizationName} onChange={(event) => setRegistration((current) => ({ ...current, organizationName: event.target.value }))} placeholder="Kedai Kopi Sari" />
            </div>
            <div className="form-group">
              <label className="form-label" htmlFor="register-name">Nama pengguna utama</label>
              <input className="form-input" id="register-name" required value={registration.name} onChange={(event) => setRegistration((current) => ({ ...current, name: event.target.value }))} placeholder="Nama lengkap" />
            </div>
            <div className="form-group">
              <label className="form-label" htmlFor="register-username">Username</label>
              <input className="form-input" id="register-username" required autoComplete="username" value={registration.username} onChange={(event) => setRegistration((current) => ({ ...current, username: event.target.value }))} placeholder="admin.kedai" />
            </div>
            <div className="form-group">
              <label className="form-label" htmlFor="register-password">Password</label>
              <input className="form-input" id="register-password" required minLength={8} autoComplete="new-password" type="password" value={registration.password} onChange={(event) => setRegistration((current) => ({ ...current, password: event.target.value }))} placeholder="Minimal 8 karakter" />
            </div>
            <button className="btn-login" type="submit" disabled={loading}>
              {loading ? "Mendaftarkan..." : "Mulai Gratis"}
            </button>
          </form>
          )}

          <div className="login-footer">
            &copy; 2026 Oneflow.id &middot; <a href="/privacy">Privacy</a> &middot; <a href="/terms">Terms</a>
          </div>
        </div>
      </section>
    </main>
  );
}

const mobilePrimaryNavByRole = {
  owner: ["overview", "wallet", "usage", "health"],
  super_admin: ["operations", "contacts", "agents", "playground"],
  admin: ["operations", "contacts", "agents", "playground"],
  operator: ["operations", "contacts", "agents", "playground"],
};

const mobileAccountNavItem = { id: "account", label: "Akun", icon: "account", mobileOnly: true };

export function Sidebar({ role, navItems, activeView, onNavigate, summary, inbox, purchases, health, wallet, waConnected, busyKey, onReportIssue, locale = "id" }) {
  const navRef = useRef(null);
  const [showMobileMore, setShowMobileMore] = useState(false);
  const isOwner = role === "owner";
  const monthlyLimit = Number(wallet?.monthlyCreditLimit || 0);
  const monthlyUsed = Number(wallet?.monthlyCreditsUsed || 0);
  const monthlyRemaining = Number(wallet?.monthlyCreditsRemaining || 0);
  const usagePercent = monthlyLimit > 0 ? Math.min(100, Math.round((monthlyUsed / monthlyLimit) * 100)) : 0;
  const nextResetAt = wallet?.nextResetAt ? `Reset ${formatDateTime(wallet.nextResetAt)}` : "Menunggu data wallet";
  const navSectionOrder = [
    { key: "primary", label: t(locale, "nav.primary") },
    { key: "ops", label: t(locale, "nav.operations") },
    { key: "more", label: t(locale, "nav.more") },
    { key: "tools", label: t(locale, "nav.tools") },
  ];
  const navGroupOf = (item) => (["ops", "more", "tools"].includes(item.group) ? item.group : "primary");
  const navSections = navSectionOrder
    .map((section) => ({ ...section, items: navItems.filter((item) => navGroupOf(item) === section.key) }))
    .filter((section) => section.items.length);
  const mobileAllNavItems = navItems.some((item) => item.id === "account") ? navItems : [...navItems, mobileAccountNavItem];
  const mobilePrimaryIds = mobilePrimaryNavByRole[role] || mobilePrimaryNavByRole.operator;
  const mobilePrimaryItems = mobilePrimaryIds
    .map((id) => mobileAllNavItems.find((item) => item.id === id))
    .filter(Boolean);
  const mobilePrimaryIdSet = new Set(mobilePrimaryItems.map((item) => item.id));
  const mobileOverflowItems = mobileAllNavItems.filter((item) => !mobilePrimaryIdSet.has(item.id));
  const mobileOverflowBadge = mobileOverflowItems.reduce((total, item) => total + dashboardBadge(item.id, { summary, inbox, purchases, health }), 0);
  const mobileMoreActive = mobileOverflowItems.some((item) => activeView === item.id);
  const mobileNavLabel = (item) => navigationLabel(locale, item.id, item.label);
  const mobileMoreHint = (item) => {
    const shortLabel = mobileNavLabel(item);
    if (item.mobileOnly) return t(locale, "account.settings");
    if (item.group === "tools") return locale === "en" ? "Business tool" : "Alat bisnis";
    if (["upgrade", "wallet"].includes(item.id)) return locale === "en" ? "Plans and credits" : "Paket & kredit";
    if (navigationLabel(locale, item.id, item.label) !== shortLabel) return navigationLabel(locale, item.id, item.label);
    return locale === "en" ? "Dashboard feature" : "Fitur dashboard";
  };
  const renderNavItem = (item, options = {}) => {
    const badge = dashboardBadge(item.id, { summary, inbox, purchases, health });
    return (
      <button
        key={item.id}
        className={`nav-item ${item.parentId ? "child" : ""} ${activeView === item.id ? "active" : ""}`}
        onClick={() => {
          onNavigate(item.id);
          if (options.closeMobileMore) setShowMobileMore(false);
        }}
        title={navigationLabel(locale, item.id, item.label)}
        data-tour={`nav-${item.id}`}
      >
        {Icons[item.icon]}
        <span className="nav-label-full">{navigationLabel(locale, item.id, item.label)}</span>
        <span className="nav-label-mobile" aria-hidden="true">{mobileNavLabel(item)}</span>
        {badge > 0 ? <span className="nav-badge">{badge}</span> : null}
      </button>
    );
  };
  const renderMobileMoreItem = (item) => {
    const badge = dashboardBadge(item.id, { summary, inbox, purchases, health });
    return (
      <button
        key={item.id}
        type="button"
        className={`mobile-more-item ${activeView === item.id ? "active" : ""}`}
        onClick={() => {
          onNavigate(item.id);
          setShowMobileMore(false);
        }}
        role="menuitem"
      >
        <span className="mobile-more-item-icon">{Icons[item.icon]}</span>
        <span className="mobile-more-item-copy">
          <strong>{mobileNavLabel(item)}</strong>
          <small>{mobileMoreHint(item)}</small>
        </span>
        {badge > 0 ? <span className="nav-badge">{badge}</span> : null}
      </button>
    );
  };

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia("(max-width: 760px)").matches) return;
    const activeButton = navRef.current?.querySelector(`.mobile-nav-list [data-tour="nav-${activeView}"]`);
    activeButton?.scrollIntoView({ inline: "center", block: "nearest", behavior: "smooth" });
  }, [activeView, navItems]);

  useEffect(() => {
    setShowMobileMore(false);
  }, [activeView]);

  return (
    <aside className={`sidebar ${showMobileMore ? "mobile-more-open" : ""}`}>
      <div className="sidebar-header">
        <div className="sidebar-logo sidebar-logo-full">
          <img src="/brand/oneflow-main-logo.png" alt="oneflow.id" />
        </div>
      </div>

      <nav className="sidebar-nav" ref={navRef}>
        <div className="desktop-nav-list">
          {navSections.map((section, index) => (
            <Fragment key={section.key}>
              <div className={`nav-section-label ${index > 0 ? "nav-section-tools" : ""}`}>{section.label}</div>
              {section.items.map(renderNavItem)}
            </Fragment>
          ))}
        </div>

        <div className="mobile-nav-list" aria-label={locale === "en" ? "Quick mobile navigation" : "Navigasi cepat mobile"}>
          {mobilePrimaryItems.map(renderNavItem)}
          {mobileOverflowItems.length ? (
            <button
              type="button"
              className={`nav-item mobile-more-trigger ${mobileMoreActive || showMobileMore ? "active" : ""}`}
              onClick={() => setShowMobileMore((current) => !current)}
              aria-label={t(locale, "nav.moreFeatures")}
              aria-expanded={showMobileMore}
              title={t(locale, "nav.moreFeatures")}
            >
              {Icons.more}
              <span className="nav-label-full">{t(locale, "nav.more")}</span>
              <span className="nav-label-mobile" aria-hidden="true">{t(locale, "nav.more")}</span>
              {mobileOverflowBadge > 0 ? <span className="nav-badge">{mobileOverflowBadge}</span> : null}
            </button>
          ) : null}
        </div>
      </nav>

      {showMobileMore && mobileOverflowItems.length ? (
        <div className="mobile-more-menu" role="menu">
          <div className="mobile-more-menu-head">
            <span>{t(locale, "nav.moreFeatures")}</span>
            <button type="button" onClick={() => setShowMobileMore(false)} aria-label={t(locale, "nav.closeMore")}>
              {Icons.close}
            </button>
          </div>
          <div className="mobile-more-menu-grid">
            {mobileOverflowItems.map(renderMobileMoreItem)}
          </div>
        </div>
      ) : null}

      <div className="sidebar-footer">
        <button className="sidebar-support" type="button" onClick={onReportIssue} disabled={busyKey === "support-report"}>
          <div className="support-icon">{Icons.help}</div>
          <div className="support-text">
            <div className="support-title">{t(locale, "nav.support")}</div>
            <div className="support-subtitle">{busyKey === "support-report" ? t(locale, "nav.sendingReport") : t(locale, "nav.supportHint")}</div>
          </div>
          <div style={{ marginLeft: "auto" }}>{Icons.arrowRight}</div>
        </button>
      </div>
    </aside>
  );
}

export function Topbar({
  title,
  auth,
  theme = "light",
  notifications,
  unreadNotificationCount,
  notificationReadIds,
  onMarkNotificationsRead,
  onNavigate,
  onToggleTheme,
  logout,
  locale = "id",
}) {
  const [showNotifications, setShowNotifications] = useState(false);
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const visibleNotifications = notifications.slice(0, 8);
  const showUpgradeEntry = ["super_admin", "admin"].includes(auth?.user?.role);

  return (
    <header className="topbar">
      <div>
        <div className="topbar-title">{title}</div>
      </div>
      <div className="topbar-search">
        {Icons.search}
        <input type="text" placeholder={t(locale, "top.search")} aria-label={t(locale, "top.search")} />
      </div>

      <div className="topbar-right">
        {showUpgradeEntry ? (
          <button type="button" className="topbar-upgrade-pill" onClick={() => onNavigate("upgrade")}>
            {Icons.wallet}
            <span>{t(locale, "top.billing")}</span>
          </button>
        ) : null}

        <div className="topbar-menu-wrap">
          <button
            type="button"
            className={`topbar-btn theme-toggle-btn ${theme === "dark" ? "is-dark" : ""}`}
            aria-label={theme === "dark" ? t(locale, "top.lightMode") : t(locale, "top.darkMode")}
            title={theme === "dark" ? t(locale, "top.lightMode") : t(locale, "top.darkMode")}
            onClick={onToggleTheme}
          >
            {theme === "dark" ? Icons.sun : Icons.moon}
          </button>
        </div>

        <div className="topbar-menu-wrap">
          <button
            type="button"
            className="topbar-btn"
            aria-label={t(locale, "top.notifications")}
            onClick={() => {
              setShowNotifications((current) => !current);
              setShowProfileMenu(false);
            }}
          >
            {Icons.bell}
            {unreadNotificationCount > 0 ? <span className="notification-badge">{unreadNotificationCount}</span> : null}
          </button>
          {showNotifications ? (
            <div className="topbar-dropdown notifications-dropdown">
              <div className="dropdown-header">
                <div>
                  <strong>{t(locale, "top.notifications")}</strong>
                  <span>{t(locale, "top.unread", { count: unreadNotificationCount })}</span>
                </div>
                {notifications.length ? (
                  <button type="button" onClick={() => onMarkNotificationsRead()}>{t(locale, "top.markRead")}</button>
                ) : null}
              </div>
              <div className="notification-list">
                {visibleNotifications.length ? visibleNotifications.map((item) => {
                  const unread = !notificationReadIds.includes(item.id);
                  return (
                    <button
                      type="button"
                      className={`notification-item ${unread ? "is-unread" : ""}`}
                      key={item.id}
                      onClick={() => {
                        onMarkNotificationsRead([item.id]);
                        if (item.actionView) onNavigate(item.actionView);
                        setShowNotifications(false);
                      }}
                    >
                      <span className={`notification-dot ${item.severity || "info"}`} />
                      <span>
                        <strong>{item.title}</strong>
                        <small>{item.message}</small>
                      </span>
                    </button>
                  );
                }) : (
                  <div className="empty-dropdown">{locale === "en" ? "No active notifications." : "Tidak ada notifikasi aktif."}</div>
                )}
              </div>
            </div>
          ) : null}
        </div>

        <div className="company-selector" title={auth.user?.organizationName || "Bisnis"}>
          <span className="company-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M3 21h18" /><path d="M5 21V7l8-4v18" /><path d="M19 21V11l-6-4" /><path d="M9 9h1" /><path d="M9 13h1" /><path d="M9 17h1" /><path d="M15 13h1" /><path d="M15 17h1" /></svg>
          </span>
          <span className="company-name">{auth.user?.organizationName || "Bisnis"}</span>
        </div>

        <div className="topbar-menu-wrap topbar-refresh-wrap">
          <button type="button" className="topbar-btn" title="Refresh" onClick={() => window.location.reload()}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" /><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10" /><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14" /></svg>
          </button>
        </div>

        <div className="topbar-menu-wrap">
          <button
            type="button"
            className="topbar-btn topbar-profile-btn"
            title={auth.user.name || auth.user.username}
            aria-haspopup="menu"
            aria-expanded={showProfileMenu}
            onClick={() => {
              setShowProfileMenu((current) => !current);
              setShowNotifications(false);
            }}
          >
            <div className="user-avatar topbar-profile-avatar">
              <img src={`https://ui-avatars.com/api/?name=${auth.user.name || auth.user.username}&background=random`} alt={auth.user.name || auth.user.username} width="36" height="36" />
            </div>
            <div className="user-info topbar-profile-info">
              <div className="user-name">{auth.user.name || auth.user.username}</div>
              <div className="user-role">{toRoleLabel(auth.user.role)}</div>
            </div>
            <span className={`profile-chevron ${showProfileMenu ? "open" : ""}`} aria-hidden="true">{Icons.chevronDown}</span>
          </button>
          {showProfileMenu ? (
            <div className="topbar-dropdown profile-dropdown">
              <button
                type="button"
                onClick={() => {
                  onNavigate("account");
                  setShowProfileMenu(false);
                }}
              >
                {t(locale, "account.settings")}
              </button>
              <button type="button" className="danger-menu-item" onClick={logout}>
                {t(locale, "top.logout")}
              </button>
            </div>
          ) : null}
        </div>
      </div>
    </header>
  );
}
