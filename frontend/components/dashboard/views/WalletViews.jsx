import { useState } from "react";
import { Icons, formatCurrencyIDR, formatDateTime, formatNumber } from "../../../lib/dashboard-core";
import { StatusPill } from "../ui";

const PLAN_ORDER = { starter: 1, growth: 2, business: 3 };
const POPULAR_PLAN_KEY = "growth";

function formatPublicPlanPrice(value) {
  return formatCurrencyIDR(value).replace(/\s+/g, "");
}
const PAYMENT_GATEWAY_VAT_RATE = 0.11;
const VIRTUAL_ACCOUNT_GATEWAY_FEE = 4000;
const QRIS_GATEWAY_FEE_RATE = 0.007;
const CARD_GATEWAY_FEE_RATE = 0.029;
const CARD_GATEWAY_FLAT_FEE = 2000;
const PAYMENT_METHODS = [
  { id: "manual_transfer", label: "Virtual Account", description: "BCA, BNI, BRI, Permata, dan Mandiri Bill.", feeHint: "+ Rp4.440", icon: "wallet", badge: "Rekomendasi" },
  { id: "qris", label: "QRIS", description: "Bayar lewat e-wallet atau mobile banking.", feeHint: "+ 0,7%", icon: "spark" },
  { id: "card", label: "Kartu", description: "Belum tersedia untuk checkout saat ini.", icon: "purchase", badge: "Segera", disabled: true },
];

function availablePaymentMethod(method) {
  const option = PAYMENT_METHODS.find((item) => item.id === method);
  return option && !option.disabled ? option.id : "manual_transfer";
}

function numberValue(value) {
  return Number(value ?? 0) || 0;
}

function ceilIDR(value) {
  return Math.max(0, Math.ceil(Number(value) || 0));
}

function flatFeeWithVAT(amount) {
  return ceilIDR(amount * (1 + PAYMENT_GATEWAY_VAT_RATE));
}

function paymentQuote(item, method) {
  const packageAmount = ceilIDR(item?.price);
  const normalizedMethod = availablePaymentMethod(method);
  let paymentFee = flatFeeWithVAT(VIRTUAL_ACCOUNT_GATEWAY_FEE);
	let feeLabel = "Biaya pembayaran Virtual Account";
  if (normalizedMethod === "qris") {
    const grossAmount = ceilIDR(packageAmount / (1 - QRIS_GATEWAY_FEE_RATE));
    paymentFee = grossAmount - packageAmount;
		feeLabel = "Biaya pembayaran QRIS";
  } else if (normalizedMethod === "card") {
    const flatFee = flatFeeWithVAT(CARD_GATEWAY_FLAT_FEE);
    const percentWithVAT = CARD_GATEWAY_FEE_RATE * (1 + PAYMENT_GATEWAY_VAT_RATE);
    const grossAmount = ceilIDR((packageAmount + flatFee) / (1 - percentWithVAT));
    paymentFee = grossAmount - packageAmount;
		feeLabel = "Biaya pembayaran kartu";
  }
  return {
    packageAmount,
    paymentFee: Math.max(0, paymentFee),
    grossAmount: packageAmount + Math.max(0, paymentFee),
    feeLabel,
  };
}

function purchasePaymentFee(item) {
  return numberValue(item?.paymentFee ?? (numberValue(item?.grossAmount) - numberValue(item?.price)));
}

function purchaseGrossAmount(item) {
  return numberValue(item?.grossAmount) || numberValue(item?.price) + purchasePaymentFee(item);
}

function walletValue(wallet, primaryKey, fallbackKey) {
  return numberValue(wallet?.[primaryKey] ?? wallet?.[fallbackKey]);
}

function percentValue(used, limit) {
  if (!limit) return 0;
  return Math.min(100, Math.round((used / limit) * 100));
}

function packageKey(item) {
  return String(item?.planKey || item?.plan_key || "").toLowerCase();
}

function packageIsPopular(item) {
  if (!item) return false;
  if (Object.prototype.hasOwnProperty.call(item, "isPopular")) return item.isPopular === true;
  if (Object.prototype.hasOwnProperty.call(item, "is_popular")) return item.is_popular === true;
  return packageKey(item) === POPULAR_PLAN_KEY;
}

function planRankFromKey(key) {
  return PLAN_ORDER[String(key || "").toLowerCase()] || 0;
}

function packageRank(item) {
  return planRankFromKey(packageKey(item));
}

function packagePeriod(item) {
  return String(item?.billingPeriod || item?.billing_period || "monthly").toLowerCase();
}

function annualDiscountValue(item) {
  return Math.max(0, Math.min(99.99, numberValue(item?.annualDiscountPercent ?? 10)));
}

function planForCheckoutPeriod(item, period) {
  if (!item || period !== "annual" || isAddOnPackage(item)) return { ...item, billingPeriod: packagePeriod(item) };
  const monthlyPrice = numberValue(item.price);
  const annualDiscountPercent = annualDiscountValue(item);
  return {
    ...item,
    billingPeriod: "annual",
    monthlyPrice,
    annualDiscountPercent,
    price: Math.round(monthlyPrice * 12 * (1 - annualDiscountPercent / 100)),
  };
}

function activePackages(packages) {
  return (packages || []).filter((item) => item.isActive !== false);
}

function isAddOnPackage(item) {
  const key = packageKey(item);
  return packagePeriod(item) === "one_time" || key.startsWith("addon_");
}

function isPlanPackage(item) {
  return packagePeriod(item) === "monthly" && !isAddOnPackage(item);
}

function isOpenPurchase(item) {
  return ["requested", "pending"].includes(String(item?.paymentStatus || ""));
}

function canContinuePayment(item) {
  return isOpenPurchase(item) && Boolean(item?.snapRedirectUrl);
}

function isConfirmedPurchase(item) {
  return String(item?.paymentStatus || "") === "confirmed";
}

function invoiceHref(item) {
  return item?.id ? `/dashboard/invoices/${encodeURIComponent(item.id)}` : "";
}

function latestOpenPurchase(purchases) {
  return [...(purchases || [])]
    .filter(isOpenPurchase)
    .sort((a, b) => new Date(b.createdAt || 0).getTime() - new Date(a.createdAt || 0).getTime())[0] || null;
}

function paymentMethodLabel(method) {
  const option = PAYMENT_METHODS.find((item) => item.id === method);
  return option?.label || "Virtual Account";
}

function openPurchaseForPackage(purchases, item) {
  if (!item) return null;
  const itemKey = packageKey(item);
  const itemPeriod = packagePeriod(item);
  return (purchases || []).find((purchase) => {
    if (!isOpenPurchase(purchase)) return false;
    const purchasePeriod = String(purchase?.billingPeriod || "monthly").toLowerCase();
    if (purchasePeriod !== itemPeriod) return false;
    if (purchase.packageId && purchase.packageId === item.id) return true;
    return itemKey && packageKey(purchase) === itemKey;
  }) || null;
}

function sortPlanPackages(items) {
  return [...items].sort((a, b) => {
    const rankA = packageRank(a) || 99;
    const rankB = packageRank(b) || 99;
    if (rankA !== rankB) return rankA - rankB;
    return numberValue(a.price) - numberValue(b.price);
  });
}

function availablePlanPackages(items, currentPlanKey, checkoutPeriod) {
  const currentRank = planRankFromKey(currentPlanKey);
  return sortPlanPackages(items).filter((item) => {
    const rank = packageRank(item);
    return !currentRank || !rank || rank > currentRank || (checkoutPeriod === "annual" && rank === currentRank);
  });
}

function packageLimits(item) {
  return {
    whatsapp: numberValue(item?.maxWhatsAppSessions ?? item?.limits?.whatsapp),
    aiAgents: numberValue(item?.maxAiAgents ?? item?.limits?.aiAgents),
    humanUsers: numberValue(item?.maxHumanUsers ?? item?.limits?.humanUsers),
  };
}

function packageDescription(item) {
  const key = packageKey(item);
  if (item?.description) return item.description;
  if (key === "starter") return "Untuk bisnis kecil yang mulai memakai AI CS.";
  if (key === "growth") return "Untuk tim operasional aktif dengan beberapa agent.";
  if (key === "business") return "Untuk multi-brand atau volume chat tinggi.";
  if (isAddOnPackage(item)) return "Kredit ekstra sekali beli tanpa mengubah limit paket.";
  return "Paket custom untuk kebutuhan khusus.";
}

function planDisplayName(item) {
  const configuredName = String(item?.name || "").trim();
  if (configuredName) return configuredName;
  const key = packageKey(item);
  if (key === "starter") return "Starter";
  if (key === "growth") return "Growth";
  if (key === "business") return "Business";
  return "Custom";
}

function planShortDescription(item) {
  const key = packageKey(item);
  if (key === "starter") return "Mulai operasional produksi untuk bisnis kecil dengan satu nomor utama.";
  if (key === "growth") return "Aktifkan operasional penuh untuk beberapa agent dan tim CS.";
  if (key === "business") return "Tingkatkan kapasitas tim untuk multi-brand dan volume chat tinggi.";
  return packageDescription(item);
}

