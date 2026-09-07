import { useMemo, useState } from "react";
import { defaultAvailableModelIds, defaultModelAliases, normalizeModelOptions, formatNumber } from "../../../lib/dashboard-core";

const defaultPricingForm = {
  creditUnitIdr: "500",
  usdToIdrRate: "16000",
  chatModelName: "openai/gpt-4o-mini",
  chatInputPricePer1m: "0.15",
  chatOutputPricePer1m: "0.60",
  embeddingModelName: "text-embedding-3-small",
  embeddingPricePer1m: "0.02",
  annualDiscountPercent: "10",
  monthlyCreditLimit: "100000",
  modelAliases: defaultModelAliases,
  availableModelIds: defaultAvailableModelIds,
};

const estimateRows = [
  ["averageAnswer", "Rata-rata jawaban"],
  ["complexAnswer", "Jawaban kompleks"],
  ["imageAnswer", "Gambar/media"],
  ["escalation", "Eskalasi ke agent"],
];

export function PricingSettingsView({ role, pricing, aiModels = [], pricingForm, setPricingForm, savePricing, busyKey }) {
  const canManageFullPricing = role === "owner";
  const [modelSearch, setModelSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");
  const [isAddingModel, setIsAddingModel] = useState(false);
  const [addModelId, setAddModelId] = useState("");
  const [addModelAlias, setAddModelAlias] = useState("");
  const catalogModels = aiModels;

  const modelOptions = useMemo(
    () => normalizeModelOptions(catalogModels, pricingForm.chatModelName),
    [catalogModels, pricingForm.chatModelName]
  );
  const userVisibleModelOptions = useMemo(
    () => modelOptions.filter((model) => model.available !== false),
    [modelOptions]
  );
  const catalogModelOptions = canManageFullPricing ? modelOptions : userVisibleModelOptions;
  const selectedModel = catalogModelOptions.find((model) => model.id === pricingForm.chatModelName) || modelOptions.find((model) => model.id === pricingForm.chatModelName) || catalogModelOptions[0] || modelOptions[0];
  const availableModelIds = useMemo(() => {
    const ids = pricingForm.availableModelIds?.length
      ? pricingForm.availableModelIds
      : modelOptions.filter((model) => model.available).map((model) => model.id);
    const normalizedIds = ids.length ? ids : defaultAvailableModelIds;
    const nextIds = new Set(normalizedIds.map((modelId) => String(modelId).trim().toLowerCase()).filter(Boolean));
    const activeModelId = String(pricingForm.chatModelName || "").trim().toLowerCase();
    if (activeModelId) nextIds.add(activeModelId);
    return nextIds;
  }, [modelOptions, pricingForm.availableModelIds, pricingForm.chatModelName]);
  const availableModelRows = useMemo(() => {
    const rows = modelOptions.filter((model) => availableModelIds.has(String(model.id).toLowerCase()));
    if (selectedModel && !rows.some((model) => model.id === selectedModel.id)) rows.unshift(selectedModel);
    return rows;
  }, [availableModelIds, modelOptions, selectedModel]);
  const availableCount = availableModelRows.length;
  const activeProviderCounts = useMemo(() => availableModelRows.reduce((counts, model) => {
    const provider = model.provider || "OpenRouter";
    counts[provider] = (counts[provider] || 0) + 1;
    return counts;
  }, {}), [availableModelRows]);
  const providerCounts = useMemo(() => modelOptions.reduce((counts, model) => {
    const provider = model.provider || "OpenRouter";
    counts[provider] = (counts[provider] || 0) + 1;
    return counts;
  }, {}), [modelOptions]);
  const providerOptions = useMemo(
    () => Object.keys(providerCounts).sort((left, right) => {
      const priority = { GPT: 0, DeepSeek: 1, Claude: 2, OpenRouter: 9 };
      return (priority[left] ?? 5) - (priority[right] ?? 5) || left.localeCompare(right);
    }),
    [providerCounts]
  );
  const addableModelOptions = useMemo(() => {
    const query = modelSearch.trim().toLowerCase();
    return modelOptions.filter((model) => {
      if (availableModelIds.has(String(model.id).toLowerCase())) return false;
      const matchesProvider = providerFilter === "all" || model.provider === providerFilter;
      if (!matchesProvider) return false;
      if (!query) return true;
      return [model.name, model.alias, model.upstreamName, model.id, model.provider]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query));
    });
  }, [availableModelIds, modelOptions, modelSearch, providerFilter]);
  const selectedAddModel = addableModelOptions.find((model) => model.id === addModelId) || addableModelOptions[0] || null;
  const defaultModelOptions = useMemo(() => {
    const options = canManageFullPricing ? modelOptions : userVisibleModelOptions;
    if (!selectedModel) return options;
    if (options.some((model) => model.id === selectedModel.id)) return options;
    return [selectedModel, ...options];
  }, [canManageFullPricing, modelOptions, selectedModel, userVisibleModelOptions]);
  const aliasForModel = (model) => {
    if (!model) return "";
    const aliases = pricingForm.modelAliases || defaultModelAliases;
    return aliases[model.id] ?? aliases[String(model.id).toLowerCase()] ?? model.alias ?? model.name ?? model.upstreamName ?? model.id;
  };
  const updateField = (key, value) => setPricingForm({ ...pricingForm, [key]: value });
  const updateAlias = (modelId, value) => {
    const normalizedModelId = String(modelId || "").trim().toLowerCase();
    if (!normalizedModelId) return;
    setPricingForm({
      ...pricingForm,
      modelAliases: {
        ...(pricingForm.modelAliases || defaultModelAliases),
        [normalizedModelId]: value,
      },
    });
  };
  const updateChatModel = (modelId) => {
    const selected = modelOptions.find((model) => model.id === modelId);
    const nextAvailableModelIds = new Set([...(pricingForm.availableModelIds || defaultAvailableModelIds)].map((id) => String(id).trim().toLowerCase()).filter(Boolean));
    nextAvailableModelIds.add(String(modelId).trim().toLowerCase());
    setPricingForm({
      ...pricingForm,
      chatModelName: modelId,
      chatInputPricePer1m: selected ? String(selected.inputPricePer1m) : pricingForm.chatInputPricePer1m,
      chatOutputPricePer1m: selected ? String(selected.outputPricePer1m) : pricingForm.chatOutputPricePer1m,
      availableModelIds: Array.from(nextAvailableModelIds),
    });
  };

  const addAvailableModel = () => {
    const model = selectedAddModel;
    if (!model) return;
    const normalizedModelId = String(model.id || "").trim().toLowerCase();
    if (!normalizedModelId) return;
    const nextAvailableModelIds = new Set([...(pricingForm.availableModelIds || defaultAvailableModelIds)].map((id) => String(id).trim().toLowerCase()).filter(Boolean));
    nextAvailableModelIds.add(normalizedModelId);
    setPricingForm({
      ...pricingForm,
      availableModelIds: Array.from(nextAvailableModelIds),
      modelAliases: {
        ...(pricingForm.modelAliases || defaultModelAliases),
        [normalizedModelId]: String(addModelAlias || aliasForModel(model)).trim() || model.upstreamName || model.id,
      },
    });
    setAddModelId("");
    setAddModelAlias("");
    setIsAddingModel(false);
  };

  const removeAvailableModel = (modelId) => {
    const normalizedModelId = String(modelId || "").trim().toLowerCase();
    if (!normalizedModelId) return;
    const nextAvailableModelIds = new Set([...(pricingForm.availableModelIds || defaultAvailableModelIds)].map((id) => String(id).trim().toLowerCase()).filter(Boolean));
    if (nextAvailableModelIds.size <= 1) return;
    nextAvailableModelIds.delete(normalizedModelId);
    const remainingIds = Array.from(nextAvailableModelIds);
    let nextChatModelName = pricingForm.chatModelName;
    let nextInputPrice = pricingForm.chatInputPricePer1m;
    let nextOutputPrice = pricingForm.chatOutputPricePer1m;
    if (normalizedModelId === String(pricingForm.chatModelName || "").toLowerCase()) {
      nextChatModelName = remainingIds[0] || "openai/gpt-4o-mini";
      const nextDefaultModel = modelOptions.find((model) => String(model.id).toLowerCase() === nextChatModelName);
      if (nextDefaultModel) {
        nextInputPrice = String(nextDefaultModel.inputPricePer1m);
        nextOutputPrice = String(nextDefaultModel.outputPricePer1m);
      }
    }
    setPricingForm({
      ...pricingForm,
      chatModelName: nextChatModelName,
      chatInputPricePer1m: nextInputPrice,
      chatOutputPricePer1m: nextOutputPrice,
      availableModelIds: remainingIds,
    });
  };

  const modelOptionLabel = (model) => {
    const alias = aliasForModel(model);
    const upstream = model.upstreamName && model.upstreamName !== alias ? ` (${model.upstreamName})` : "";
    if (!canManageFullPricing) return `${alias}${upstream}`;
    return `${alias}${upstream} - ${model.id}`;
  };
  const openRouterDropdownLabel = (model) => {
    const provider = model.provider || "OpenRouter";
    const modelName = model.upstreamName || model.name || model.id;
    return `${modelName} - ${provider} - ${model.id}`;
  };
  const activeModelId = String(pricingForm.chatModelName || "").trim().toLowerCase();
  const formatUSD = (value) => {
    const amount = Number(value || 0);
    if (!Number.isFinite(amount) || amount <= 0) return "$0";
    return `$${amount.toLocaleString("en-US", { maximumFractionDigits: 4 })}`;
  };
  const modelTierLabel = (model) => {
    const id = String(model?.id || "").toLowerCase();
    if (id === "openai/gpt-4.1-mini" || id.includes("gpt-4.1")) return "Advance";
    if (id.includes("deepseek") && (id.includes("pro") || id.includes("r1"))) return "Advance";
    if (id.includes("deepseek")) return "Hemat";
    if (id === "openai/gpt-4o-mini" || id.includes("mini") || id.includes("haiku")) return "Hemat";
    if (id.includes("opus") || id.includes("pro")) return "Premium";
    if (id.includes("sonnet") || id.includes("gpt-5") || id.includes("gpt-4.1")) return "Advance";
    return "Hemat";
  };

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">{canManageFullPricing ? "Model & Harga" : "Model AI"}</div>
          <div className="page-desc">
            {canManageFullPricing
              ? "Pilih default model, tentukan model yang tampil ke user, lalu atur harga kredit."
              : "Pilih model AI aktif dan lihat estimasi pemakaian kredit per jenis jawaban."}
          </div>
        </div>
        <div className="page-actions">
          {canManageFullPricing ? (
            <button className="btn btn-secondary" type="button" onClick={() => setPricingForm(defaultPricingForm)}>Reset Default</button>
          ) : null}
          <button className="btn btn-primary" onClick={savePricing} disabled={busyKey === "pricing"}>Simpan Perubahan</button>
        </div>
      </div>

      {canManageFullPricing ? (
        <div className="pricing-owner-flow">
          <section className="card pricing-model-manager-card">
            <div className="pricing-manager-head">
              <div>
                <span className="pricing-section-kicker">Model user</span>
                <div className="card-title">Daftar model aktif</div>
                <p>Yang aktif di sini akan muncul di dashboard user. Alias adalah nama yang mereka lihat.</p>
              </div>
              <div className="pricing-manager-stats" aria-label="Ringkasan model aktif">
                <span><strong>{availableCount}</strong> aktif</span>
                <span><strong>{activeProviderCounts.GPT || 0}</strong> GPT</span>
                <span><strong>{activeProviderCounts.DeepSeek || 0}</strong> DeepSeek</span>
                <span><strong>{activeProviderCounts.Claude || 0}</strong> Claude</span>
              </div>
            </div>

            <div className="pricing-default-strip">
              <div>
                <span>Default chat saat ini</span>
                <strong>{aliasForModel(selectedModel)}</strong>
                <code>{selectedModel?.id || pricingForm.chatModelName}</code>
              </div>
              <div className="pricing-default-price">
                <span>Input {formatUSD(pricingForm.chatInputPricePer1m)}/1M</span>
                <span>Output {formatUSD(pricingForm.chatOutputPricePer1m)}/1M</span>
              </div>
            </div>

            <div className="pricing-variant-manager">
              <div className="pricing-variant-list-head">
                <div>
                  <span className="pricing-section-kicker">Variant aktif</span>
                  <strong>Model yang sudah tersedia untuk user</strong>
                </div>
                <button
                  className="btn btn-primary"
                  type="button"
                  onClick={() => {
                    setIsAddingModel((current) => !current);
                    if (!isAddingModel && selectedAddModel) setAddModelAlias(aliasForModel(selectedAddModel));
                  }}
                >
                  {isAddingModel ? "Tutup form" : "Tambah variant model"}
                </button>
              </div>

              <div className="pricing-active-table">
                <div className="pricing-active-table-head">
                  <span>Alias user</span>
                  <span>Model OpenRouter</span>
                  <span>Harga dasar</span>
                  <span>Aksi</span>
                </div>
                <div className="pricing-active-list">
                  {availableModelRows.map((model) => {
                    const modelId = String(model.id).toLowerCase();
                    const isActive = modelId === activeModelId;
                    return (
                      <div className={`pricing-active-row ${isActive ? "default" : ""}`} key={model.id}>
                        <label className="pricing-alias-cell">
                          <span>Alias user</span>
                          <input
                            className="form-input"
                            value={aliasForModel(model)}
                            onChange={(e) => updateAlias(model.id, e.target.value)}
                            maxLength={80}
                            placeholder={model.upstreamName || model.id}
                          />
                          {isActive ? <em>Default chat</em> : null}
                        </label>

                        <div className="pricing-model-cell">
                          <strong>{model.upstreamName || model.name || model.id}</strong>
                          <code>{model.id}</code>
                          <div>
                            <span>{model.provider || "OpenRouter"}</span>
                            <span>{modelTierLabel(model)}</span>
                          </div>
                        </div>

                        <div className="pricing-model-price">
                          <span>Input {formatUSD(model.inputPricePer1m)}/1M</span>
                          <span>Output {formatUSD(model.outputPricePer1m)}/1M</span>
                        </div>

                        <div className="pricing-active-row-actions">
                          <button className="btn btn-secondary" type="button" onClick={() => updateChatModel(model.id)} disabled={isActive}>
                            {isActive ? "Default" : "Set default"}
                          </button>
                          <button className="btn btn-secondary pricing-danger-action" type="button" onClick={() => removeAvailableModel(model.id)} disabled={availableModelRows.length <= 1}>
                            Hapus
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {isAddingModel ? (
                <div className="pricing-model-dropdown-panel">
                  <div className="pricing-add-sidebar-head">
                    <div>
                      <span className="pricing-section-kicker">Tambah variant baru</span>
                      <strong>Pilih model OpenRouter lalu kasih alias</strong>
                    </div>
                    <span>{addableModelOptions.length} tersedia</span>
                  </div>

                  <div className="pricing-catalog-toolbar">
                    <div className="form-group">
                      <label className="form-label">Cari model</label>
                      <input
                        className="form-input"
                        value={modelSearch}
                        onChange={(event) => {
                          setModelSearch(event.target.value);
                          setAddModelId("");
                          setAddModelAlias("");
                        }}
                        placeholder="GPT 4.1, DeepSeek, Claude Sonnet, model ID"
                      />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Provider</label>
                      <select
                        className="form-select"
                        value={providerFilter}
                        onChange={(event) => {
                          setProviderFilter(event.target.value);
                          setAddModelId("");
                          setAddModelAlias("");
                        }}
                      >
                        <option value="all">Semua ({modelOptions.length})</option>
                        {providerOptions.map((provider) => (
                          <option key={provider} value={provider}>{provider} ({providerCounts[provider] || 0})</option>
                        ))}
                      </select>
                    </div>
                  </div>

                  <label className="form-group pricing-openrouter-select">
                    <span className="form-label">Model OpenRouter</span>
                    <select
                      className="form-select"
                      value={selectedAddModel?.id || ""}
                      onChange={(event) => {
                        const nextModel = addableModelOptions.find((model) => model.id === event.target.value);
                        setAddModelId(event.target.value);
                        setAddModelAlias(nextModel ? aliasForModel(nextModel) : "");
                      }}
                      disabled={!addableModelOptions.length}
                    >
                      {addableModelOptions.map((model) => (
                        <option key={model.id} value={model.id}>{openRouterDropdownLabel(model)}</option>
                      ))}
                    </select>
                  </label>

                  <div className="pricing-add-controls">
                    <label className="form-group">
                      <span className="form-label">Alias yang tampil ke user</span>
                      <input
                        className="form-input"
                        value={addModelAlias}
                        onChange={(event) => setAddModelAlias(event.target.value)}
                        placeholder={selectedAddModel?.upstreamName || "Contoh: DeepSeek V4 Flash"}
                        disabled={!selectedAddModel}
                        maxLength={80}
                      />
                    </label>

                    <button className="btn btn-primary pricing-add-submit" type="button" onClick={addAvailableModel} disabled={!selectedAddModel}>
                      Tambah variant
                    </button>
                  </div>

                  {selectedAddModel ? (
                    <div className="pricing-add-model-preview">
                      <strong>{selectedAddModel.id}</strong>
                      <span>{selectedAddModel.provider || "OpenRouter"}</span>
                      <span>{selectedAddModel.contextLength ? `${formatNumber(selectedAddModel.contextLength)} context` : "Context OpenRouter"}</span>
                      <span>Input {formatUSD(selectedAddModel.inputPricePer1m)}/1M</span>
                      <span>Output {formatUSD(selectedAddModel.outputPricePer1m)}/1M</span>
                    </div>
                  ) : (
                    <div className="pricing-model-empty">Semua model yang cocok filter sudah aktif.</div>
                  )}

                  <div className="pricing-version-note">
                    Dropdown ini berisi model OpenRouter GPT, DeepSeek, dan Claude versi eksplisit. Route dinamis seperti latest tidak dimasukkan.
                  </div>
                </div>
              ) : null}
            </div>
          </section>

          <section className="card pricing-controls-card">
            <div className="pricing-section-head">
              <div>
                <span className="pricing-section-kicker">3. Harga & kredit</span>
                <div className="card-title">Formula pemotongan kredit</div>
              </div>
            </div>
            <div className="pricing-control-grid">
              <label className="form-group">
                <span className="form-label">Harga input chat / 1M token (USD)</span>
                <input type="number" step="0.0001" className="form-input" value={pricingForm.chatInputPricePer1m} onChange={(e) => updateField("chatInputPricePer1m", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Harga output chat / 1M token (USD)</span>
                <input type="number" step="0.0001" className="form-input" value={pricingForm.chatOutputPricePer1m} onChange={(e) => updateField("chatOutputPricePer1m", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Nilai 1 kredit (IDR)</span>
                <input type="number" step="1" className="form-input" value={pricingForm.creditUnitIdr} onChange={(e) => updateField("creditUnitIdr", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Kurs USD ke IDR</span>
                <input type="number" step="1" className="form-input" value={pricingForm.usdToIdrRate} onChange={(e) => updateField("usdToIdrRate", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Limit kredit bulanan</span>
                <input type="number" className="form-input" value={pricingForm.monthlyCreditLimit} onChange={(e) => updateField("monthlyCreditLimit", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Diskon paket tahunan (%)</span>
                <input type="number" min="0" max="99.99" step="0.01" className="form-input" value={pricingForm.annualDiscountPercent} onChange={(e) => updateField("annualDiscountPercent", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Model embedding</span>
                <input className="form-input" value={pricingForm.embeddingModelName} onChange={(e) => updateField("embeddingModelName", e.target.value)} />
              </label>
              <label className="form-group">
                <span className="form-label">Harga embedding / 1M token (USD)</span>
                <input type="number" step="0.0001" className="form-input" value={pricingForm.embeddingPricePer1m} onChange={(e) => updateField("embeddingPricePer1m", e.target.value)} />
              </label>
            </div>
          </section>

          <section className="card pricing-credit-card">
            <div className="card-title mb-16">Status kredit aktif</div>
            <div className="pricing-credit-grid">
              <div>
                <span>Sisa kredit bulanan</span>
                <strong>{formatNumber(pricing?.monthlyCreditsRemaining ?? 0)}</strong>
              </div>
              <div>
                <span>Kredit tambahan</span>
                <strong>{formatNumber(pricing?.additionalCreditsRemaining ?? 0)}</strong>
              </div>
            </div>
          </section>
        </div>
      ) : (
        <div className="pricing-user-grid">
          <section className="card">
            <div className="card-title mb-16">Model AI Chat</div>
            <div className="form-group mb-16">
              <label className="form-label">Model Chat</label>
              <select className="form-select" value={pricingForm.chatModelName} onChange={(e) => updateChatModel(e.target.value)}>
                {defaultModelOptions.map((model) => (
                  <option key={model.id} value={model.id}>{modelOptionLabel(model)}</option>
                ))}
              </select>
            </div>
          </section>
          <section className="card">
            <div className="card-title mb-16">Estimasi pemakaian kredit</div>
            <div className="text-sm text-muted mb-16">{selectedModel?.name || "Model aktif"} memakai estimasi berikut.</div>
            <div className="pricing-estimate-list">
              {estimateRows.map(([key, label]) => (
                <div className="flex-between" key={key}>
                  <span className="text-sm text-muted">{label}</span>
                  <strong>{formatNumber(selectedModel?.creditEstimate?.[key] ?? 0)} kredit</strong>
                </div>
              ))}
            </div>
          </section>
        </div>
      )}
    </>
  );
}
