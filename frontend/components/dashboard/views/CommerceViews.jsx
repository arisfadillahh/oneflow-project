import { useEffect, useMemo, useState } from "react";
import { Icons, formatCurrencyIDR, formatDateTime } from "../../../lib/dashboard-core";

function orderStatusLabel(status) {
  if (status === "awaiting_confirmation") return "Menunggu konfirmasi";
  if (status === "confirmed") return "Terkonfirmasi";
  if (status === "cancelled") return "Batal";
  return "Draf";
}

function bookingStatusLabel(status) {
  if (status === "confirmed") return "Terkonfirmasi";
  if (status === "cancelled") return "Batal";
  if (status === "completed") return "Selesai";
  if (status === "draft") return "Draf";
  return "Terjadwal";
}

function recordStatusLabel(status) {
  if (status === "active") return "Aktif";
  if (status === "inactive") return "Nonaktif";
  if (status === "archived") return "Arsip";
  return status || "-";
}

function orderSourceLabel(item) {
  if (String(item.notes || "").toLowerCase().includes("playground")) return "Test dari Playground";
  if (item.source === "ai_tool") return "Dari AI";
  if (item.source === "dashboard") return "Manual";
  return item.source || "Manual";
}

function orderLineSummary(items = []) {
  if (!items.length) return "Belum ada item.";
  return items.map((line) => `${line.quantity || 0}x ${line.productName || "Produk"}`).join(", ");
}

const ORDER_FULFILLMENT_OPTIONS = [
  { value: "pickup", label: "Ambil di tempat", requiresAddress: false },
  { value: "delivery", label: "Diantar kurir lokal", requiresAddress: true },
  { value: "shipping", label: "Dikirim ekspedisi", requiresAddress: true },
  { value: "digital", label: "Online / digital", requiresAddress: false },
  { value: "onsite_service", label: "Layanan ke alamat pelanggan", requiresAddress: true },
];

function enabledFulfillmentOptions(settings) {
  const enabled = Array.isArray(settings?.enabledFulfillmentTypes) && settings.enabledFulfillmentTypes.length
    ? settings.enabledFulfillmentTypes
    : ORDER_FULFILLMENT_OPTIONS.map((item) => item.value);
  const options = ORDER_FULFILLMENT_OPTIONS.filter((item) => enabled.includes(item.value));
  return options.length ? options : ORDER_FULFILLMENT_OPTIONS;
}

function fulfillmentNeedsAddress(type) {
  return ORDER_FULFILLMENT_OPTIONS.find((item) => item.value === type)?.requiresAddress || false;
}

function bookingLocationLabel(type) {
  if (type === "online") return "Online";
  if (type === "customer_address") return "Alamat customer";
  return "Lokasi bisnis";
}

function stageTypeLabel(type) {
  if (type === "completed") return "Selesai";
  if (type === "cancelled") return "Batal";
  return "Berjalan";
}

function stageTypeBadgeClass(type) {
  if (type === "completed") return "green";
  if (type === "cancelled") return "red";
  return "blue";
}

function availabilitySummary(availability) {
  const days = Array.isArray(availability?.days) ? availability.days.join(", ") : "1, 2, 3, 4, 5";
  const start = availability?.start || "09:00";
  const end = availability?.end || "17:00";
  return `Hari ${days} - ${start}-${end}`;
}

function aiModeLabel(mode) {
  if (mode === "read") return "AI baca data";
  if (mode === "draft") return "AI bisa buat draft";
  if (mode === "action") return "AI aksi langsung";
  return "AI mati";
}

function toolAIBannerText(toolState, surface) {
  const mode = toolState?.aiMode || "off";
  if (mode === "off") return "AI belum memakai data halaman ini. Aktifkan izin AI dari Alat Bisnis jika ingin AI membantu customer.";
  if (surface === "products") {
    if (mode === "read") return "AI boleh membaca produk aktif, harga, dan stok tersedia. AI tidak mengubah harga atau stok.";
    return "AI boleh membaca katalog dan membuat draft pesanan dari chat. Stok tetap berubah hanya setelah pesanan dikonfirmasi.";
  }
  if (surface === "orders") {
    if (mode === "read") return "AI hanya boleh menjawab katalog dan stok. Draf pesanan tetap dibuat manual dari dashboard.";
    return "AI boleh membuat draft pesanan dari chat customer. Admin tetap confirm untuk reserve stok.";
  }
  if (surface === "booking") {
    if (mode === "read") return "AI boleh menjawab layanan dan jam tersedia. Appointment tetap dibuat manual.";
    if (mode === "draft") return "AI boleh membuat draf jadwal booking. Admin bisa konfirmasi, ubah, atau batalkan dari dashboard.";
    return "AI boleh membuat jadwal booking jika data pelanggan dan waktu lengkap.";
  }
  return "AI sudah diizinkan memakai alat ini.";
}

function ToolAIBanner({ toolState, surface }) {
  return (
    <div className="business-ai-banner">
      {Icons.robot}
      <div>
        <strong>{aiModeLabel(toolState?.aiMode)}</strong>
        <span>{toolAIBannerText(toolState, surface)}</span>
      </div>
    </div>
  );
}