function planBenefits(item) {
  const key = packageKey(item);
  if (key === "starter") {
    return [
      "AI customer service untuk kebutuhan dasar",
      "WhatsApp produksi untuk 1 nomor",
      "Knowledge base dan Playground",
      "Top-up kredit setelah paket aktif",
    ];
  }
  if (key === "growth") {
    return [
      "Routing lanjutan untuk beberapa agent",
      "2 nomor WhatsApp produksi",
      "Analitik dan pelacakan pemakaian",
      "Top-up add-on tersedia melalui Billing",
      "Dukungan tim hingga 10 user",
    ];
  }
  if (key === "business") {
    return [
      "5 nomor WhatsApp produksi",
      "15 AI agent untuk berbagai alur bisnis",
      "25 user untuk tim operasional besar",
      "Kredit bulanan naik signifikan dari Growth",
      "Kapasitas lebih aman untuk campaign dan periode ramai",
    ];
  }
  return ["Paket custom sesuai konfigurasi owner", "Kredit bulanan dan limit mengikuti katalog aktif"];
}

function WalletMetric({ label, value, meta, tone = "blue" }) {
  return (
    <div className="wallet-metric">
      <div className={`wallet-metric-dot ${tone}`} />
      <div>
        <div className="wallet-metric-label">{label}</div>
        <div className="wallet-metric-value">{value}</div>
        <div className="wallet-metric-meta">{meta}</div>
      </div>
    </div>
  );
}

