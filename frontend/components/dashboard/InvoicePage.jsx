"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiBase, authStorageKey, formatCurrencyIDR, formatDateTime, legacyAuthStorageKey, readAndMigrateStorageValue } from "../../lib/dashboard-core";

const paymentMethodLabels = {
  bank_transfer: "Virtual Account",
  qris: "QRIS",
  gopay: "GoPay",
  shopeepay: "ShopeePay",
  credit_card: "Kartu",
};

function readStoredAuth() {
  if (typeof window === "undefined") return null;
  try {
    return JSON.parse(readAndMigrateStorageValue(window.localStorage, authStorageKey, legacyAuthStorageKey) || "null");
  } catch {
    return null;
  }
}

function formatDateOnly(value) {
  if (!value) return "-";
  try {
    return new Intl.DateTimeFormat("id-ID", { day: "2-digit", month: "long", year: "numeric" }).format(new Date(value));
  } catch {
    return formatDateTime(value);
  }
}

function billingPeriodLabel(value) {
  const period = String(value || "").toLowerCase();
  if (period === "one_time") return "Top-up add-on";
  if (period === "annual") return "Paket tahunan";
  return "Paket bulanan";
}

function paymentMethodLabel(value) {
  return paymentMethodLabels[String(value || "").toLowerCase()] || "Pembayaran online";
}

function invoiceErrorMessage(payload, status) {
  const message = String(payload?.error || "").trim();
  if (message) return message;
  if (status === 404) return "Invoice tidak ditemukan.";
  if (status === 409) return "Invoice tersedia setelah pembayaran berhasil.";
  return "Invoice belum bisa dimuat.";
}

export default function InvoicePage() {
  const params = useParams();
  const router = useRouter();
  const purchaseId = params?.purchaseId ? String(params.purchaseId) : "";
  const [invoice, setInvoice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!purchaseId) return;
    const auth = readStoredAuth();
    if (!auth?.token) {
      router.replace(`/login?next=${encodeURIComponent(window.location.pathname)}`);
      return;
    }

    const controller = new AbortController();
    async function loadInvoice() {
      setLoading(true);
      setError("");
      try {
        const response = await fetch(`${apiBase}/api/billing/purchases/${encodeURIComponent(purchaseId)}/invoice`, {
          headers: { Authorization: `Bearer ${auth.token}` },
          signal: controller.signal,
        });
        let payload = {};
        try {
          payload = await response.json();
        } catch {}
        if (response.status === 401) {
          router.replace(`/login?next=${encodeURIComponent(window.location.pathname)}`);
          return;
        }
        if (!response.ok) {
          throw new Error(invoiceErrorMessage(payload, response.status));
        }
        setInvoice(payload.invoice || null);
      } catch (err) {
        if (err.name !== "AbortError") setError(err.message || "Invoice belum bisa dimuat.");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }

    loadInvoice();
    return () => controller.abort();
  }, [purchaseId, router]);

  const lineItems = invoice?.lineItems || [];
  const purchase = invoice?.purchase || {};
  const billToName = useMemo(() => {
    const name = String(invoice?.billTo?.name || "").trim();
    return name || "Pelanggan Oneflow.id";
  }, [invoice]);

  return (
    <main className="invoice-page-shell">
      <div className="invoice-actions">
        <a className="btn btn-secondary" href="/dashboard/upgrade">Kembali ke Billing</a>
        <button className="btn btn-primary" type="button" onClick={() => window.print()} disabled={!invoice}>Cetak Invoice</button>
      </div>

      {loading ? (
        <section className="invoice-state">
          <strong>Memuat invoice</strong>
          <span>Mohon tunggu sebentar.</span>
        </section>
      ) : error ? (
        <section className="invoice-state invoice-state-error">
          <strong>Invoice belum tersedia</strong>
          <span>{error}</span>
          <a className="btn btn-secondary btn-sm" href="/dashboard/upgrade">Lihat riwayat pembelian</a>
        </section>
      ) : invoice ? (
        <article className="invoice-document">
          <header className="invoice-header">
            <div className="invoice-brand-lockup">
              <img className="invoice-logo" src="/brand/oneflow-main-logo.png" alt="Oneflow.id - AI Chat Automation & Business Assistant" />
            </div>
            <div className="invoice-title-block">
              <span className="invoice-status-paid">Lunas</span>
              <h1>Invoice</h1>
              <strong>{invoice.number}</strong>
            </div>
          </header>

          <section className="invoice-meta-grid">
            <div>
              <span>Ditagihkan kepada</span>
              <strong>{billToName}</strong>
              {invoice.billTo?.slug ? <small>{invoice.billTo.slug}</small> : null}
            </div>
            <div>
              <span>Tanggal invoice</span>
              <strong>{formatDateOnly(invoice.issuedAt)}</strong>
              <small>{formatDateTime(invoice.issuedAt)}</small>
            </div>
            <div>
              <span>Metode pembayaran</span>
              <strong>{paymentMethodLabel(purchase.paymentMethod)}</strong>
              {purchase.midtransOrderId ? <small>Order {purchase.midtransOrderId}</small> : null}
            </div>
            <div>
              <span>Masa aktif</span>
              <strong>{purchase.activeUntil ? formatDateOnly(purchase.activeUntil) : "Periode berjalan"}</strong>
              {purchase.activeFrom ? <small>Mulai {formatDateOnly(purchase.activeFrom)}</small> : null}
            </div>
          </section>

          <section className="invoice-table-wrap">
            <table className="invoice-table">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Tipe</th>
                  <th>Kredit</th>
                  <th>Jumlah</th>
                </tr>
              </thead>
              <tbody>
                {lineItems.map((item, index) => (
                  <tr key={`${item.description}-${index}`}>
                    <td>
                      <strong>{item.description}</strong>
                      {index === 0 ? (
                        <small>{[purchase.planKey || "Oneflow.id", item.creditAmount ? `${item.creditAmount.toLocaleString("id-ID")} Cr` : ""].filter(Boolean).join(" - ")}</small>
                      ) : null}
                    </td>
                    <td>{index === 0 ? billingPeriodLabel(purchase.billingPeriod) : "Pembayaran"}</td>
                    <td>{item.creditAmount ? `${item.creditAmount.toLocaleString("id-ID")} Cr` : "-"}</td>
                    <td>{formatCurrencyIDR(item.amount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section className="invoice-summary">
            <div className="invoice-note">
              <strong>Terima kasih.</strong>
              <span>Invoice ini diterbitkan otomatis setelah pembayaran berhasil dan dapat dicetak sebagai arsip pembelian.</span>
            </div>
            <div className="invoice-total-card">
              <div><span>Subtotal</span><strong>{formatCurrencyIDR(invoice.subtotal)}</strong></div>
              {invoice.paymentFee > 0 ? <div><span>Biaya pembayaran</span><strong>{formatCurrencyIDR(invoice.paymentFee)}</strong></div> : null}
              <div className="invoice-total-row"><span>Total dibayar</span><strong>{formatCurrencyIDR(invoice.total)}</strong></div>
            </div>
          </section>
        </article>
      ) : null}
    </main>
  );
}