export function CommerceProductsView({
  products = [],
  productForm,
  setProductForm,
  onSubmitProduct,
  onEditProduct,
  onDeleteProduct,
  onResetProduct,
  onOpenOrders,
  canManage = false,
  busyKey = "",
  toolState,
}) {
  const activeProductCount = products.filter((item) => item.status === "active").length;
  const lowStockCount = products.filter((item) => item.status === "active" && item.lowStock).length;
  const reservedStockCount = products.reduce((total, item) => total + Number(item.reservedQuantity || 0), 0);
  const [isProductModalOpen, setProductModalOpen] = useState(false);

  const openNewProduct = () => {
    onResetProduct?.();
    setProductModalOpen(true);
  };
  const openEditProduct = (item) => {
    onEditProduct?.(item);
    setProductModalOpen(true);
  };
  const closeProductEditor = () => {
    setProductModalOpen(false);
    onResetProduct?.();
  };
  const handleProductSubmit = async (event) => {
    const saved = await onSubmitProduct?.(event);
    if (saved !== false) setProductModalOpen(false);
  };
  const handleDeleteProduct = async (item) => {
    const deleted = await onDeleteProduct?.(item);
    if (deleted) setProductModalOpen(false);
  };

  return (
    <div className="business-workspace-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Produk & Stok</div>
          <div className="page-desc">Katalog, stok, dan dasar pembuatan pesanan. Izin AI mengikuti mode yang dipilih di Alat Bisnis.</div>
        </div>
        <div className="page-actions">
          {canManage ? (
            <button className="btn btn-primary btn-sm" type="button" onClick={openNewProduct}>
              {Icons.plus}
              Tambah Produk
            </button>
          ) : null}
          <button className="btn btn-secondary btn-sm" type="button" onClick={onOpenOrders}>
            Lihat Pesanan
          </button>
        </div>
      </div>

      <ToolAIBanner toolState={toolState} surface="products" />

      <div className="business-plugin-summary" aria-label="Ringkasan produk dan stok">
        <div>
          <strong>{products.length}</strong>
          <span>Total produk</span>
        </div>
        <div>
          <strong>{activeProductCount}</strong>
          <span>Aktif dijual</span>
        </div>
        <div>
          <strong>{lowStockCount}</strong>
          <span>Perlu restok</span>
        </div>
        <div>
          <strong>{reservedStockCount}</strong>
          <span>Stok reserved</span>
        </div>
      </div>

      <section className="card business-section-card">
        <div className="card-header">
          <div>
            <div className="card-title">Katalog Produk</div>
            <div className="card-subtitle">Owner dan admin bisa menyesuaikan produk, harga, dan stok kapan pun.</div>
          </div>
        </div>
        {!canManage ? (
          <div className="business-muted-panel">Role ini bisa melihat produk dan stok, tapi perubahan katalog dilakukan oleh admin.</div>
        ) : null}
        <div className="table-wrap business-table-wrap">
          <table>
            <thead><tr><th>Produk</th><th>Harga</th><th>Stok</th><th>Restok</th><th>Status</th><th>Aksi</th></tr></thead>
            <tbody>
              {products.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.name}</strong><div className="text-xs text-muted">{item.sku || "-"}</div></td>
                  <td>{formatCurrencyIDR(item.unitPrice || 0)}</td>
                  <td>{item.availableStock} tersedia <div className="text-xs text-muted">{item.reservedQuantity || 0} reserved</div></td>
                  <td>
                    {item.lowStock ? <span className="badge orange">Perlu restok</span> : <span className="text-xs text-muted">Aman</span>}
                    <div className="text-xs text-muted">Batas {item.lowStockThreshold || 0}</div>
                  </td>
                  <td><span className={`badge ${item.status === "active" ? "green" : "gray"}`}>{recordStatusLabel(item.status)}</span></td>
                  <td>
                    {canManage ? (
                      <div className="table-actions business-product-actions">
                        <button className="btn btn-secondary btn-sm" type="button" onClick={() => openEditProduct(item)} aria-label={`Edit ${item.name}`}>
                          {Icons.edit}
                          Edit
                        </button>
                        <button className="btn btn-danger btn-sm" type="button" onClick={() => handleDeleteProduct(item)} disabled={busyKey === `commerce-product-delete-${item.id}`} aria-label={`Hapus ${item.name}`}>
                          {Icons.trash}
                          Hapus
                        </button>
                      </div>
                    ) : "-"}
                  </td>
                </tr>
              ))}
              {!products.length ? <tr><td colSpan={6} className="knowledge-empty-cell">Belum ada produk.</td></tr> : null}
            </tbody>
          </table>
        </div>
      </section>

      {isProductModalOpen ? (
        <div
          className="modal-backdrop billing-modal-backdrop business-product-editor-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) closeProductEditor();
          }}
        >
          <div className="modal-card billing-modal-card business-product-editor-modal" role="dialog" aria-modal="true" aria-labelledby="product-editor-title" onMouseDown={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <div>
                <h3 id="product-editor-title">{productForm.id ? "Edit produk" : "Tambah produk"}</h3>
                <p>Kelola harga, stok tersedia, dan status katalog yang dipakai agent saat menjawab customer.</p>
              </div>
              <button className="modal-close" type="button" onClick={closeProductEditor} aria-label="Tutup">{Icons.close}</button>
            </div>
            <form id="business-product-editor-form" className="billing-modal-body business-product-editor-form" onSubmit={handleProductSubmit}>
              <label className="business-field">
                <span>Nama produk</span>
                <input className="form-input" required value={productForm.name} onChange={(event) => setProductForm?.({ ...productForm, name: event.target.value })} placeholder="Nama produk" />
              </label>
              <label className="business-field">
                <span>SKU</span>
                <input className="form-input" value={productForm.sku} onChange={(event) => setProductForm?.({ ...productForm, sku: event.target.value })} placeholder="Contoh: SKU-001" />
              </label>
              <label className="business-field">
                <span>Harga jual (Rp)</span>
                <input className="form-input" type="number" min="0" value={productForm.unitPrice} onChange={(event) => setProductForm?.({ ...productForm, unitPrice: event.target.value })} placeholder="Contoh: 18000" />
              </label>
              <label className="business-field">
                <span>Stok tersedia</span>
                <input className="form-input" type="number" min="0" value={productForm.stockQuantity} onChange={(event) => setProductForm?.({ ...productForm, stockQuantity: event.target.value })} placeholder="Contoh: 12" />
              </label>
              <label className="business-field">
                <span>Alert restok</span>
                <input className="form-input" type="number" min="0" value={productForm.lowStockThreshold} onChange={(event) => setProductForm?.({ ...productForm, lowStockThreshold: event.target.value })} placeholder="Contoh: 5" />
              </label>
              <label className="business-field">
                <span>Status</span>
                <select className="form-select" value={productForm.status} onChange={(event) => setProductForm?.({ ...productForm, status: event.target.value })}>
                  <option value="active">Aktif</option>
                  <option value="inactive">Nonaktif</option>
                  <option value="archived">Arsip</option>
                </select>
              </label>
              <label className="business-field business-form-wide">
                <span>Deskripsi</span>
                <textarea className="form-textarea" rows={3} value={productForm.description} onChange={(event) => setProductForm?.({ ...productForm, description: event.target.value })} placeholder="Deskripsi singkat yang aman dibaca AI" />
              </label>
            </form>
            <div className="modal-actions business-product-editor-actions">
              {productForm.id ? (
                <button className="btn btn-danger" type="button" onClick={() => handleDeleteProduct(productForm)} disabled={busyKey === `commerce-product-delete-${productForm.id}`}>
                  {Icons.trash}
                  Hapus
                </button>
              ) : null}
              <button className="btn btn-secondary" type="button" onClick={closeProductEditor}>Batal</button>
              <button className="btn btn-primary" type="submit" form="business-product-editor-form" disabled={busyKey === "commerce-product"}>
                {productForm.id ? "Simpan perubahan" : "Tambah produk"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function CommerceOrdersView({
  products = [],
  orderDrafts = [],
  orderPipelines = [],
  orderStages = [],
  orderSettings,
  orderDraftForm,
  setOrderDraftForm,
  onSubmitOrderDraft,
  onUpdateOrderStatus,
  onUpdateOrderDraft,
  onSaveOrderStage,
  onDeleteOrderStage,
  onUpdateOrderSettings,
  onOpenProducts,
  canManagePipeline = false,
  busyKey = "",
  toolState,
}) {
  const activeProducts = products.filter((item) => item.status === "active");
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [isOrderFormOpen, setOrderFormOpen] = useState(false);
  const [stageDrafts, setStageDrafts] = useState({});
  const [fulfillmentDraft, setFulfillmentDraft] = useState(() => enabledFulfillmentOptions(orderSettings).map((item) => item.value));
  const [newStageName, setNewStageName] = useState("");
  const items = orderDraftForm.items?.length ? orderDraftForm.items : [{ productId: "", quantity: "1" }];
  const activePipeline = orderPipelines.find((item) => item.isDefault) || orderPipelines[0] || null;
  const orderedStages = useMemo(() => (
    [...orderStages].sort((left, right) => Number(left.position || 0) - Number(right.position || 0))
  ), [orderStages]);
  const availableFulfillmentOptions = useMemo(() => enabledFulfillmentOptions(orderSettings), [orderSettings]);
  const defaultFulfillmentType = availableFulfillmentOptions[0]?.value || "pickup";
  const defaultStageId = orderedStages.find((stage) => stage.type === "open")?.id || orderedStages[0]?.id || "";
  const draftTotal = items.reduce((total, line) => {
    const product = activeProducts.find((item) => item.id === line.productId);
    return total + (product ? Number(product.unitPrice || 0) * Math.max(1, Number(line.quantity || 1)) : 0);
  }, 0);
  const lowStockProducts = products.filter((item) => item.status === "active" && item.lowStock);
  const awaitingOrderCount = orderDrafts.filter((item) => item.status === "draft" || item.status === "awaiting_confirmation").length;
  const confirmedOrderCount = orderDrafts.filter((item) => item.status === "confirmed").length;
  const orderTotalAmount = orderDrafts.reduce((total, item) => total + Number(item.totalAmount || 0), 0);
  const pipelineBuckets = useMemo(() => {
    const buckets = Object.fromEntries(orderedStages.map((stage) => [stage.id, []]));
    const backlog = [];
    orderDrafts.forEach((item) => {
      if (item.stageId && buckets[item.stageId]) {
        buckets[item.stageId].push(item);
        return;
      }
      backlog.push(item);
    });
    return { buckets, backlog };
  }, [orderDrafts, orderedStages]);
  const boardColumnCount = orderedStages.length + (pipelineBuckets.backlog.length ? 1 : 0);

  useEffect(() => {
    setStageDrafts(Object.fromEntries(orderedStages.map((stage) => [stage.id, {
      name: stage.name || "",
      position: String(stage.position || 0),
      type: stage.type || "open",
      color: stage.color || "#2196F3",
    }])));
  }, [orderedStages]);

  useEffect(() => {
    if (!orderDraftForm.stageId && defaultStageId) {
      setOrderDraftForm?.((current) => ({ ...current, stageId: current.stageId || defaultStageId }));
    }
  }, [defaultStageId, orderDraftForm.stageId, setOrderDraftForm]);

  useEffect(() => {
    setFulfillmentDraft(availableFulfillmentOptions.map((item) => item.value));
  }, [availableFulfillmentOptions]);

  useEffect(() => {
    if (!availableFulfillmentOptions.some((item) => item.value === orderDraftForm.fulfillmentType)) {
      setOrderDraftForm?.((current) => ({ ...current, fulfillmentType: defaultFulfillmentType }));
    }
  }, [availableFulfillmentOptions, defaultFulfillmentType, orderDraftForm.fulfillmentType, setOrderDraftForm]);

  const selectedOrderLive = useMemo(() => (
    selectedOrder ? orderDrafts.find((item) => item.id === selectedOrder.id) || selectedOrder : null
  ), [orderDrafts, selectedOrder]);

  function updateLine(index, patch) {
    setOrderDraftForm?.((current) => {
      const nextItems = (current.items?.length ? current.items : [{ productId: "", quantity: "1" }]).map((item, itemIndex) => (
        itemIndex === index ? { ...item, ...patch } : item
      ));
      return { ...current, items: nextItems };
    });
  }

  function addLine() {
    setOrderDraftForm?.((current) => ({ ...current, items: [...(current.items || []), { productId: "", quantity: "1" }] }));
  }

  function removeLine(index) {
    setOrderDraftForm?.((current) => {
      const nextItems = (current.items || []).filter((_, itemIndex) => itemIndex !== index);
      return { ...current, items: nextItems.length ? nextItems : [{ productId: "", quantity: "1" }] };
    });
  }

  async function saveStage(stage, patch = {}) {
    const draft = { ...(stageDrafts[stage.id] || {}), ...patch };
    await onSaveOrderStage?.(stage.id, {
      name: draft.name,
      position: Number(draft.position || stage.position || 10),
      type: draft.type || stage.type || "open",
      color: draft.color || stage.color || "#2196F3",
    });
  }

  async function moveStage(stage, direction) {
    const currentIndex = orderedStages.findIndex((item) => item.id === stage.id);
    const targetStage = orderedStages[currentIndex + direction];
    if (!targetStage) return;
    await saveStage(stage, { position: targetStage.position || ((currentIndex + direction + 1) * 10) });
    await saveStage(targetStage, { position: stage.position || ((currentIndex + 1) * 10) });
  }

  function deleteStage(stage) {
    onDeleteOrderStage?.(stage.id);
  }

  async function addStage(event) {
    event.preventDefault();
    if (!newStageName.trim() || !activePipeline?.id) return;
    await onSaveOrderStage?.("", {
      pipelineId: activePipeline.id,
      name: newStageName,
      position: ((orderStages[orderStages.length - 1]?.position || 0) + 10),
      type: "open",
      color: "#2196F3",
    });
    setNewStageName("");
  }

  async function submitOrderDraft(event) {
    const result = await onSubmitOrderDraft?.(event);
    if (result !== false) setOrderFormOpen(false);
  }

  function toggleFulfillmentType(type) {
    setFulfillmentDraft((current) => {
      if (current.includes(type)) {
        const next = current.filter((item) => item !== type);
        return next.length ? next : current;
      }
      return [...current, type];
    });
  }

  return (
    <div className="business-workspace-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Pesanan</div>
          <div className="page-desc">Buat dan pantau draf pesanan dari produk aktif.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary btn-sm" type="button" onClick={onOpenProducts}>
            Produk & Stok
          </button>
          <button className={`btn ${isOrderFormOpen ? "btn-secondary" : "btn-primary"} btn-sm`} type="button" disabled={!activeProducts.length} title={!activeProducts.length ? "Tambahkan produk aktif dulu" : undefined} onClick={() => setOrderFormOpen((current) => !current)}>
            {isOrderFormOpen ? Icons.close : Icons.plus} {isOrderFormOpen ? "Tutup Form" : "Pesanan Baru"}
          </button>
        </div>
      </div>

      <ToolAIBanner toolState={toolState} surface="orders" />

      <div className="business-plugin-summary" aria-label="Ringkasan pesanan">
        <div>
          <strong>{orderDrafts.length}</strong>
          <span>Total draf</span>
        </div>
        <div>
          <strong>{awaitingOrderCount}</strong>
          <span>Perlu follow-up</span>
        </div>
        <div>
          <strong>{confirmedOrderCount}</strong>
          <span>Terkonfirmasi</span>
        </div>
        <div>
          <strong>{formatCurrencyIDR(orderTotalAmount)}</strong>
          <span>Nilai pesanan</span>
        </div>
      </div>

      {lowStockProducts.length ? (
        <div className="business-alert-panel">
          <strong>{lowStockProducts.length} produk perlu restok</strong>
          <span>{lowStockProducts.slice(0, 3).map((item) => `${item.name} (${item.availableStock})`).join(", ")}</span>
        </div>
      ) : null}

      {!activeProducts.length ? (
        <div className="business-alert-panel business-empty-action">
          <span>Belum ada produk aktif untuk dibuat pesanan. Tambahkan produk aktif dulu supaya form pesanan tidak kosong.</span>
          <button className="btn btn-secondary btn-sm" type="button" onClick={onOpenProducts}>Buka Produk & Stok</button>
        </div>
      ) : null}

      <section className="business-pipeline-focus">
        <div className="card business-section-card">
          <div className="card-header">
            <div>
              <div className="card-title">Alur Pesanan</div>
              <div className="card-subtitle">Pantau tahap, status, dan total draf pesanan dari satu board.</div>
            </div>
          </div>
          <div className="business-order-pipeline">
            {boardColumnCount ? (
              <div className="business-pipeline-board" style={{ "--business-stage-count": boardColumnCount }}>
                {orderedStages.map((stage) => {
                  const stageOrders = pipelineBuckets.buckets[stage.id] || [];
                  const stageTotal = stageOrders.reduce((total, item) => total + Number(item.totalAmount || 0), 0);
                  return (
                    <article className="business-pipeline-column" style={{ "--stage-color": stage.color || "#2196F3" }} key={stage.id}>
                      <div className="business-pipeline-head">
                        <div>
                          <div className="business-pipeline-title"><span />{stage.name}</div>
                          <div className="business-pipeline-meta">{stageOrders.length} pesanan - {formatCurrencyIDR(stageTotal)}</div>
                        </div>
                        <span className={`badge ${stageTypeBadgeClass(stage.type)}`}>{stageTypeLabel(stage.type)}</span>
                      </div>
                      <div className="business-pipeline-card-stack">
                        {stageOrders.slice(0, 4).map((item) => (
                          <button className="business-pipeline-card" type="button" key={item.id} onClick={() => setSelectedOrder(item)}>
                            <strong>{item.customerName || item.customerPhone || "Pelanggan"}</strong>
                            <span>{formatCurrencyIDR(item.totalAmount || 0)}</span>
                            <small>{orderLineSummary(item.items || [])}</small>
                          </button>
                        ))}
                        {!stageOrders.length ? <div className="business-pipeline-empty">Kosong</div> : null}
                        {stageOrders.length > 4 ? <div className="business-pipeline-more">+{stageOrders.length - 4} lagi</div> : null}
                      </div>
                    </article>
                  );
                })}
                {pipelineBuckets.backlog.length ? (
                  <article className="business-pipeline-column business-pipeline-column-muted">
                    <div className="business-pipeline-head">
                      <div>
                        <div className="business-pipeline-title"><span />Tanpa tahap</div>
                        <div className="business-pipeline-meta">{pipelineBuckets.backlog.length} pesanan</div>
                      </div>
                    </div>
                    <div className="business-pipeline-card-stack">
                      {pipelineBuckets.backlog.slice(0, 4).map((item) => (
                        <button className="business-pipeline-card" type="button" key={item.id} onClick={() => setSelectedOrder(item)}>
                          <strong>{item.customerName || item.customerPhone || "Pelanggan"}</strong>
                          <span>{formatCurrencyIDR(item.totalAmount || 0)}</span>
                          <small>{orderLineSummary(item.items || [])}</small>
                        </button>
                      ))}
                      {pipelineBuckets.backlog.length > 4 ? <div className="business-pipeline-more">+{pipelineBuckets.backlog.length - 4} lagi</div> : null}
                    </div>
                  </article>
                ) : null}
              </div>
            ) : (
              <div className="business-muted-panel">Belum ada stage pipeline.</div>
            )}
          </div>
          {canManagePipeline ? (
            <>
              <details className="business-pipeline-settings">
                <summary>Atur cara terima pesanan</summary>
                <div className="business-fulfillment-settings">
                  {ORDER_FULFILLMENT_OPTIONS.map((option) => (
                    <label className="business-fulfillment-option" key={option.value}>
                      <input type="checkbox" checked={fulfillmentDraft.includes(option.value)} onChange={() => toggleFulfillmentType(option.value)} disabled={fulfillmentDraft.length <= 1 && fulfillmentDraft.includes(option.value)} />
                      <span>
                        <strong>{option.label}</strong>
                        <small>{option.requiresAddress ? "Butuh alamat pelanggan" : "Tanpa alamat pengiriman"}</small>
                      </span>
                    </label>
                  ))}
                </div>
                <div className="business-form-actions">
                  <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateOrderSettings?.(fulfillmentDraft)} disabled={busyKey === "commerce-order-settings"}>Simpan cara terima</button>
                </div>
              </details>
              <details className="business-pipeline-settings">
                <summary>Atur alur pesanan</summary>
                <div className="business-stage-editor-list">
                  {orderedStages.map((stage, index) => (
                    <div className="business-stage-editor-row" key={stage.id}>
                      <div className="business-stage-name-field">
                        <input className="form-input" value={stageDrafts[stage.id]?.name || ""} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), name: event.target.value } }))} />
                        <small>Urutan {index + 1}</small>
                      </div>
                      <select className="form-select" value={stageDrafts[stage.id]?.type || "open"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), type: event.target.value } }))}>
                        <option value="open">Berjalan</option>
                        <option value="completed">Selesai</option>
                        <option value="cancelled">Batal</option>
                      </select>
                      <input className="form-input business-color-input" type="color" value={stageDrafts[stage.id]?.color || "#2196F3"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), color: event.target.value } }))} />
                      <div className="business-stage-move-actions">
                        <button className="btn btn-secondary btn-sm btn-icon business-icon-up" type="button" title="Naik" aria-label={`Naikkan ${stage.name}`} onClick={() => moveStage(stage, -1)} disabled={index === 0 || busyKey.startsWith("commerce-order-stage-")}>{Icons.chevronDown}</button>
                        <button className="btn btn-secondary btn-sm btn-icon" type="button" title="Turun" aria-label={`Turunkan ${stage.name}`} onClick={() => moveStage(stage, 1)} disabled={index === orderedStages.length - 1 || busyKey.startsWith("commerce-order-stage-")}>{Icons.chevronDown}</button>
                      </div>
                      <button className="btn btn-secondary btn-sm" type="button" onClick={() => saveStage(stage)} disabled={busyKey === `commerce-order-stage-${stage.id}`}>Simpan</button>
                      <button className="btn btn-danger btn-sm" type="button" onClick={() => deleteStage(stage)} disabled={orderedStages.length <= 1 || busyKey === `commerce-order-stage-delete-${stage.id}`}>Hapus</button>
                    </div>
                  ))}
                </div>
                <form className="business-stage-add-row" onSubmit={addStage}>
                  <input className="form-input" value={newStageName} onChange={(event) => setNewStageName(event.target.value)} placeholder="Tahap baru" />
                  <button className="btn btn-primary btn-sm" type="submit" disabled={!newStageName.trim() || busyKey === "commerce-order-stage-new"}>Tambah tahap</button>
                </form>
              </details>
            </>
          ) : null}
        </div>
      </section>
      {isOrderFormOpen ? (
        <div className="business-detail-backdrop" role="presentation" onMouseDown={() => setOrderFormOpen(false)}>
          <aside className="business-detail-drawer business-order-form-drawer" role="dialog" aria-modal="true" aria-label="Pesanan Baru" onMouseDown={(event) => event.stopPropagation()}>
            <div className="business-detail-head">
              <div>
                <span>Draf Pesanan</span>
                <h3>Pesanan Baru</h3>
                <p>Isi dari produk aktif, lalu simpan sebagai draf.</p>
              </div>
              <button className="btn btn-secondary btn-sm" type="button" onClick={() => setOrderFormOpen(false)}>{Icons.close} Tutup</button>
            </div>

            {!activeProducts.length ? (
              <div className="business-muted-panel business-empty-action">
                <span>Belum ada produk aktif. Tambahkan produk dulu supaya pesanan bisa dibuat.</span>
                <button className="btn btn-primary btn-sm" type="button" onClick={onOpenProducts}>
                  Tambah Produk
                </button>
              </div>
            ) : (
              <form className="business-form-grid order-form" onSubmit={submitOrderDraft}>
                <label className="business-field">
                  <span>Nama pelanggan</span>
                  <input className="form-input" value={orderDraftForm.customerName} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, customerName: event.target.value })} placeholder="Nama pelanggan" />
                </label>
                <label className="business-field">
                  <span>No. WhatsApp</span>
                  <input className="form-input" inputMode="tel" value={orderDraftForm.customerPhone} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, customerPhone: event.target.value })} placeholder="628..." />
                </label>
                <label className="business-field">
                  <span>Tahap</span>
                  <select className="form-select" value={orderDraftForm.stageId || defaultStageId} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, stageId: event.target.value })}>
                    {orderedStages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}
                  </select>
                </label>
                <label className="business-field">
                  <span>Cara pelanggan menerima pesanan</span>
                  <select className="form-select" value={orderDraftForm.fulfillmentType || defaultFulfillmentType} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, fulfillmentType: event.target.value })}>
                    {availableFulfillmentOptions.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </label>
                <div className="business-line-items business-form-wide">
                  <div className="business-line-items-head">
                    <span>Item pesanan</span>
                    <button className="btn btn-secondary btn-sm" type="button" onClick={addLine}>Tambah item</button>
                  </div>
                  {items.map((line, index) => {
                    const product = activeProducts.find((item) => item.id === line.productId);
                    const quantity = Math.max(1, Number(line.quantity || 1));
                    return (
                      <div className="business-line-item-row" key={`${index}-${line.productId || "new"}`}>
                        <select className="form-select" required value={line.productId} onChange={(event) => updateLine(index, { productId: event.target.value })}>
                          <option value="">Pilih produk</option>
                          {activeProducts.map((item) => (
                            <option key={item.id} value={item.id}>{item.name} ({item.availableStock} stok)</option>
                          ))}
                        </select>
                        <input className="form-input" type="number" inputMode="numeric" min="1" value={line.quantity} onChange={(event) => updateLine(index, { quantity: event.target.value })} />
                        <span>{formatCurrencyIDR(product ? Number(product.unitPrice || 0) * quantity : 0)}</span>
                        <button className="btn btn-secondary btn-sm" type="button" onClick={() => removeLine(index)} disabled={items.length <= 1}>Hapus</button>
                      </div>
                    );
                  })}
                </div>
                {fulfillmentNeedsAddress(orderDraftForm.fulfillmentType) ? (
                  <>
                    <label className="business-field">
                      <span>Nama penerima</span>
                      <input className="form-input" value={orderDraftForm.recipientName} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, recipientName: event.target.value })} placeholder="Kosongkan jika sama" />
                    </label>
                    <label className="business-field">
                      <span>No. penerima</span>
                      <input className="form-input" inputMode="tel" value={orderDraftForm.recipientPhone} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, recipientPhone: event.target.value })} placeholder="Kosongkan jika sama" />
                    </label>
                    <label className="business-field business-form-wide">
                      <span>Alamat lengkap</span>
                      <input className="form-input" required value={orderDraftForm.addressLine} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, addressLine: event.target.value })} placeholder="Jalan, nomor, patokan" />
                    </label>
                    <label className="business-field">
                      <span>Area/kota</span>
                      <input className="form-input" value={orderDraftForm.addressArea} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, addressArea: event.target.value })} placeholder="Kecamatan/kota" />
                    </label>
                    <label className="business-field">
                      <span>Catatan alamat</span>
                      <input className="form-input" value={orderDraftForm.addressNotes} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, addressNotes: event.target.value })} placeholder="Patokan/kurir" />
                    </label>
                  </>
                ) : null}
                <label className="business-field business-form-wide">
                  <span>Catatan</span>
                  <input className="form-input" value={orderDraftForm.notes} onChange={(event) => setOrderDraftForm?.({ ...orderDraftForm, notes: event.target.value })} placeholder="Catatan pesanan" />
                </label>
                <div className="business-form-actions">
                  <span className="text-sm text-muted">{formatCurrencyIDR(draftTotal)}</span>
                  <button className="btn btn-primary btn-sm" type="submit" disabled={busyKey === "commerce-order-draft"}>
                    Buat Draf
                  </button>
                </div>
              </form>
            )}
          </aside>
        </div>
      ) : null}
      {selectedOrderLive ? (
        <OrderDetailDrawer
          item={selectedOrderLive}
          products={activeProducts}
          stages={orderStages}
          fulfillmentOptions={availableFulfillmentOptions}
          onClose={() => setSelectedOrder(null)}
          onUpdateOrderStatus={onUpdateOrderStatus}
          onUpdateOrderDraft={onUpdateOrderDraft}
          busyKey={busyKey}
        />
      ) : null}
    </div>
  );
}