function PurchaseHistory({ purchases, cancelPurchase, busyKey, readOnly = false }) {
  const [cancelTarget, setCancelTarget] = useState(null);
  const requested = purchases.filter(isOpenPurchase).length;
  const confirmed = purchases.filter(isConfirmedPurchase).length;
  const hasActions = !readOnly && (purchases.some(isOpenPurchase) || purchases.some(isConfirmedPurchase));
  const confirmCancel = async () => {
    if (!cancelTarget) return;
    const target = cancelTarget;
    await cancelPurchase?.(target);
    setCancelTarget(null);
  };

  return (
    <div className="wallet-panel wallet-table-panel">
      <div className="wallet-panel-header">
        <div>
          <div className="wallet-panel-title">Riwayat Pembelian</div>
          <div className="wallet-panel-subtitle">{requested} pembayaran berjalan, {confirmed} pembelian berhasil.</div>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Paket</th>
              <th>Tipe</th>
              <th>Jumlah</th>
              <th>Status</th>
              <th>Diajukan</th>
              {hasActions && <th>Aksi</th>}
            </tr>
          </thead>
          <tbody>
            {purchases.map((item) => {
              const paymentFee = purchasePaymentFee(item);
              const grossAmount = purchaseGrossAmount(item);
              return (
              <tr key={item.id}>
                <td>
                  <strong>{item.packageName}</strong>
                  {item.organization && <div className="text-xs text-muted">{item.organization}</div>}
                  <div className="text-xs text-muted">Oleh {item.requestedBy || "Sistem"}</div>
                </td>
                <td>{item.billingPeriod === "one_time" ? "Top-up add-on" : "Paket bulanan"}</td>
                <td>
                  <strong>{formatNumber(item.creditAmount)} Cr</strong>
                  <div className="text-xs text-muted">Paket {formatCurrencyIDR(item.price)}</div>
                  {paymentFee > 0 ? <div className="text-xs text-muted">Biaya pembayaran {formatCurrencyIDR(paymentFee)}</div> : null}
                  {paymentFee > 0 ? <div className="text-xs text-muted">Total {formatCurrencyIDR(grossAmount)}</div> : null}
                </td>
                <td>
                  <StatusPill value={item.paymentStatus} kind="payment" />
                  {item.activeUntil ? <div className="text-xs text-muted">Aktif sampai {formatDateTime(item.activeUntil)}</div> : null}
                </td>
                <td className="text-sm text-muted">{formatDateTime(item.createdAt)}</td>
                {hasActions && (
                  <td>
                    <div className="table-actions">
                      {isOpenPurchase(item) ? (
                        <>
                          {canContinuePayment(item) ? (
                            <a className="btn btn-primary btn-sm" href={item.snapRedirectUrl} target="_blank" rel="noreferrer">Bayar</a>
                          ) : (
                            <button className="btn btn-secondary btn-sm" type="button" disabled>Bayar</button>
                          )}
                          {cancelPurchase ? (
                            <button className="btn btn-danger btn-sm" type="button" onClick={() => setCancelTarget(item)} disabled={busyKey === `purchase-cancel-${item.id}`}>
                              {busyKey === `purchase-cancel-${item.id}` ? "Membatalkan..." : "Batal"}
                            </button>
                          ) : null}
                        </>
                      ) : null}
                      {isConfirmedPurchase(item) ? (
                        <a className="btn btn-secondary btn-sm" href={invoiceHref(item)}>Invoice</a>
                      ) : null}
                      {!isOpenPurchase(item) && !isConfirmedPurchase(item) ? (
                        <span className="text-xs text-muted">-</span>
                      ) : null}
                    </div>
                  </td>
                )}
              </tr>
              );
            })}
            {!purchases.length && (
              <tr>
                <td colSpan={hasActions ? 6 : 5} className="wallet-empty-cell">Belum ada pembelian.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {cancelTarget ? (
        <div
          className="modal-backdrop billing-modal-backdrop purchase-cancel-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setCancelTarget(null);
          }}
        >
          <div className="billing-modal-card purchase-cancel-modal" role="dialog" aria-modal="true" aria-labelledby="purchase-cancel-title" onMouseDown={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <div>
                <h3 id="purchase-cancel-title">Batalkan pembayaran?</h3>
                <p>{cancelTarget.packageName || "Pembelian"} akan dikeluarkan dari pembayaran berjalan.</p>
              </div>
              <button type="button" className="modal-close" onClick={() => setCancelTarget(null)}>{Icons.close}</button>
            </div>
            <div className="billing-modal-body">
              <div className="purchase-cancel-summary">
                <span>Total pembayaran</span>
                <strong>{formatCurrencyIDR(purchaseGrossAmount(cancelTarget))}</strong>
                <small>{paymentMethodLabel(cancelTarget.paymentMethod)} - dibuat {formatDateTime(cancelTarget.createdAt)}</small>
              </div>
            </div>
            <div className="modal-actions">
              <button className="btn btn-secondary" type="button" onClick={() => setCancelTarget(null)}>Kembali</button>
              <button className="btn btn-danger" type="button" onClick={confirmCancel} disabled={busyKey === `purchase-cancel-${cancelTarget.id}`}>
                {busyKey === `purchase-cancel-${cancelTarget.id}` ? "Membatalkan..." : "Batalkan pembayaran"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function OwnerPackageCard({ item, type, canManage = false, onEdit, onDelete, busyKey = "" }) {
  const limits = packageLimits(item);
  const isTopUp = type === "topup";
  const isPopular = !isTopUp && packageIsPopular(item);
  return (
    <article className={`owner-package-card ${isTopUp ? "topup" : "service"} ${isPopular ? "popular" : ""}`}>
      <div className="owner-package-card-head">
        <div className="owner-package-card-badges">
          <span className={`badge ${isTopUp ? "purple" : "blue"}`}>{isTopUp ? "Top-up credit" : "Paket layanan"}</span>
          {isPopular ? <span className="badge purple">{Icons.spark} Populer</span> : null}
        </div>
        <div className="owner-package-card-tools">
          <StatusPill value={item.isActive ? "active" : "inactive"} kind="status" />
          {canManage ? (
            <div className="owner-package-actions" aria-label={`Aksi ${item.name}`}>
              <button className="btn btn-secondary btn-sm btn-icon" type="button" onClick={() => onEdit?.(item)} aria-label={`Edit ${item.name}`} title="Edit">
                {Icons.edit}
              </button>
              <button className="btn btn-danger btn-sm btn-icon" type="button" onClick={() => onDelete?.(item)} disabled={busyKey === `package-delete-${item.id}`} aria-label={`Hapus ${item.name}`} title="Hapus">
                {Icons.trash}
              </button>
            </div>
          ) : null}
        </div>
      </div>
      <div>
        <h3>{isTopUp ? item.name : planDisplayName(item)}</h3>
        <p>{packageDescription(item)}</p>
      </div>
      <div className="owner-package-price">
        <strong>{formatCurrencyIDR(item.price)}</strong>
        <span>{isTopUp ? "sekali bayar" : "per bulan"}</span>
      </div>
      <div className="owner-package-credit">{formatNumber(item.creditAmount)} kredit {isTopUp ? "tambahan" : "bulanan"}</div>
      {isTopUp ? (
        <div className="owner-package-note">Tidak mengubah limit WhatsApp, AI agent, atau user.</div>
      ) : (
        <div className="owner-package-limits">
          <div><strong>{formatNumber(limits.whatsapp)}</strong><span>WA</span></div>
          <div><strong>{formatNumber(limits.aiAgents)}</strong><span>AI agent</span></div>
          <div><strong>{formatNumber(limits.humanUsers)}</strong><span>User</span></div>
        </div>
      )}
    </article>
  );
}

function OwnerCatalogSection({ title, subtitle, packages, type, emptyText, canManage = false, onAdd, onEdit, onDelete, busyKey = "" }) {
  return (
    <section className="owner-catalog-section">
      <div className="owner-catalog-section-head">
        <div>
          <h2>{title}</h2>
          <p>{subtitle}</p>
        </div>
        <div className="owner-catalog-section-actions">
          <span className="badge gray">{packages.length} item</span>
          {canManage && onAdd ? (
            <button className="btn btn-primary btn-sm" type="button" onClick={onAdd}>
              {Icons.plus}
              {type === "topup" ? "Tambah Top-up" : "Tambah Paket"}
            </button>
          ) : null}
        </div>
      </div>
      {packages.length ? (
        <div className={`owner-package-grid ${type}`}>
          {packages.map((item) => (
            <OwnerPackageCard
              key={item.id}
              item={item}
              type={type}
              canManage={canManage}
              onEdit={onEdit}
              onDelete={onDelete}
              busyKey={busyKey}
            />
          ))}
        </div>
      ) : (
        <div className="pricing-empty">{emptyText}</div>
      )}
    </section>
  );
}

function CreatePackageForm({ packageForm, setPackageForm, createPackage, busyKey }) {
  const isPopularDraft = packageForm.isPopular === true || packageForm.isPopular === "true";
  return (
    <form className="wallet-side-form" onSubmit={createPackage}>
      <label>
        Nama Paket
        <input className="form-input" value={packageForm.name} onChange={(event) => setPackageForm((current) => ({ ...current, name: event.target.value }))} placeholder="Contoh: Paket Custom" />
      </label>
      <label>
        Kredit Bulanan
        <input className="form-input" type="number" value={packageForm.creditAmount} onChange={(event) => setPackageForm((current) => ({ ...current, creditAmount: event.target.value }))} />
      </label>
      <label>
        Harga (IDR)
        <input className="form-input" type="number" value={packageForm.price} onChange={(event) => setPackageForm((current) => ({ ...current, price: event.target.value }))} />
      </label>
      <label className="owner-package-mini-toggle">
        <input type="checkbox" checked={isPopularDraft} onChange={(event) => setPackageForm((current) => ({ ...current, isPopular: event.target.checked }))} />
        <span>Paket populer</span>
      </label>
      <button className="btn btn-primary" disabled={busyKey === "package"}>Buat Paket Layanan</button>
    </form>
  );
}

function PackageEditorModal({ packageForm, setPackageForm, onSubmit, onDelete, onClose, busyKey }) {
  const isTopUp = packageForm.billingPeriod === "one_time";
  const isPopularDraft = packageForm.isPopular === true || packageForm.isPopular === "true";
  const title = packageForm.id ? (isTopUp ? "Edit top-up" : "Edit paket layanan") : (isTopUp ? "Tambah top-up" : "Tambah paket layanan");
  const updateField = (key, value) => setPackageForm((current) => ({ ...current, [key]: value }));
  return (
    <div
      className="modal-backdrop billing-modal-backdrop owner-package-editor-backdrop"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose?.();
      }}
    >
      <div className="modal-card billing-modal-card owner-package-editor-modal" role="dialog" aria-modal="true" aria-labelledby="owner-package-editor-title" onMouseDown={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 id="owner-package-editor-title">{title}</h3>
            <p>{isTopUp ? "Atur kredit tambahan sekali bayar yang muncul di katalog checkout." : "Atur harga, kredit bulanan, dan limit utama paket layanan."}</p>
          </div>
          <button className="modal-close" type="button" onClick={onClose} aria-label="Tutup">{Icons.close}</button>
        </div>
        <form id="owner-package-editor-form" className="billing-modal-body owner-package-editor-form" onSubmit={onSubmit}>
          <label>
            {isTopUp ? "Nama top-up" : "Nama paket"}
            <input className="form-input" required value={packageForm.name} onChange={(event) => updateField("name", event.target.value)} placeholder={isTopUp ? "Contoh: Top-up 25K Kredit" : "Contoh: Growth"} />
          </label>
          <label>
            {isTopUp ? "Kredit tambahan" : "Kredit bulanan"}
            <input className="form-input" type="number" min="1" required value={packageForm.creditAmount} onChange={(event) => updateField("creditAmount", event.target.value)} />
          </label>
          <label>
            Harga (IDR)
            <input className="form-input" type="number" min="1" required value={packageForm.price} onChange={(event) => updateField("price", event.target.value)} />
          </label>
          <label>
            Status katalog
            <select className="form-select" value={packageForm.isActive === false ? "inactive" : "active"} onChange={(event) => updateField("isActive", event.target.value === "active")}>
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </select>
          </label>
          {!isTopUp ? (
            <label className="owner-package-popular-toggle owner-package-editor-wide">
              <input type="checkbox" checked={isPopularDraft} onChange={(event) => updateField("isPopular", event.target.checked)} />
              <span>
                <strong>Tandai sebagai paket populer</strong>
                <small>Badge Populer muncul di kartu paket. Kalau aktif, paket populer lain otomatis dilepas.</small>
              </span>
            </label>
          ) : null}
          {!isTopUp ? (
            <>
              <label>
                Nomor WhatsApp
                <input className="form-input" type="number" min="1" required value={packageForm.maxWhatsAppSessions} onChange={(event) => updateField("maxWhatsAppSessions", event.target.value)} />
              </label>
              <label>
                AI agent
                <input className="form-input" type="number" min="1" required value={packageForm.maxAiAgents} onChange={(event) => updateField("maxAiAgents", event.target.value)} />
              </label>
              <label>
                User tim
                <input className="form-input" type="number" min="1" required value={packageForm.maxHumanUsers} onChange={(event) => updateField("maxHumanUsers", event.target.value)} />
              </label>
            </>
          ) : null}
          <label className="owner-package-editor-wide">
            Deskripsi
            <textarea className="form-textarea" rows={3} value={packageForm.description || ""} onChange={(event) => updateField("description", event.target.value)} placeholder="Deskripsi singkat untuk katalog billing" />
          </label>
          <div className="owner-package-type-note owner-package-editor-wide">
            <span className={`badge ${isTopUp ? "purple" : "blue"}`}>{isTopUp ? "Top-up sekali bayar" : "Subscription bulanan"}</span>
            <p>{isTopUp ? "Top-up menambah kredit tambahan dan tidak mengubah limit WhatsApp, AI agent, atau user." : "Paket layanan mengatur limit utama organisasi per bulan."}</p>
          </div>
        </form>
        <div className="modal-actions owner-package-editor-actions">
          {packageForm.id ? (
            <button className="btn btn-danger" type="button" onClick={() => onDelete?.(packageForm)} disabled={busyKey === `package-delete-${packageForm.id}`}>
              {Icons.trash}
              Hapus
            </button>
          ) : null}
          <button className="btn btn-secondary" type="button" onClick={onClose}>Batal</button>
          <button className="btn btn-primary" type="submit" form="owner-package-editor-form" disabled={busyKey === "package"}>
            {packageForm.id ? "Simpan perubahan" : isTopUp ? "Tambah top-up" : "Tambah paket"}
          </button>
        </div>
      </div>
    </div>
  );
}

function PlanCard({ item, currentPlanKey, currentBillingPeriod = "monthly", canRequestPurchases, isTrial, onSelect, variant = "default", animationIndex = 0 }) {
  const key = packageKey(item);
  const limits = packageLimits(item);
  const billingPeriod = packagePeriod(item);
  const isAnnual = billingPeriod === "annual";
  const isCurrent = key && key === currentPlanKey && billingPeriod === String(currentBillingPeriod || "monthly").toLowerCase();
  const isPopular = packageIsPopular(item);
  const creditAmount = numberValue(item.creditAmount);
  const displayName = planDisplayName(item);
  const isPicker = variant === "picker";

  if (isPicker) {
    const features = [
      `${formatNumber(creditAmount)} kredit per bulan`,
      `${formatNumber(limits.whatsapp)} sesi WhatsApp`,
      `${formatNumber(limits.aiAgents)} AI agent`,
      `${formatNumber(limits.humanUsers)} pengguna tim`,
    ];
    return (
      <article
        className={`billing-public-plan-card ${isPopular ? "dark" : ""}`}
        style={{ "--billing-plan-index": animationIndex }}
      >
        <div className={isPopular ? "billing-public-plan-header" : ""}>
          <div className="billing-public-plan-title">
            <h3>{displayName}</h3>
            <p>{packageDescription(item)}</p>
          </div>
          {isPopular ? <span className="billing-public-popular-badge">Populer</span> : null}
        </div>
        <div className="billing-public-plan-price">
          <strong>{formatPublicPlanPrice(item.price)}</strong>
          <span>/ {isAnnual ? "tahun" : "bulan"}</span>
        </div>
        <button
          className={`billing-public-plan-cta ${isPopular ? "light" : ""}`}
          type="button"
          disabled={!canRequestPurchases || isCurrent}
          onClick={() => onSelect(item)}
          aria-label={isCurrent ? `${displayName} sedang aktif` : `Mulai ${displayName} ${isAnnual ? "tahunan" : "bulanan"}`}
        >
          {isCurrent ? "Paket aktif" : "Mulai"}
        </button>
        <ul className="billing-public-plan-list">
          {features.map((feature) => (
            <li key={feature}>
              <span>{Icons.check}</span>
              {feature}
            </li>
          ))}
        </ul>
      </article>
    );
  }

  return (
    <article className={`upgrade-plan-card ${isPopular ? "popular" : ""} ${key === "business" ? "featured" : ""} ${isCurrent ? "active" : ""}`}>
      <div className="pricing-plan-top">
        <div>
          <h3>{displayName}</h3>
          <p>{planShortDescription(item)}</p>
        </div>
        {isCurrent ? <span className="pricing-plan-badge current">Paket aktif</span> : isPopular ? <span className="pricing-plan-badge">Populer</span> : null}
      </div>

      <div className="pricing-plan-price">
        <strong>{formatCurrencyIDR(item.price)}</strong>
        <span>/ bulan</span>
      </div>

      <div className="pricing-credit-line">{formatNumber(creditAmount)} kredit per bulan</div>

      <div className="pricing-limit-grid">
        <div><strong>{formatNumber(limits.whatsapp)}</strong><span>WhatsApp</span></div>
        <div><strong>{formatNumber(limits.aiAgents)}</strong><span>AI agent</span></div>
        <div><strong>{formatNumber(limits.humanUsers)}</strong><span>User</span></div>
      </div>

      <ul className="pricing-benefit-list">
        {planBenefits(item).map((benefit) => <li key={benefit}>{benefit}</li>)}
      </ul>

      <button
        className={`btn ${isCurrent ? "btn-secondary" : "btn-primary"}`}
        type="button"
        disabled={!canRequestPurchases || isCurrent}
        onClick={() => onSelect(item)}
      >
        {isCurrent ? "Paket aktif" : isTrial ? `Pilih ${displayName}` : key === "business" ? `Upgrade ke ${displayName}` : `Pilih ${displayName}`}
      </button>
    </article>
  );
}

function AddOnCard({ item, selectedPackageId, canRequestPurchases, addOnsEnabled, onSelect }) {
  const selected = selectedPackageId === item.id;
  const disabled = !canRequestPurchases || !addOnsEnabled;
  return (
    <article className={`pricing-addon-card ${selected ? "selected" : ""}`}>
      <div>
        <h4>{item.name}</h4>
        <p>{packageDescription(item)}</p>
      </div>
      <div className="pricing-addon-meta">
        <strong>{formatNumber(item.creditAmount)} Cr</strong>
        <span>{formatCurrencyIDR(item.price)}</span>
      </div>
      <button
        className={`btn btn-sm ${selected ? "btn-secondary" : "btn-primary"}`}
        type="button"
        disabled={disabled}
        onClick={() => onSelect(item.id)}
      >
        {!addOnsEnabled ? "Butuh paket" : selected ? "Dipilih" : "Pilih add-on"}
      </button>
    </article>
  );
}

function PurchaseRequestPanel({ selectedPackage, purchaseForm, setPurchaseForm, submitPurchaseRequest, busyKey, canRequestPurchases }) {
  return (
    <form className="pricing-request-panel" onSubmit={submitPurchaseRequest}>
      <div>
        <div className="wallet-panel-title">Request ke Owner</div>
        <div className="wallet-panel-subtitle">
          {selectedPackage ? `${selectedPackage.name} - ${formatCurrencyIDR(selectedPackage.price)}` : "Pilih paket atau add-on dulu."}
        </div>
      </div>
      <textarea
        className="form-textarea"
        rows={2}
        value={purchaseForm.notes || ""}
        onChange={(event) => setPurchaseForm((current) => ({ ...current, notes: event.target.value }))}
        placeholder="Catatan untuk owner..."
        disabled={!canRequestPurchases}
      />
      <button className="btn btn-primary" disabled={!canRequestPurchases || !selectedPackage || busyKey === "purchase-request"}>
        Ajukan request
      </button>
    </form>
  );
}

function PaymentMethodOptions({ selectedMethod, onSelect }) {
  const normalizedSelectedMethod = availablePaymentMethod(selectedMethod);
  return (
    <div className="billing-payment-list" role="radiogroup" aria-label="Metode pembayaran">
      {PAYMENT_METHODS.map((method) => {
        const selected = normalizedSelectedMethod === method.id;
        return (
          <button
            key={method.id}
            type="button"
            className={`billing-payment-option ${selected ? "selected" : ""} ${method.disabled ? "disabled" : ""}`}
            onClick={() => {
              if (!method.disabled) onSelect(method.id);
            }}
            disabled={method.disabled}
            role="radio"
            aria-checked={selected}
            aria-label={method.disabled ? `${method.label} belum tersedia` : `Pilih ${method.label}`}
          >
            <span className="billing-payment-option-head">
              <span className="billing-payment-icon" aria-hidden="true">{Icons[method.icon] || Icons.wallet}</span>
              <span className="billing-radio-control" aria-hidden="true">{selected ? Icons.check : null}</span>
            </span>
            <span className="billing-payment-copy">
              <span className="billing-payment-title">
                <strong>{method.label}</strong>
                {method.badge ? <em>{method.badge}</em> : null}
              </span>
              <small>{method.description}</small>
              {method.feeHint ? <span className="billing-payment-fee">{method.feeHint}</span> : null}
            </span>
          </button>
        );
      })}
    </div>
  );
}

function CheckoutSummary({ item, title, subtitle, paymentMethod }) {
  if (!item) return null;
  const limits = packageLimits(item);
  const isTopUp = isAddOnPackage(item);
  const isAnnual = packagePeriod(item) === "annual";
  const quote = paymentQuote(item, paymentMethod);
  return (
    <div className="billing-checkout-summary">
      <div className="billing-checkout-main">
        <span>{title}</span>
        <strong>{item.name || planDisplayName(item)}</strong>
        <small>{subtitle}</small>
      </div>
      <div className="billing-checkout-price">
        <span>Total dibayar</span>
        <strong>{formatCurrencyIDR(quote.grossAmount)}</strong>
        <small>Sudah termasuk biaya pembayaran</small>
      </div>
      <div className="billing-checkout-limits">
        <div><span>{isTopUp ? "Kredit" : "Kredit/bln"}</span><strong>{formatNumber(item.creditAmount)} Cr</strong></div>
        {isTopUp ? (
          <>
            <div><span>Tipe</span><strong>Top-up</strong></div>
            <div><span>Masa aktif</span><strong>Ikut paket aktif</strong></div>
          </>
        ) : (
          <>
            <div><span>WhatsApp</span><strong>{formatNumber(limits.whatsapp)}</strong></div>
            <div><span>AI agent</span><strong>{formatNumber(limits.aiAgents)}</strong></div>
            <div><span>User</span><strong>{formatNumber(limits.humanUsers)}</strong></div>
            <div><span>Masa aktif</span><strong>{isAnnual ? "12 bulan" : "1 bulan"}</strong></div>
          </>
        )}
      </div>
    </div>
  );
}

function BillingPeriodToggle({ value, onChange, annualDiscountPercent }) {
  const isAnnual = value === "annual";
  return (
    <div className="billing-public-period-row" role="group" aria-label="Pilih periode pembayaran">
      <span>Bulanan</span>
      <button
        type="button"
        className={`billing-public-switch ${isAnnual ? "active" : ""}`}
        role="switch"
        aria-checked={isAnnual}
        aria-label="Ganti periode harga"
        onClick={() => onChange(isAnnual ? "monthly" : "annual")}
      >
        <span />
      </button>
      <span>Tahunan</span>
      <small>Hemat {annualDiscountPercent}%</small>
    </div>
  );
}

function PaymentFeeBreakdown({ item, paymentMethod }) {
  if (!item) return null;
  const quote = paymentQuote(item, paymentMethod);
  return (
    <div className="billing-fee-box">
      <div>
        <span>Harga paket</span>
        <strong>{formatCurrencyIDR(quote.packageAmount)}</strong>
      </div>
      <div>
        <span>{quote.feeLabel}</span>
        <strong>{formatCurrencyIDR(quote.paymentFee)}</strong>
      </div>
      <div className="total">
        <span>Total dibayar</span>
        <strong>{formatCurrencyIDR(quote.grossAmount)}</strong>
      </div>
    </div>
  );
}

function PendingPaymentBanner({ purchase, onChangePaymentMethod, onCancelPurchase, busyKey }) {
  if (!purchase) return null;
  return (
    <div className="billing-pending-banner">
      <div>
        <strong>Pembayaran belum selesai</strong>
        <span>{purchase.packageName || "Paket"} - {paymentMethodLabel(purchase.paymentMethod)} - dibuat {formatDateTime(purchase.createdAt)}</span>
      </div>
      <div className="billing-pending-actions">
        <button className="btn btn-secondary btn-sm" type="button" onClick={() => onChangePaymentMethod?.(purchase)}>
          Ubah metode pembayaran
        </button>
        {canContinuePayment(purchase) ? (
          <a className="btn btn-primary btn-sm" href={purchase.snapRedirectUrl} target="_blank" rel="noreferrer">Lanjutkan pembayaran</a>
        ) : null}
        <button className="btn btn-danger btn-sm" type="button" onClick={() => onCancelPurchase?.(purchase)} disabled={busyKey === `purchase-cancel-${purchase.id}`}>
          {busyKey === `purchase-cancel-${purchase.id}` ? "Membatalkan..." : "Batalkan"}
        </button>
      </div>
    </div>
  );
}

function BillingUsageSummary({ wallet, billingPlan, isTrial }) {
  const monthlyLimit = walletValue(wallet, "monthlyCreditLimit", "monthlyLimit");
  const monthlyUsed = walletValue(wallet, "monthlyCreditsUsed", "monthlyUsed");
  const monthlyRemaining = walletValue(wallet, "monthlyCreditsRemaining", "monthlyRemaining");
  const additionalRemaining = walletValue(wallet, "additionalCreditsRemaining", "additionalRemaining");
  const usagePercent = percentValue(monthlyUsed, monthlyLimit);
  const totalRemaining = monthlyRemaining + additionalRemaining;
  const isTrialExhausted = isTrial && totalRemaining <= 0;

  if (isTrial) {
    return (
      <div className="billing-usage-card trial">
        <div className="billing-usage-top">
          <div>
            <h3>{isTrialExhausted ? "Kredit trial telah habis" : `${formatNumber(totalRemaining)} Cr trial tersedia`}</h3>
            <p>
              {isTrialExhausted
                ? `${formatNumber(monthlyUsed)} dari ${formatNumber(monthlyLimit)} chat trial telah digunakan. Playground dan AI test butuh paket aktif atau top-up sebelum lanjut.`
                : `${formatNumber(monthlyRemaining)} dari ${formatNumber(monthlyLimit)} chat trial akun baru masih bisa dipakai untuk setup AI CS dan tes Playground. WhatsApp produksi tetap perlu paket bulanan.`}
            </p>
          </div>
          <div className={`billing-usage-total ${isTrialExhausted ? "danger" : ""}`}>
            <strong>{formatNumber(totalRemaining)} Cr</strong>
            <span>tersisa</span>
          </div>
        </div>
        <div className={`billing-progress ${isTrialExhausted ? "danger" : ""}`}><span style={{ width: `${usagePercent}%` }} /></div>
        <div className="billing-progress-caption">
          <span>{formatNumber(monthlyUsed)} dari {formatNumber(monthlyLimit)} kredit trial telah digunakan</span>
          <strong>{wallet?.nextResetAt ? `Reset ${formatDateTime(wallet.nextResetAt)}` : "Trial aktif"}</strong>
        </div>
        <div className="billing-usage-breakdown">
          <div><span>Paket</span><strong>Trial</strong></div>
          <div><span>Kredit trial</span><strong>{formatNumber(monthlyRemaining)} Cr</strong></div>
          <div><span>Produksi</span><strong>Terkunci</strong></div>
        </div>
      </div>
    );
  }

  return (
    <div className="billing-usage-card">
      <div className="billing-usage-top">
        <div>
          <h3>{formatNumber(totalRemaining)} Cr tersedia</h3>
          <p>Kredit bulanan hampir habis, tetapi masih ada kredit add-on yang dapat dipakai setelah kredit bulanan habis.</p>
        </div>
        <div className="billing-usage-total">
          <strong>{formatNumber(totalRemaining)} Cr</strong>
          <span>tersisa</span>
        </div>
      </div>
      <div className="billing-progress"><span style={{ width: `${usagePercent}%` }} /></div>
      <div className="billing-progress-caption">
        <span>{formatNumber(monthlyUsed)} dari {formatNumber(monthlyLimit)} kredit bulanan telah digunakan</span>
        <strong>{wallet?.nextResetAt ? `Reset ${formatDateTime(wallet.nextResetAt)}` : "Reset periode berikutnya"}</strong>
      </div>
      <div className="billing-usage-breakdown">
        <div><span>Paket aktif</span><strong>{billingPlan?.planName || "Paket aktif"}</strong></div>
        <div><span>Kredit bulanan</span><strong>{formatNumber(monthlyRemaining)} Cr</strong></div>
        <div><span>Kredit add-on</span><strong>{formatNumber(additionalRemaining)} Cr</strong></div>
        <div><span>Masa aktif</span><strong>{billingPlan?.activeUntil ? formatDateTime(billingPlan.activeUntil) : "Periode berjalan"}</strong></div>
      </div>
      <div className="billing-note">Top-up menambah kredit add-on dan tidak mengubah limit WhatsApp, AI agent, atau user.</div>
    </div>
  );
}

function PlanCheckoutModal({ plan, currentPlanKey, paymentMethod, setPaymentMethod, existingPurchase, busyKey, onClose, onSubmit }) {
  if (!plan) return null;
  const key = packageKey(plan);
  const isUpgrade = currentPlanKey && key !== currentPlanKey;
  const isAnnual = packagePeriod(plan) === "annual";

  return (
    <div className="modal-backdrop billing-modal-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose();
    }}>
      <div className="modal-card billing-modal-card billing-checkout-card" role="dialog" aria-modal="true" aria-labelledby="plan-checkout-title">
        <div className="modal-header">
          <div>
            <h3 id="plan-checkout-title">{isUpgrade ? `Upgrade ke ${planDisplayName(plan)}` : `Checkout ${planDisplayName(plan)}`}</h3>
            <p>Periksa detail paket dan pilih metode pembayaran.</p>
          </div>
          <button className="modal-close" type="button" onClick={onClose} aria-label="Tutup">{Icons.close}</button>
        </div>
        <div className="billing-modal-body billing-checkout-body">
          <CheckoutSummary item={plan} title="Paket dipilih" subtitle={isAnnual ? "Langganan tahunan" : "Langganan bulanan"} paymentMethod={paymentMethod} />
          <div className="billing-period-safety-note" role="note">
            <strong>{isAnnual ? `Tahunan - Hemat ${annualDiscountValue(plan)}%` : "Tagihan bulanan"}</strong>
            <span>{isAnnual ? "Dibayar sekali untuk 12 bulan. Kredit paket tersedia per bulan dan direset pada setiap siklus bulanan selama masa aktif." : "Checkout ini mengaktifkan paket selama 1 bulan. Pilih opsi tahunan di langkah sebelumnya untuk harga hemat setahun."}</span>
          </div>
          {existingPurchase ? (
            <div className="billing-existing-request">
              <div>
                <strong>Pembayaran paket ini belum selesai</strong>
                <span>{paymentMethodLabel(existingPurchase.paymentMethod)} - dibuat {formatDateTime(existingPurchase.createdAt)}</span>
              </div>
              <StatusPill value={existingPurchase.paymentStatus} kind="payment" />
            </div>
          ) : null}
          <div className="billing-modal-section compact">
            <div className="billing-modal-section-head">
              <h4>Metode pembayaran</h4>
              <span>Aman oleh Midtrans</span>
            </div>
            <PaymentMethodOptions selectedMethod={paymentMethod} onSelect={setPaymentMethod} />
          </div>
          <PaymentFeeBreakdown item={plan} paymentMethod={paymentMethod} />
        </div>
        <div className="modal-actions">
          <button className="btn btn-secondary" type="button" onClick={onClose}>Batal</button>
          {existingPurchase?.snapRedirectUrl ? (
            <a className="btn btn-secondary" href={existingPurchase.snapRedirectUrl} target="_blank" rel="noreferrer">Lanjutkan pembayaran</a>
          ) : null}
          <button className="btn btn-primary" type="button" disabled={busyKey === "purchase-request"} onClick={() => onSubmit(plan, `Pembelian paket ${planDisplayName(plan)} ${isAnnual ? "tahunan" : "bulanan"}`)}>
            {busyKey === "purchase-request" ? "Membuka Midtrans..." : <>{existingPurchase ? "Ganti metode pembayaran" : "Bayar sekarang"} {Icons.arrowRight}</>}
          </button>
        </div>
      </div>
    </div>
  );
}

function PlanPickerModal({ plans, currentPlanKey, currentBillingPeriod, billingPlan, checkoutPeriod, annualDiscountPercent, isTrial, canRequestPurchases, onPeriodChange, onSelect, onClose }) {
  const currentPlanName = isTrial ? "Free" : billingPlan?.planName || "Paket aktif";
  const planCountClass = `count-${Math.min(Math.max(plans.length, 1), 3)}`;
  return (
    <div className="modal-backdrop billing-modal-backdrop billing-plan-picker-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose();
    }}>
      <div className="modal-card billing-modal-card billing-plan-picker-card" role="dialog" aria-modal="true" aria-labelledby="plan-picker-title">
        <div className="modal-header billing-plan-picker-header">
          <div className="billing-plan-picker-heading">
            <div className="billing-plan-picker-mark">{Icons.products}</div>
            <div>
              <h3 id="plan-picker-title">{isTrial ? "Pilih paket" : "Ubah paket"}</h3>
              <p>Pilih paket yang sesuai, lalu lanjut ke rincian pembayaran.</p>
            </div>
          </div>
          <button className="modal-close" type="button" onClick={onClose} aria-label="Tutup">{Icons.close}</button>
        </div>
        <div className="billing-modal-body billing-plan-picker-body">
          <div className="billing-current-plan-strip">
            <div className="billing-current-plan-main">
              <div className="billing-current-plan-icon">{Icons.wallet}</div>
              <div>
                <span>Paket saat ini</span>
                <strong>{currentPlanName}</strong>
              </div>
            </div>
            <div className="billing-current-plan-date">
              {Icons.calendar}
              <em>{billingPlan?.activeUntil ? `Aktif sampai ${formatDateTime(billingPlan.activeUntil)}` : isTrial ? "Trial aktif" : "Periode berjalan"}</em>
            </div>
          </div>
          <div className="billing-public-pricing-shell">
            <BillingPeriodToggle value={checkoutPeriod} onChange={onPeriodChange} annualDiscountPercent={annualDiscountPercent} />
            {plans.length ? (
              <div className={`upgrade-plan-grid billing-plan-picker-grid ${planCountClass}`}>
                {plans.map((item, index) => (
                  <PlanCard
                    key={`${item.id}-${checkoutPeriod}`}
                    item={item}
                    currentPlanKey={currentPlanKey}
                    currentBillingPeriod={currentBillingPeriod}
                    canRequestPurchases={canRequestPurchases}
                    isTrial={isTrial}
                    onSelect={onSelect}
                    variant="picker"
                    animationIndex={index}
                  />
                ))}
              </div>
            ) : (
              <div className="pricing-empty billing-plan-empty">
                <strong>Paket tertinggi sudah aktif</strong>
                <span>Belum ada paket lanjutan yang tersedia untuk akun ini.</span>
              </div>
            )}
          </div>
          <div className="billing-period-safety-note" role="note">
            <strong>{checkoutPeriod === "annual" ? "Ditagih tahunan" : "Ditagih bulanan"}</strong>
            <span>{checkoutPeriod === "annual" ? `Harga tahunan mengikuti halaman Harga dengan diskon ${annualDiscountPercent}%. Kredit tetap dialokasikan per bulan selama 12 bulan.` : "Pembayaran bulanan mengaktifkan paket selama 1 bulan dan dapat diperpanjang pada periode berikutnya."}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function TopUpModal({ packages, selectedPackageId, paymentMethod, setPaymentMethod, existingPurchase, busyKey, onSelectPackage, onClose, onSubmit }) {
  const selectedPackage = packages.find((item) => item.id === selectedPackageId) || packages[0];

  return (
    <div className="modal-backdrop billing-modal-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose();
    }}>
      <div className="modal-card billing-modal-card" role="dialog" aria-modal="true" aria-labelledby="topup-checkout-title">
        <div className="modal-header">
          <div>
            <h3 id="topup-checkout-title">Top-up kredit</h3>
            <p>Periksa nominal top-up dan pilih metode pembayaran.</p>
          </div>
          <button className="modal-close" type="button" onClick={onClose} aria-label="Tutup">{Icons.close}</button>
        </div>
        <div className="billing-modal-body">
          <CheckoutSummary item={selectedPackage} title="Top-up dipilih" subtitle="Kredit add-on sekali beli" paymentMethod={paymentMethod} />
          {existingPurchase ? (
            <div className="billing-existing-request">
              <div>
                <strong>Pembayaran top-up ini belum selesai</strong>
                <span>{paymentMethodLabel(existingPurchase.paymentMethod)} - dibuat {formatDateTime(existingPurchase.createdAt)}</span>
              </div>
              <StatusPill value={existingPurchase.paymentStatus} kind="payment" />
            </div>
          ) : null}
          <div className="billing-modal-section compact">
            <div className="billing-modal-section-head">
              <h4>Nominal top-up</h4>
              <span>Kredit add-on</span>
            </div>
            <div className="billing-topup-list">
              {packages.map((item) => {
                const selected = selectedPackage?.id === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    className={`billing-topup-option ${selected ? "selected" : ""}`}
                    onClick={() => onSelectPackage(item.id)}
                    aria-pressed={selected}
                  >
                    <span>
                      <strong>{item.name}</strong>
                      <small>{formatNumber(item.creditAmount)} Cr</small>
                    </span>
                    <em>{formatCurrencyIDR(item.price)}</em>
                  </button>
                );
              })}
            </div>
          </div>
          <div className="billing-modal-section compact">
            <div className="billing-modal-section-head">
              <h4>Metode pembayaran</h4>
              <span>Aman oleh Midtrans</span>
            </div>
            <PaymentMethodOptions selectedMethod={paymentMethod} onSelect={setPaymentMethod} />
          </div>
          <PaymentFeeBreakdown item={selectedPackage} paymentMethod={paymentMethod} />
        </div>
        <div className="modal-actions">
          <button className="btn btn-secondary" type="button" onClick={onClose}>Batal</button>
          {existingPurchase?.snapRedirectUrl ? (
            <a className="btn btn-secondary" href={existingPurchase.snapRedirectUrl} target="_blank" rel="noreferrer">Lanjutkan pembayaran</a>
          ) : null}
          <button className="btn btn-primary" type="button" disabled={!selectedPackage || busyKey === "purchase-request"} onClick={() => onSubmit(selectedPackage, `Pembelian top-up ${selectedPackage?.name || ""}`)}>
            {busyKey === "purchase-request" ? "Membuka Midtrans..." : <>{existingPurchase ? "Ganti metode pembayaran" : "Bayar sekarang"} {Icons.arrowRight}</>}
          </button>
        </div>
      </div>
    </div>
  );
}

function UserUpgradeView({
  wallet,
  billingPlan,
  packages,
  purchases,
  purchaseForm,
  setPurchaseForm,
  submitPurchaseRequest,
  cancelPurchase,
  canRequestPurchases,
  busyKey,
}) {
  const [planPickerOpen, setPlanPickerOpen] = useState(false);
  const [checkoutPlan, setCheckoutPlan] = useState(null);
  const [topUpOpen, setTopUpOpen] = useState(false);
  const [selectedTopUpId, setSelectedTopUpId] = useState("");
  const [paymentMethod, setPaymentMethod] = useState(availablePaymentMethod(purchaseForm.paymentMethod));
  const [checkoutPeriod, setCheckoutPeriod] = useState("monthly");
  const monthlyLimit = walletValue(wallet, "monthlyCreditLimit", "monthlyLimit");
  const monthlyUsed = walletValue(wallet, "monthlyCreditsUsed", "monthlyUsed");
  const monthlyRemaining = walletValue(wallet, "monthlyCreditsRemaining", "monthlyRemaining");
  const additionalRemaining = walletValue(wallet, "additionalCreditsRemaining", "additionalRemaining");
  const usagePercent = percentValue(monthlyUsed, monthlyLimit);
  const currentPlanKey = String(billingPlan?.planKey || "").toLowerCase();
  const currentBillingPeriod = String(billingPlan?.billingPeriod || "monthly").toLowerCase();
  const isTrial = monthlyLimit <= 10 || currentPlanKey === "trial";
  const addOnsEnabled = Boolean(currentPlanKey && currentPlanKey !== "trial");
  const active = activePackages(packages);
  const annualDiscountPercent = annualDiscountValue(active.find(isPlanPackage));
  const planPackages = availablePlanPackages(active.filter(isPlanPackage), currentPlanKey, checkoutPeriod)
    .map((item) => planForCheckoutPeriod(item, checkoutPeriod));
  const addOnPackages = active.filter(isAddOnPackage).sort((a, b) => numberValue(a.creditAmount) - numberValue(b.creditAmount));
  const pendingPurchase = latestOpenPurchase(purchases);
  const totalRemaining = monthlyRemaining + additionalRemaining;
  const isTrialExhausted = isTrial && totalRemaining <= 0;
  const defaultTopUpId = selectedTopUpId || addOnPackages[1]?.id || addOnPackages[0]?.id || "";
  const selectedTopUpPackage = addOnPackages.find((item) => item.id === defaultTopUpId) || addOnPackages[0] || null;
  const checkoutOpenPurchase = openPurchaseForPackage(purchases, checkoutPlan);
  const topUpOpenPurchase = openPurchaseForPackage(purchases, selectedTopUpPackage);

  const changeCheckoutPeriod = (nextPeriod) => {
    if (nextPeriod === checkoutPeriod) return;
    const prefersReducedMotion = typeof window !== "undefined"
      && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (typeof document !== "undefined" && typeof document.startViewTransition === "function" && !prefersReducedMotion) {
      document.startViewTransition(() => setCheckoutPeriod(nextPeriod));
      return;
    }
    setCheckoutPeriod(nextPeriod);
  };

  const openPlanCheckout = (item) => {
    const existing = openPurchaseForPackage(purchases, item);
    setPaymentMethod(availablePaymentMethod(existing?.paymentMethod || purchaseForm.paymentMethod));
    setPlanPickerOpen(false);
    setCheckoutPlan(item);
  };

  const openTopUpModal = () => {
    const nextTopUp = addOnPackages.find((item) => item.id === defaultTopUpId) || addOnPackages[0] || null;
    const existing = openPurchaseForPackage(purchases, nextTopUp);
    setPaymentMethod(availablePaymentMethod(existing?.paymentMethod || purchaseForm.paymentMethod));
    setSelectedTopUpId(defaultTopUpId);
    setTopUpOpen(true);
  };

  const openPaymentMethodForPurchase = (purchase) => {
    const purchaseKey = packageKey(purchase);
    const basePackage = active.find((item) => purchase?.packageId && item.id === purchase.packageId)
      || active.find((item) => purchaseKey && packageKey(item) === purchaseKey && packagePeriod(item) === String(purchase?.billingPeriod || "monthly").toLowerCase());
    const targetPackage = basePackage && !isAddOnPackage(basePackage)
      ? planForCheckoutPeriod(basePackage, String(purchase?.billingPeriod || "monthly").toLowerCase())
      : basePackage;
    if (!targetPackage) return;
    setPaymentMethod(availablePaymentMethod(purchase?.paymentMethod || purchaseForm.paymentMethod));
    setPlanPickerOpen(false);
    if (isAddOnPackage(targetPackage)) {
      setCheckoutPlan(null);
      setSelectedTopUpId(targetPackage.id);
      setTopUpOpen(true);
      return;
    }
    setTopUpOpen(false);
    setCheckoutPeriod(packagePeriod(targetPackage));
    setCheckoutPlan(targetPackage);
  };

  const requestPurchase = async (item, notes) => {
    if (!item) return;
    const nextForm = {
      ...purchaseForm,
      packageId: item.id,
      billingPeriod: packagePeriod(item),
      paymentMethod,
      notes,
    };
    setPurchaseForm(nextForm);
    await submitPurchaseRequest({ preventDefault() {} }, nextForm);
    setPlanPickerOpen(false);
    setCheckoutPlan(null);
    setTopUpOpen(false);
  };

  return (
    <div className="wallet-page pricing-page">
      <section className="pricing-hero billing-status-hero">
        <div>
          <h2>Billing & Status Kredit</h2>
          <p>Pantau paket aktif, masa aktif, kredit tersedia, pembayaran berjalan, dan riwayat pembelian dalam satu tempat.</p>
        </div>
        <div className="pricing-current-box">
          <span>{isTrial ? "Status trial" : "Total kredit tersedia"}</span>
          <strong>{formatNumber(totalRemaining)} Cr</strong>
          <small>{isTrial ? (isTrialExhausted ? "Kredit trial sudah habis" : `${formatNumber(monthlyRemaining)} dari ${formatNumber(monthlyLimit)} kredit trial tersisa`) : `${formatNumber(monthlyRemaining)} kredit bulanan + ${formatNumber(additionalRemaining)} kredit add-on`}</small>
        </div>
      </section>

      <PendingPaymentBanner purchase={pendingPurchase} onChangePaymentMethod={openPaymentMethodForPurchase} onCancelPurchase={cancelPurchase} busyKey={busyKey} />

      {isTrial ? (
        <div className={`billing-trial-banner ${isTrialExhausted ? "exhausted" : "active"}`}>
          <strong>{isTrialExhausted ? "Batas trial telah tercapai" : "Trial aktif"}</strong>
          <span>
            {isTrialExhausted
              ? "Pilih paket bulanan atau tahunan untuk mengaktifkan WhatsApp produksi, AI agent tambahan, user, dan kredit bulanan."
              : `${formatNumber(monthlyRemaining)} kredit trial masih tersedia untuk mencoba AI di Playground. Pilih paket bulanan atau tahunan saat siap mengaktifkan WhatsApp produksi.`}
          </span>
        </div>
      ) : null}

      <section className="billing-split-grid">
        <div className="billing-panel">
          <div className="billing-panel-head">
            <div>
              <h3>Status kredit</h3>
              <p>{isTrial ? "Ringkasan pemakaian trial." : "Ringkasan pemakaian kredit pada periode berjalan."}</p>
            </div>
            <span className={`badge ${isTrialExhausted ? "red" : "orange"}`}>{isTrialExhausted ? "Trial habis" : `${usagePercent}% terpakai`}</span>
          </div>
          <BillingUsageSummary wallet={wallet} billingPlan={billingPlan} isTrial={isTrial} />
        </div>

        <div className="billing-panel">
          <div className="billing-panel-head">
            <div>
              <h3>Aksi billing</h3>
              <p>Ubah paket atau tambah kredit tanpa meninggalkan halaman status.</p>
            </div>
          </div>
          <div className="billing-action-list">
            <div className="billing-action-row">
              <div>
                <strong>{isTrial ? "Aktifkan paket" : "Ubah paket"}</strong>
                <span>{isTrial ? "Pilih paket bulanan atau tahunan untuk membuka layanan produksi." : "Lihat pilihan periode dan paket yang tersedia untuk akun ini."}</span>
              </div>
              <button className="btn btn-primary" type="button" onClick={() => setPlanPickerOpen(true)} disabled={!canRequestPurchases}>
                {isTrial ? "Pilih Paket" : "Ubah Paket"}
              </button>
            </div>
            <div className={`billing-action-row ${isTrial ? "locked" : ""}`}>
              <div>
                <strong>Top-up kredit</strong>
                <span>{isTrial ? "Top-up tersedia setelah paket bulanan aktif." : "Tambah kredit add-on tanpa mengubah paket utama."}</span>
              </div>
              <button className="btn btn-secondary" type="button" onClick={openTopUpModal} disabled={!addOnsEnabled || !addOnPackages.length}>
                Top-up Kredit
              </button>
            </div>
          </div>
          <div className="muted-note">Kredit add-on dipakai setelah kredit bulanan habis dan tidak ikut reset bulanan.</div>
        </div>
      </section>

      <PurchaseHistory purchases={purchases} cancelPurchase={cancelPurchase} busyKey={busyKey} />

      {planPickerOpen ? (
        <PlanPickerModal
          plans={planPackages}
          currentPlanKey={currentPlanKey}
          currentBillingPeriod={currentBillingPeriod}
          billingPlan={billingPlan}
          checkoutPeriod={checkoutPeriod}
          annualDiscountPercent={annualDiscountPercent}
          isTrial={isTrial}
          canRequestPurchases={canRequestPurchases}
          onPeriodChange={changeCheckoutPeriod}
          onSelect={openPlanCheckout}
          onClose={() => setPlanPickerOpen(false)}
        />
      ) : null}

      {checkoutPlan ? (
        <PlanCheckoutModal
          plan={checkoutPlan}
          currentPlanKey={currentPlanKey}
          paymentMethod={paymentMethod}
          setPaymentMethod={setPaymentMethod}
          existingPurchase={checkoutOpenPurchase}
          busyKey={busyKey}
          onClose={() => setCheckoutPlan(null)}
          onSubmit={requestPurchase}
        />
      ) : null}

      {topUpOpen ? (
        <TopUpModal
          packages={addOnPackages}
          selectedPackageId={defaultTopUpId}
          paymentMethod={paymentMethod}
          setPaymentMethod={setPaymentMethod}
          existingPurchase={topUpOpenPurchase}
          busyKey={busyKey}
          onSelectPackage={setSelectedTopUpId}
          onClose={() => setTopUpOpen(false)}
          onSubmit={requestPurchase}
        />
      ) : null}
    </div>
  );
}