function OrderDetailDrawer({ item, products = [], stages = [], fulfillmentOptions = ORDER_FULFILLMENT_OPTIONS, onClose, onUpdateOrderStatus, onUpdateOrderDraft, busyKey }) {
  const [draft, setDraft] = useState(() => ({
    stageId: item.stageId || "",
    notes: item.notes || "",
    fulfillmentType: item.fulfillmentType || "pickup",
    recipientName: item.recipientName || "",
    recipientPhone: item.recipientPhone || "",
    addressLine: item.addressLine || "",
    addressArea: item.addressArea || "",
    addressNotes: item.addressNotes || "",
  }));
  const [itemDrafts, setItemDrafts] = useState(() => (item.items || []).map((line) => ({ productId: line.productId || "", quantity: String(line.quantity || 1) })));
  const canEditItems = !["confirmed", "cancelled"].includes(item.status);
  const drawerFulfillmentOptions = useMemo(() => {
    if (fulfillmentOptions.some((option) => option.value === draft.fulfillmentType)) return fulfillmentOptions;
    const current = ORDER_FULFILLMENT_OPTIONS.find((option) => option.value === draft.fulfillmentType);
    return current ? [current, ...fulfillmentOptions] : fulfillmentOptions;
  }, [draft.fulfillmentType, fulfillmentOptions]);

  useEffect(() => {
    setDraft({
      stageId: item.stageId || "",
      notes: item.notes || "",
      fulfillmentType: item.fulfillmentType || "pickup",
      recipientName: item.recipientName || "",
      recipientPhone: item.recipientPhone || "",
      addressLine: item.addressLine || "",
      addressArea: item.addressArea || "",
      addressNotes: item.addressNotes || "",
    });
    setItemDrafts((item.items || []).map((line) => ({ productId: line.productId || "", quantity: String(line.quantity || 1) })));
  }, [item.id, item.updatedAt, item.stageId, item.notes, item.fulfillmentType, item.recipientName, item.recipientPhone, item.addressLine, item.addressArea, item.addressNotes]);

  function saveDraft() {
    const payload = { ...draft };
    if (canEditItems) {
      payload.items = itemDrafts.filter((line) => line.productId).map((line) => ({ productId: line.productId, quantity: Number(line.quantity || 1) }));
    }
    onUpdateOrderDraft?.(item.id, payload, { title: "Detail pesanan disimpan", message: "Stage, alamat, item, dan catatan pesanan sudah diperbarui." });
  }

  function updateItemDraft(index, patch) {
    setItemDrafts((current) => current.map((line, itemIndex) => itemIndex === index ? { ...line, ...patch } : line));
  }

  function addItemDraft() {
    setItemDrafts((current) => [...current, { productId: "", quantity: "1" }]);
  }

  function removeItemDraft(index) {
    setItemDrafts((current) => {
      const next = current.filter((_, itemIndex) => itemIndex !== index);
      return next.length ? next : [{ productId: "", quantity: "1" }];
    });
  }

  return (
    <div className="business-detail-backdrop" role="presentation" onMouseDown={onClose}>
      <aside className="business-detail-drawer" role="dialog" aria-modal="true" aria-label="Detail pesanan" onMouseDown={(event) => event.stopPropagation()}>
        <div className="business-detail-head">
          <div>
            <span>Detail Pesanan</span>
            <h3>{item.customerName || item.customerPhone || "Pelanggan"}</h3>
            <p>{formatCurrencyIDR(item.totalAmount || 0)} - {orderStatusLabel(item.status)} - {orderSourceLabel(item)}</p>
          </div>
          <button className="btn btn-secondary btn-sm" type="button" onClick={onClose}>Tutup</button>
        </div>

        <div className="business-detail-section">
          <h4>Identitas pemesan</h4>
          <div className="business-detail-grid">
            <div><span>Nama</span><strong>{item.customerName || "-"}</strong></div>
            <div><span>No. WhatsApp</span><strong>{item.customerPhone || "-"}</strong></div>
            <div><span>Sumber</span><strong>{orderSourceLabel(item)}</strong></div>
            <div><span>Dibuat</span><strong>{formatDateTime(item.createdAt)}</strong></div>
          </div>
        </div>

        <div className="business-detail-section">
          <h4>Pipeline & status</h4>
          <label className="business-field">
            <span>Tahap alur</span>
            <select className="form-select" value={draft.stageId} onChange={(event) => setDraft((current) => ({ ...current, stageId: event.target.value }))}>
              <option value="">Belum ada tahap</option>
              {stages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}
            </select>
          </label>
          <div className="business-order-actions">
            {item.status !== "confirmed" && item.status !== "cancelled" ? <button className="btn btn-primary btn-sm" type="button" onClick={() => onUpdateOrderStatus?.(item.id, "confirmed")}>Konfirmasi</button> : null}
            {item.status !== "cancelled" ? <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateOrderStatus?.(item.id, "cancelled")}>Batal</button> : null}
          </div>
        </div>

        <div className="business-detail-section">
          <h4>Item pesanan</h4>
          {canEditItems ? (
            <div className="business-line-items">
              <div className="business-line-items-head">
                <span>Item draf bisa diubah sebelum dikonfirmasi</span>
                <button className="btn btn-secondary btn-sm" type="button" onClick={addItemDraft}>Tambah item</button>
              </div>
              {itemDrafts.map((line, index) => {
                const product = products.find((productItem) => productItem.id === line.productId);
                const quantity = Math.max(1, Number(line.quantity || 1));
                return (
                  <div className="business-line-item-row" key={`${index}-${line.productId || "draft"}`}>
                    <select className="form-select" value={line.productId} onChange={(event) => updateItemDraft(index, { productId: event.target.value })}>
                      <option value="">Pilih produk</option>
                      {products.map((productItem) => <option key={productItem.id} value={productItem.id}>{productItem.name} ({productItem.availableStock} stok)</option>)}
                    </select>
                    <input className="form-input" type="number" inputMode="numeric" min="1" value={line.quantity} onChange={(event) => updateItemDraft(index, { quantity: event.target.value })} />
                    <span>{formatCurrencyIDR(product ? Number(product.unitPrice || 0) * quantity : 0)}</span>
                    <button className="btn btn-secondary btn-sm" type="button" onClick={() => removeItemDraft(index)} disabled={itemDrafts.length <= 1}>Hapus</button>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="business-detail-lines">
              {(item.items || []).map((line) => (
                <div key={line.id || line.productId || line.productName}>
                  <span>{line.quantity}x {line.productName}</span>
                  <strong>{formatCurrencyIDR(line.lineTotal || 0)}</strong>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="business-detail-section">
          <h4>Cara terima & alamat</h4>
          <div className="business-form-grid order-form">
            <label className="business-field">
              <span>Cara pelanggan menerima pesanan</span>
              <select className="form-select" value={draft.fulfillmentType} onChange={(event) => setDraft((current) => ({ ...current, fulfillmentType: event.target.value }))}>
                {drawerFulfillmentOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </label>
            <label className="business-field">
              <span>Nama penerima</span>
              <input className="form-input" value={draft.recipientName} onChange={(event) => setDraft((current) => ({ ...current, recipientName: event.target.value }))} />
            </label>
            <label className="business-field">
              <span>No. penerima</span>
              <input className="form-input" inputMode="tel" value={draft.recipientPhone} onChange={(event) => setDraft((current) => ({ ...current, recipientPhone: event.target.value }))} />
            </label>
            <label className="business-field business-form-wide">
              <span>Alamat</span>
              <input className="form-input" required={fulfillmentNeedsAddress(draft.fulfillmentType)} value={draft.addressLine} onChange={(event) => setDraft((current) => ({ ...current, addressLine: event.target.value }))} />
            </label>
            <label className="business-field">
              <span>Area/kota</span>
              <input className="form-input" value={draft.addressArea} onChange={(event) => setDraft((current) => ({ ...current, addressArea: event.target.value }))} />
            </label>
            <label className="business-field">
              <span>Catatan alamat</span>
              <input className="form-input" value={draft.addressNotes} onChange={(event) => setDraft((current) => ({ ...current, addressNotes: event.target.value }))} />
            </label>
          </div>
        </div>

        <div className="business-detail-section">
          <h4>Catatan</h4>
          <textarea className="form-input" rows={3} value={draft.notes} onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))} />
        </div>

        <div className="business-detail-actions">
          <button className="btn btn-primary" type="button" onClick={saveDraft} disabled={busyKey === `commerce-order-${item.id}-update`}>Simpan Detail</button>
        </div>
      </aside>
    </div>
  );
}

export function BookingView({
  bookingServices = [],
  bookingAppointments = [],
  bookingServiceForm,
  setBookingServiceForm,
  bookingAppointmentForm,
  setBookingAppointmentForm,
  onSubmitBookingService,
  onEditBookingService,
  onResetBookingService,
  onSubmitBookingAppointment,
  onUpdateBookingAppointmentStatus,
  canManageServices = false,
  busyKey = "",
  toolState,
}) {
  const activeServices = bookingServices.filter((item) => item.status === "active");
  const scheduledCount = bookingAppointments.filter((item) => ["draft", "scheduled"].includes(item.status)).length;
  const confirmedCount = bookingAppointments.filter((item) => item.status === "confirmed").length;
  const completedCount = bookingAppointments.filter((item) => item.status === "completed").length;
  const [selectedAppointment, setSelectedAppointment] = useState(null);
  const [isServiceFormOpen, setServiceFormOpen] = useState(false);
  const selectedAppointmentLive = useMemo(() => (
    selectedAppointment ? bookingAppointments.find((item) => item.id === selectedAppointment.id) || selectedAppointment : null
  ), [bookingAppointments, selectedAppointment]);

  useEffect(() => {
    if (bookingServiceForm?.id) setServiceFormOpen(true);
  }, [bookingServiceForm?.id]);

  function openNewServiceForm() {
    onResetBookingService?.();
    setServiceFormOpen(true);
  }

  function openEditServiceForm(item) {
    onEditBookingService?.(item);
    setServiceFormOpen(true);
  }

  function closeServiceForm() {
    setServiceFormOpen(false);
    onResetBookingService?.();
  }

  async function submitServiceForm(event) {
    const result = await onSubmitBookingService?.(event);
    if (result !== false) setServiceFormOpen(false);
  }

  return (
    <div className="business-workspace-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Booking</div>
          <div className="page-desc">Layanan, jam tersedia, dan jadwal booking bisa disesuaikan per bisnis.</div>
        </div>
        {canManageServices ? (
          <div className="page-actions">
            <button className={`btn ${isServiceFormOpen ? "btn-secondary" : "btn-primary"} btn-sm`} type="button" onClick={isServiceFormOpen ? closeServiceForm : openNewServiceForm}>
              {isServiceFormOpen ? Icons.close : Icons.plus} {isServiceFormOpen ? "Tutup Form" : "Layanan Baru"}
            </button>
          </div>
        ) : null}
      </div>

      <ToolAIBanner toolState={toolState} surface="booking" />

      <div className="business-plugin-summary" aria-label="Ringkasan booking">
        <div>
          <strong>{activeServices.length}</strong>
          <span>Layanan aktif</span>
        </div>
        <div>
          <strong>{scheduledCount}</strong>
          <span>Perlu konfirmasi</span>
        </div>
        <div>
          <strong>{confirmedCount}</strong>
          <span>Terkonfirmasi</span>
        </div>
        <div>
          <strong>{completedCount}</strong>
          <span>Selesai</span>
        </div>
      </div>

      <section className="business-split-grid booking-workspace-grid">
        <div className="card business-section-card">
          <div className="card-header">
            <div>
              <div className="card-title">Layanan Booking</div>
              <div className="card-subtitle">Atur layanan, durasi, buffer, harga, dan jam tersedia.</div>
            </div>
          </div>
          {!canManageServices ? (
            <div className="business-muted-panel">Role ini bisa memakai jadwal booking, tapi perubahan layanan dilakukan oleh admin.</div>
          ) : null}
          <div className="business-service-list">
            {bookingServices.map((item) => (
              <article className="business-service-item" key={item.id}>
                <div>
                  <strong>{item.name}</strong>
                  <span>{item.durationMinutes} menit + {item.bufferMinutes || 0} menit jeda - {formatCurrencyIDR(item.price || 0)}</span>
                  <small>{availabilitySummary(item.availability)}</small>
                </div>
                <div className="business-order-actions">
                  <span className={`badge ${item.status === "active" ? "green" : "gray"}`}>{recordStatusLabel(item.status)}</span>
                  {canManageServices ? <button className="btn btn-secondary btn-sm" type="button" onClick={() => openEditServiceForm(item)}>Edit</button> : null}
                </div>
              </article>
            ))}
            {!bookingServices.length ? <div className="business-muted-panel">Belum ada layanan booking.</div> : null}
          </div>
        </div>

        <div className="card business-section-card">
          <div className="card-header">
            <div>
              <div className="card-title">Jadwal Booking</div>
              <div className="card-subtitle">Buat jadwal pelanggan dari layanan aktif.</div>
            </div>
          </div>
          {activeServices.length ? (
            <form className="business-form-grid booking-appointment-form" onSubmit={onSubmitBookingAppointment}>
              <label className="business-field">
                <span>Layanan</span>
                <select className="form-select" required value={bookingAppointmentForm.serviceId} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, serviceId: event.target.value })}>
                  <option value="">Pilih layanan</option>
                  {activeServices.map((item) => (
                    <option key={item.id} value={item.id}>{item.name}</option>
                  ))}
                </select>
              </label>
              <label className="business-field">
                <span>Nama customer</span>
                <input className="form-input" value={bookingAppointmentForm.customerName} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, customerName: event.target.value })} placeholder="Nama customer" />
              </label>
              <label className="business-field">
                <span>No. WhatsApp</span>
                <input className="form-input" inputMode="tel" value={bookingAppointmentForm.customerPhone} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, customerPhone: event.target.value })} placeholder="628..." />
              </label>
              <label className="business-field">
                <span>Jadwal mulai</span>
                <input className="form-input" type="datetime-local" required value={bookingAppointmentForm.scheduledStart} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, scheduledStart: event.target.value })} />
              </label>
              <label className="business-field">
                <span>Lokasi</span>
                <select className="form-select" value={bookingAppointmentForm.locationType} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, locationType: event.target.value })}>
                  <option value="business_location">Lokasi bisnis</option>
                  <option value="online">Online</option>
                  <option value="customer_address">Alamat customer</option>
                </select>
              </label>
              {bookingAppointmentForm.locationType === "customer_address" ? (
                <label className="business-field business-form-wide">
                  <span>Alamat customer</span>
                  <input className="form-input" required value={bookingAppointmentForm.locationAddress} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, locationAddress: event.target.value })} placeholder="Alamat lengkap customer" />
                </label>
              ) : null}
              <label className="business-field business-form-wide">
                <span>Catatan lokasi</span>
                <input className="form-input" value={bookingAppointmentForm.locationNotes} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, locationNotes: event.target.value })} placeholder="Link rapat/patokan/ruangan" />
              </label>
              <label className="business-field business-form-wide">
                <span>Catatan</span>
                <input className="form-input" value={bookingAppointmentForm.notes} onChange={(event) => setBookingAppointmentForm?.({ ...bookingAppointmentForm, notes: event.target.value })} placeholder="Catatan jadwal" />
              </label>
              <div className="business-form-actions">
                <button className="btn btn-primary btn-sm" type="submit" disabled={busyKey === "booking-appointment"}>
                  Buat Booking
                </button>
              </div>
            </form>
          ) : (
            <div className="business-muted-panel business-empty-action">
              <span>Belum ada layanan aktif. Buat atau aktifkan layanan dulu sebelum menjadwalkan customer.</span>
              {canManageServices ? <button className="btn btn-secondary btn-sm" type="button" onClick={openNewServiceForm}>Buat layanan</button> : null}
            </div>
          )}
          <div className="business-order-list">
            {bookingAppointments.map((item) => (
              <article className="business-order-item booking-appointment-item" key={item.id}>
                <div>
                  <strong>{item.customerName || item.customerPhone || "Pelanggan"}</strong>
                  <span>{item.serviceName} - {bookingStatusLabel(item.status)}</span>
                  <small>{orderSourceLabel(item)}</small>
                  <small>{item.customerPhone || "No. WA belum diisi"}</small>
                  <small>{bookingLocationLabel(item.locationType)}{item.locationAddress ? ` - ${item.locationAddress}` : ""}</small>
                  <small>{formatDateTime(item.scheduledStart)}</small>
                </div>
                <div className="business-order-actions">
                  <button className="btn btn-secondary btn-sm" type="button" onClick={() => setSelectedAppointment(item)}>Detail</button>
                  {["draft", "scheduled"].includes(item.status) ? (
                    <button className="btn btn-primary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "confirmed")}>Konfirmasi</button>
                  ) : null}
                  {item.status !== "completed" && item.status !== "cancelled" ? (
                    <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "completed")}>Selesai</button>
                  ) : null}
                  {item.status !== "cancelled" ? (
                    <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "cancelled")}>Batal</button>
                  ) : null}
                </div>
              </article>
            ))}
            {!bookingAppointments.length ? <div className="business-muted-panel">Belum ada jadwal booking.</div> : null}
          </div>
        </div>
      </section>
      {isServiceFormOpen && canManageServices ? (
        <div className="business-detail-backdrop" role="presentation" onMouseDown={closeServiceForm}>
          <aside className="business-detail-drawer business-service-form-drawer" role="dialog" aria-modal="true" aria-label={bookingServiceForm.id ? "Edit layanan booking" : "Layanan booking baru"} onMouseDown={(event) => event.stopPropagation()}>
            <div className="business-detail-head">
              <div>
                <span>Layanan Booking</span>
                <h3>{bookingServiceForm.id ? "Edit Layanan" : "Layanan Baru"}</h3>
                <p>Atur durasi, harga, jam tersedia, dan status layanan yang dipakai customer.</p>
              </div>
              <button className="btn btn-secondary btn-sm" type="button" onClick={closeServiceForm}>{Icons.close} Tutup</button>
            </div>
            <form className="business-form-grid booking-service-form" onSubmit={submitServiceForm}>
              <label className="business-field">
                <span>Nama layanan</span>
                <input className="form-input" required value={bookingServiceForm.name} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, name: event.target.value })} placeholder="Nama layanan" />
              </label>
              <label className="business-field">
                <span>Durasi menit</span>
                <input className="form-input" type="number" min="1" value={bookingServiceForm.durationMinutes} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, durationMinutes: event.target.value })} placeholder="Contoh: 60" />
              </label>
              <label className="business-field">
                <span>Jeda antar jadwal</span>
                <input className="form-input" type="number" min="0" value={bookingServiceForm.bufferMinutes} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, bufferMinutes: event.target.value })} placeholder="Contoh: 10" />
              </label>
              <label className="business-field">
                <span>Harga (Rp)</span>
                <input className="form-input" type="number" min="0" value={bookingServiceForm.price} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, price: event.target.value })} placeholder="Contoh: 50000" />
              </label>
              <label className="business-field">
                <span>Timezone</span>
                <input className="form-input" value={bookingServiceForm.timezone} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, timezone: event.target.value })} placeholder="Asia/Jakarta" />
              </label>
              <label className="business-field">
                <span>Hari tersedia</span>
                <input className="form-input" value={bookingServiceForm.availabilityDays} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, availabilityDays: event.target.value })} placeholder="1,2,3,4,5" />
              </label>
              <label className="business-field">
                <span>Jam mulai</span>
                <input className="form-input" type="time" value={bookingServiceForm.availabilityStart} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, availabilityStart: event.target.value })} />
              </label>
              <label className="business-field">
                <span>Jam selesai</span>
                <input className="form-input" type="time" value={bookingServiceForm.availabilityEnd} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, availabilityEnd: event.target.value })} />
              </label>
              <label className="business-field">
                <span>Status</span>
                <select className="form-select" value={bookingServiceForm.status} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, status: event.target.value })}>
                  <option value="active">Aktif</option>
                  <option value="inactive">Nonaktif</option>
                  <option value="archived">Arsip</option>
                </select>
              </label>
              <label className="business-field business-form-wide">
                <span>Deskripsi</span>
                <input className="form-input" value={bookingServiceForm.description} onChange={(event) => setBookingServiceForm?.({ ...bookingServiceForm, description: event.target.value })} placeholder="Deskripsi layanan" />
              </label>
              <div className="business-form-actions">
                <button className="btn btn-primary btn-sm" type="submit" disabled={busyKey === "booking-service"}>
                  {bookingServiceForm.id ? "Simpan layanan" : "Tambah layanan"}
                </button>
                <button className="btn btn-secondary btn-sm" type="button" onClick={closeServiceForm}>Batal</button>
              </div>
            </form>
          </aside>
        </div>
      ) : null}
      {selectedAppointmentLive ? (
        <BookingDetailDrawer
          item={selectedAppointmentLive}
          onClose={() => setSelectedAppointment(null)}
          onUpdateBookingAppointmentStatus={onUpdateBookingAppointmentStatus}
        />
      ) : null}
    </div>
  );
}