function OwnerBillingView({
  wallet,
  packages,
  pricingForm,
  setPricingForm,
  savePricing,
  packageForm,
  setPackageForm,
  createPackage,
  editPackage,
  deletePackage,
  resetPackageForm,
  canManagePackages,
  purchases,
  cancelPurchase,
  busyKey,
}) {
  const [packageEditorOpen, setPackageEditorOpen] = useState(false);
  const monthlyLimit = walletValue(wallet, "monthlyCreditLimit", "monthlyLimit");
  const monthlyUsed = walletValue(wallet, "monthlyCreditsUsed", "monthlyUsed");
  const monthlyRemaining = walletValue(wallet, "monthlyCreditsRemaining", "monthlyRemaining");
  const confirmedPurchases = purchases.filter((item) => item.paymentStatus === "confirmed");
  const confirmedValue = confirmedPurchases.reduce((acc, item) => acc + numberValue(item.price), 0);
  const active = activePackages(packages);
  const servicePackages = sortPlanPackages(active.filter(isPlanPackage));
  const topUpPackages = active.filter(isAddOnPackage).sort((a, b) => numberValue(a.price) - numberValue(b.price));
  const largestServiceCredit = servicePackages.reduce((acc, item) => Math.max(acc, numberValue(item.creditAmount)), 0);
  const totalTopUpCredit = topUpPackages.reduce((acc, item) => acc + numberValue(item.creditAmount), 0);
  const openPackageEditor = (item = null, type = "topup") => {
    if (item) {
      editPackage?.(item);
    } else {
      const isTopUp = type === "topup";
      resetPackageForm?.({
        name: "",
        description: "",
        creditAmount: isTopUp ? "10000" : "5000",
        price: isTopUp ? "250000" : "2500000",
        billingPeriod: isTopUp ? "one_time" : "monthly",
        maxWhatsAppSessions: "1",
        maxAiAgents: "1",
        maxHumanUsers: "1",
        isActive: true,
        isPopular: false,
      });
    }
    setPackageEditorOpen(true);
  };
  const closePackageEditor = () => {
    setPackageEditorOpen(false);
    resetPackageForm?.();
  };
  const submitPackageEditor = async (event) => {
    const saved = await createPackage?.(event);
    if (saved !== false) setPackageEditorOpen(false);
  };
  const deleteFromPackageEditor = async (item) => {
    const deleted = await deletePackage?.(item);
    if (deleted) setPackageEditorOpen(false);
  };

  return (
    <div className="wallet-page owner-billing-page">
      <section className="owner-billing-hero">
        <div>
          <h2>Billing Control</h2>
          <p>Kelola katalog paket layanan dan top-up credit. Status pembayaran mengikuti callback Midtrans setelah customer menyelesaikan checkout.</p>
        </div>
        <div className="owner-billing-total">
          <span>Katalog aktif</span>
          <strong>{formatNumber(servicePackages.length + topUpPackages.length)}</strong>
          <small>{servicePackages.length} paket layanan, {topUpPackages.length} top-up credit</small>
        </div>
      </section>

      <div className="wallet-metric-grid">
        <WalletMetric label="Paket Layanan" value={formatNumber(servicePackages.length)} meta={`${formatNumber(largestServiceCredit)} kredit bulanan terbesar`} tone="blue" />
        <WalletMetric label="Top-up Credit" value={formatNumber(topUpPackages.length)} meta={`${formatNumber(totalTopUpCredit)} total kredit tambahan`} tone="purple" />
        <WalletMetric label="Credits Available" value={formatNumber(monthlyRemaining)} meta={`${formatNumber(monthlyUsed)} dari ${formatNumber(monthlyLimit)} terpakai`} tone="green" />
        <WalletMetric label="Confirmed Value" value={formatCurrencyIDR(confirmedValue)} meta={`${confirmedPurchases.length} transaksi otomatis terkonfirmasi`} tone="orange" />
      </div>

      <section className="owner-annual-discount-card">
        <div className="owner-annual-discount-copy">
          <span className="badge blue">Harga landing page</span>
          <h3>Diskon Paket Tahunan</h3>
          <p>Atur persentase hemat untuk harga tahunan yang tampil pada section pricing publik.</p>
        </div>
        <form className="owner-annual-discount-form" onSubmit={savePricing}>
          <label>
            <span>Diskon tahunan (%)</span>
            <input
              className="form-input"
              type="number"
              min="0"
              max="99.99"
              step="0.01"
              value={pricingForm?.annualDiscountPercent ?? "10"}
              onChange={(event) => setPricingForm?.((current) => ({ ...current, annualDiscountPercent: event.target.value }))}
            />
          </label>
          <button className="btn btn-primary" type="submit" disabled={busyKey === "pricing"}>
            {busyKey === "pricing" ? "Menyimpan..." : "Simpan Diskon"}
          </button>
        </form>
      </section>

      <div className="owner-billing-layout">
        <main className="wallet-main">
          <div className="owner-catalog-panel">
            <div className="owner-catalog-panel-head">
              <div>
                <div className="wallet-panel-title">Katalog Paket Aktif</div>
                <div className="wallet-panel-subtitle">Paket layanan adalah subscription bulanan. Top-up credit adalah pembelian sekali bayar untuk menambah kredit.</div>
              </div>
              <span className="badge green">Pembayaran gateway</span>
            </div>
            <OwnerCatalogSection
              title="Paket Layanan"
              subtitle="Menentukan limit utama organisasi: kredit bulanan, WhatsApp, AI agent, dan user."
              packages={servicePackages}
              type="service"
              emptyText="Belum ada paket layanan aktif."
              canManage={canManagePackages}
              onEdit={(item) => openPackageEditor(item, "service")}
              onDelete={deletePackage}
              busyKey={busyKey}
            />
            <OwnerCatalogSection
              title="Top-up Credit"
              subtitle="Credit tambahan sekali beli. Tidak mengubah limit paket layanan yang sedang aktif."
              packages={topUpPackages}
              type="topup"
              emptyText="Belum ada top-up credit aktif."
              canManage={canManagePackages}
              onAdd={() => openPackageEditor(null, "topup")}
              onEdit={(item) => openPackageEditor(item, "topup")}
              onDelete={deletePackage}
              busyKey={busyKey}
            />
          </div>
          <PurchaseHistory purchases={purchases} cancelPurchase={cancelPurchase} busyKey={busyKey} readOnly />
        </main>

        <aside className="wallet-side">
          <div className="wallet-panel">
            <div className="wallet-panel-title">Buat Paket Layanan</div>
            <div className="wallet-panel-subtitle">Paket ini menjadi subscription bulanan dan menentukan limit utama organisasi.</div>
            {canManagePackages ? (
              <CreatePackageForm packageForm={packageForm} setPackageForm={setPackageForm} createPackage={createPackage} busyKey={busyKey} />
            ) : null}
          </div>
          <div className="wallet-panel owner-payment-note">
            <div className="wallet-panel-title">Alur Pembayaran</div>
            <div className="wallet-panel-subtitle">Midtrans mengirim status transaksi ke sistem. Setelah status paid/settlement masuk, paket aktif atau kredit top-up bertambah.</div>
            <div className="owner-payment-note-list">
              <span>Paket layanan: subscription bulanan dan limit utama.</span>
              <span>Top-up credit: kredit tambahan sekali bayar.</span>
              <span>Tidak ada antrean purchase request manual.</span>
            </div>
          </div>
        </aside>
      </div>
      {packageEditorOpen ? (
        <PackageEditorModal
          packageForm={packageForm}
          setPackageForm={setPackageForm}
          onSubmit={submitPackageEditor}
          onDelete={deleteFromPackageEditor}
          onClose={closePackageEditor}
          busyKey={busyKey}
        />
      ) : null}
    </div>
  );
}

export function WalletView({
  role,
  wallet,
  billingPlan,
  packages,
  pricingForm,
  setPricingForm,
  savePricing,
  packageForm,
  setPackageForm,
  createPackage,
  editPackage,
  deletePackage,
  resetPackageForm,
  canManagePackages,
  purchases,
  purchaseForm,
  setPurchaseForm,
  submitPurchaseRequest,
  cancelPurchase,
  canRequestPurchases,
  busyKey,
}) {
  if (role === "owner") {
    return (
      <OwnerBillingView
        wallet={wallet}
        packages={packages}
        pricingForm={pricingForm}
        setPricingForm={setPricingForm}
        savePricing={savePricing}
        packageForm={packageForm}
        setPackageForm={setPackageForm}
        createPackage={createPackage}
        editPackage={editPackage}
        deletePackage={deletePackage}
        resetPackageForm={resetPackageForm}
        canManagePackages={canManagePackages}
        purchases={purchases}
        cancelPurchase={cancelPurchase}
        busyKey={busyKey}
      />
    );
  }

  return (
    <UserUpgradeView
      wallet={wallet}
      billingPlan={billingPlan}
      packages={packages}
      purchases={purchases}
      purchaseForm={purchaseForm}
      setPurchaseForm={setPurchaseForm}
      submitPurchaseRequest={submitPurchaseRequest}
      cancelPurchase={cancelPurchase}
      canRequestPurchases={canRequestPurchases}
      busyKey={busyKey}
    />
  );
}

export function PurchaseView() {
  return null;
}