function BookingDetailDrawer({ item, onClose, onUpdateBookingAppointmentStatus }) {
  return (
    <div className="business-detail-backdrop" role="presentation" onMouseDown={onClose}>
      <aside className="business-detail-drawer" role="dialog" aria-modal="true" aria-label="Detail booking" onMouseDown={(event) => event.stopPropagation()}>
        <div className="business-detail-head">
          <div>
            <span>Detail Booking</span>
            <h3>{item.customerName || item.customerPhone || "Pelanggan"}</h3>
            <p>{item.serviceName} - {bookingStatusLabel(item.status)}</p>
          </div>
          <button className="btn btn-secondary btn-sm" type="button" onClick={onClose}>Tutup</button>
        </div>
        <div className="business-detail-section">
          <h4>Identitas pemesan</h4>
          <div className="business-detail-grid">
            <div><span>Nama</span><strong>{item.customerName || "-"}</strong></div>
            <div><span>No. WhatsApp</span><strong>{item.customerPhone || "-"}</strong></div>
            <div><span>Sumber</span><strong>{orderSourceLabel(item)}</strong></div>
            <div><span>Dibuat</span><strong>{formatDateTime(item.createdAt)}</strong></div>
          </div>
        </div>
        <div className="business-detail-section">
          <h4>Jadwal & layanan</h4>
          <div className="business-detail-grid">
            <div><span>Layanan</span><strong>{item.serviceName || "-"}</strong></div>
            <div><span>Mulai</span><strong>{formatDateTime(item.scheduledStart)}</strong></div>
            <div><span>Selesai</span><strong>{formatDateTime(item.scheduledEnd)}</strong></div>
            <div><span>Status</span><strong>{bookingStatusLabel(item.status)}</strong></div>
          </div>
          <div className="business-order-actions">
            {["draft", "scheduled"].includes(item.status) ? <button className="btn btn-primary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "confirmed")}>Konfirmasi</button> : null}
            {item.status !== "completed" && item.status !== "cancelled" ? <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "completed")}>Selesai</button> : null}
            {item.status !== "cancelled" ? <button className="btn btn-secondary btn-sm" type="button" onClick={() => onUpdateBookingAppointmentStatus?.(item.id, "cancelled")}>Batal</button> : null}
          </div>
        </div>
        <div className="business-detail-section">
          <h4>Lokasi</h4>
          <div className="business-detail-grid">
            <div><span>Tipe</span><strong>{bookingLocationLabel(item.locationType)}</strong></div>
            <div><span>Alamat/link</span><strong>{item.locationAddress || "-"}</strong></div>
          </div>
          <p className="business-detail-note">{item.locationNotes || "Tidak ada catatan lokasi."}</p>
        </div>
        <div className="business-detail-section">
          <h4>Catatan</h4>
          <p className="business-detail-note">{item.notes || "Tidak ada catatan."}</p>
        </div>
      </aside>
    </div>
  );
}
