import { useEffect, useMemo, useRef, useState } from "react";
import { EmptyState } from "../ui";
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
  settingsApiToDraft,
  dashboardBadge,
  activeViewStorageKey,
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

// Messages stored before migration 052 can still carry this inbound sender value.
const legacyInboundSenderType = "candidate";

export function InboxView({
  auth,
  inbox,
  displayInbox,
  inboxTab,
  setInboxTab,
  inboxSearch,
  setInboxSearch,
  waSessions = [],
  aiAgents = [],
  activeWaSessionId = "",
  setActiveWaSessionId,
  activeAIAgentId = "",
  setActiveAIAgentId,
  selectedConversation,
  fetchConversation,
  runConversationAction,
  teamMembers = [],
  saveConversationWorkflow,
  createDealFromConversation,
  openDealFromConversation,
  openContactFromConversation,
  manualMessage,
  setManualMessage,
  manualMedia,
  setManualMedia,
  takeoverNote,
  canOpenChatOps,
  canForceTakeover,
  canUseProspects = true,
  isOwner,
  isOperator,
  navigateToWhatsApp,
  aiTyping,
  busyKey,
  onNotify,
}) {
  const conversation = selectedConversation?.conversation;
  const messages = selectedConversation?.messages ?? [];
  const composer = selectedConversation?.composer ?? { enabled: false, hint: "Composer belum tersedia." };
  const whatsappWindow = composer?.whatsappWindow || conversation?.whatsappWindow || null;
  const openDeals = Array.isArray(conversation?.openDeals) ? conversation.openDeals : [];
  const isHumanMode = conversation?.mode === "human";
  const canSendManualMessage = Boolean(composer.enabled);
  const canSubmitManualMessage = canSendManualMessage && (manualMessage.trim() || manualMedia);
  const aiTypingDetails = aiTyping?.details || {};
  const isAIThinking = Boolean(aiTyping?.active && conversation?.id && aiTypingDetails.conversationId === conversation.id);
  const messageThreadRef = useRef(null);
  const [workflowDraft, setWorkflowDraft] = useState({ status: "open", priority: "normal", assignedToId: "", slaDueAt: "", internalNote: "" });

  function dateTimeLocalValue(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    const shifted = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return shifted.toISOString().slice(0, 16);
  }

  function dateTimeLocalToISO(value) {
    if (!value) return null;
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return null;
    return date.toISOString();
  }

  useEffect(() => {
    if (!conversation?.id) return;
    setWorkflowDraft({
      status: conversation.status || "open",
      priority: conversation.priority || "normal",
      assignedToId: conversation.assignedToId || "",
      slaDueAt: dateTimeLocalValue(conversation.slaDueAt),
      internalNote: conversation.internalNote || "",
    });
  }, [conversation?.id, conversation?.status, conversation?.priority, conversation?.assignedToId, conversation?.slaDueAt, conversation?.internalNote]);
  const previousConversationIdRef = useRef(null);
  const manualTextareaRef = useRef(null);
  const [emojiPickerOpen, setEmojiPickerOpen] = useState(false);
  const [mobilePane, setMobilePane] = useState("list");

  const conversationName = conversation?.contactName || conversation?.phone || "Contact";
  const conversationInitials = avatarInitials(conversationName);
  const conversationAvatarColor = avatarColorClass(conversationName);

  useEffect(() => {
    const thread = messageThreadRef.current;
    if (!thread) return;
    const changedConversation = previousConversationIdRef.current !== conversation?.id;
    previousConversationIdRef.current = conversation?.id || null;
    const distanceFromBottom = thread.scrollHeight - thread.scrollTop - thread.clientHeight;
    if (changedConversation || distanceFromBottom < 220) {
      thread.scrollTop = thread.scrollHeight;
    }
  }, [conversation?.id, messages.length, isAIThinking]);

  useEffect(() => {
    setEmojiPickerOpen(false);
  }, [conversation?.id, canSendManualMessage]);

  useEffect(() => {
    if (!conversation?.id) {
      setMobilePane("list");
      return;
    }
    setMobilePane((current) => (current === "list" ? "chat" : current));
  }, [conversation?.id]);

  const counts = {
    all: inbox.length,
    ai: inbox.filter((item) => item.mode === "ai").length,
    human: inbox.filter((item) => item.mode === "human").length,
    pending: inbox.filter((item) => item.status === "pending_human").length,
    resolved: inbox.filter((item) => item.status === "resolved").length,
  };
  const canReturnToAI = isHumanMode && (!isOperator || conversation?.assignedToId === auth?.user?.id);

  function submitWorkflow() {
    if (!conversation?.id || !saveConversationWorkflow) return;
    saveConversationWorkflow(conversation.id, {
      status: workflowDraft.status,
      priority: workflowDraft.priority,
      assignedToId: workflowDraft.assignedToId,
      slaDueAt: dateTimeLocalToISO(workflowDraft.slaDueAt),
      internalNote: workflowDraft.internalNote,
    });
  }

  const timelineItems = useMemo(() => {
    const rows = messages.slice(-7).map((message) => {
      const sender = message.senderType || "";
      const direction = message.direction || "";
      let label = "Chat updated";
      let tone = "gray";
      if (["customer", legacyInboundSenderType].includes(sender) || direction === "inbound") {
        label = "Pesan masuk dari kontak";
        tone = "gray";
      } else if (sender === "ai") {
        label = "AI menjawab kontak";
        tone = "green";
      } else if (sender === "agent") {
        label = "Pesan dikirim agent";
        tone = "blue";
      } else if (sender === "system") {
        label = "System update";
        tone = "orange";
      }
      return {
        id: message.id || `${sender}-${message.createdAt}`,
        label,
        tone,
        at: message.createdAt || message.sentAt || message.deliveredAt,
      };
    }).filter((item) => item.at);
    if (conversation?.createdAt) {
      rows.unshift({
        id: `conversation-created-${conversation.id}`,
        label: "Percakapan dibuka",
        tone: "gray",
        at: conversation.createdAt,
      });
    }
    return rows.sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime()).slice(0, 8);
  }, [conversation?.createdAt, conversation?.id, messages]);

  async function handleManualMediaChange(event) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (file.size > 8 * 1024 * 1024) {
      onNotify?.({ tone: "warn", title: "File terlalu besar", message: "Ukuran media maksimal 8 MB." });
      return;
    }
    const base64 = await fileToBase64(file);
    setManualMedia({
      base64,
      mimeType: file.type || "application/octet-stream",
      fileName: file.name || "attachment",
      size: file.size,
      kind: mediaKindFromMime(file.type),
    });
  }

  function appendManualEmoji(emoji) {
    if (!canSendManualMessage) return;
    const textarea = manualTextareaRef.current;
    const current = manualMessage || "";
    const start = textarea?.selectionStart ?? current.length;
    const end = textarea?.selectionEnd ?? current.length;
    const next = `${current.slice(0, start)}${emoji}${current.slice(end)}`;
    setManualMessage(next);
    setEmojiPickerOpen(false);
    window.requestAnimationFrame(() => {
      if (!manualTextareaRef.current) return;
      const caret = start + emoji.length;
      manualTextareaRef.current.focus();
      manualTextareaRef.current.setSelectionRange(caret, caret);
    });
  }

  function sendManualMessage() {
    if (!conversation?.id || !canSubmitManualMessage || busyKey === "conversation-manual-message") return;
    runConversationAction(conversation.id, "manual-message", {
      text: manualMessage,
      phone: conversation.phone,
      mediaBase64: manualMedia?.base64,
      mediaMime: manualMedia?.mimeType,
      mediaName: manualMedia?.fileName,
      mediaKind: manualMedia?.kind,
    });
  }

  function handleManualMessageKeyDown(event) {
    if (event.key !== "Enter" || event.shiftKey || event.nativeEvent?.isComposing) return;
    event.preventDefault();
    sendManualMessage();
  }

  function modeBadge(mode) {
    return (
      <span className={`badge ${mode === "human" ? "blue" : "green"} inbox-mini-badge`}>
        {mode === "human" ? "Human" : "AI"}
      </span>
    );
  }

  function statusBadge(status) {
    const normalized = String(status || "open");
    const tone = normalized === "pending_human" ? "orange" : normalized === "resolved" ? "gray" : "green";
    const label = normalized === "pending_human" ? "Pending" : normalized === "resolved" ? "Resolved" : "Active";
    return <span className={`badge ${tone} inbox-mini-badge`}>{label}</span>;
  }

  function avatarInitials(value) {
    const normalized = String(value || "").trim();
    if (!normalized) return "?";
    if (/^\+?\d/.test(normalized)) return "WA";
    return getInitials(normalized);
  }

  function avatarColorClass(value) {
    const colors = ["blue", "green", "orange", "purple", "red"];
    const text = String(value || "");
    let hash = 0;
    for (let index = 0; index < text.length; index += 1) {
      hash = text.charCodeAt(index) + ((hash << 5) - hash);
    }
    return colors[Math.abs(hash) % colors.length];
  }

  function senderClass(message) {
    if (["customer", legacyInboundSenderType].includes(message.senderType) || message.direction === "inbound") return "customer";
    if (message.senderType === "ai") return "ai";
    if (message.senderType === "system") return "system";
    return "agent";
  }

  function senderLabel(type) {
    if (type === "ai") return "AI Assistant";
    if (type === "agent") return conversation?.assignedToName ? `${conversation.assignedToName} (Tim Support)` : "Tim Support";
    return "";
  }

  function openConversation(itemId) {
    fetchConversation(itemId);
    setMobilePane("chat");
  }

  const activeMobilePane = conversation ? mobilePane : "list";

  return (
    <>
    {!canOpenChatOps && (
      <div className="notice warn inbox-connection-warning">
        <div>
          <strong>Belum ada channel yang terkoneksi</strong>
          <p>Inbox tetap bisa dibuka untuk review dan Return to AI. Kirim pesan manual butuh WhatsApp connected.</p>
        </div>
        <button className="btn btn-primary btn-sm" type="button" onClick={navigateToWhatsApp}>
          Buka WhatsApp
        </button>
      </div>
    )}
    <div className={`ops-layout inbox-layout inbox-mobile-${activeMobilePane}`}>
      {/* Column 1: Conversation List */}
      <section className="inbox-list">
        <div className="inbox-list-header">
          <div className="inbox-search">
            <span className="icon-inline">{Icons.search}</span>
            <input value={inboxSearch} onChange={(event) => setInboxSearch(event.target.value)} placeholder="Cari kontak..." />
          </div>
          <div className="inbox-filter-grid">
            <select value={activeWaSessionId} onChange={(event) => setActiveWaSessionId?.(event.target.value)} title="Filter WhatsApp">
              <option value="">Semua WhatsApp</option>
              {waSessions.map((session) => (
                <option key={session.id} value={session.id}>{session.label}</option>
              ))}
            </select>
            <select value={activeAIAgentId} onChange={(event) => setActiveAIAgentId?.(event.target.value)} title="Filter AI Agent">
              <option value="">Semua Agent</option>
              {aiAgents.map((agent) => (
                <option key={agent.id} value={agent.id}>{agent.name}</option>
              ))}
            </select>
          </div>
          <div className="inbox-tabs">
            {[
              { id: "all", label: "All" },
              { id: "ai", label: "AI" },
              { id: "human", label: "Human" },
              { id: "pending", label: "Pending" },
            ].map((tab) => (
              <button
                key={tab.id}
                className={`inbox-tab ${inboxTab === tab.id ? "active" : ""}`}
                onClick={() => setInboxTab(tab.id)}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div className="inbox-items">
          {displayInbox.length ? displayInbox.map((item) => {
            const isActive = conversation?.id === item.id;
            const itemName = item.contactName || item.phone || "Unknown";
            return (
              <button key={item.id} className={`inbox-item ${isActive ? "active" : ""}`} onClick={() => openConversation(item.id)}>
                <div className={`avatar avatar-sm ${avatarColorClass(itemName)}`}>
                  {avatarInitials(itemName)}
                </div>
                <div className="conversation-main">
                  <div className="conversation-topline inbox-item-header">
                    <strong className="inbox-name">{itemName}</strong>
                    <span className="conversation-time inbox-time">{formatTime(item.lastMessageAt)}</span>
                  </div>
                  <div className="conversation-preview inbox-preview">
                    {truncateText(item.lastMessageText || "No messages yet.", 40)}
                  </div>
                  <div className="conversation-tags inbox-meta">
                    {modeBadge(item.mode)}
                    {statusBadge(item.status)}
                    {item.whatsappSession ? <span className="badge gray inbox-mini-badge">{item.whatsappSession}</span> : null}
                    {item.aiAgentName ? <span className="badge blue inbox-mini-badge">{item.aiAgentName}</span> : null}
                  </div>
                </div>
              </button>
            );
          }) : (
            <EmptyState title="Belum ada percakapan" copy="Coba ubah filter atau pencarian." />
          )}
        </div>
      </section>

      {/* Column 2: Chat Thread */}
      <section className="chat-panel">
        {conversation ? (
          <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
            <div className="chat-header">
              <div className="mobile-chat-nav">
                <button className="mobile-pane-btn" type="button" onClick={() => setMobilePane("list")}>
                  Daftar chat
                </button>
                <button className="mobile-pane-btn" type="button" onClick={() => setMobilePane("details")}>
                  Detail
                </button>
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: "12px", minWidth: 0 }}>
                <div className={`avatar avatar-md ${conversationAvatarColor}`} style={{ flexShrink: 0 }}>
                  {conversationInitials}
                </div>
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontSize: "14px", fontWeight: "700", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{conversation.contactName || conversation.phone}</div>
                  <div className="chat-subline">
                    {conversation.phone} {modeBadge(conversation.mode)}
                    {conversation.whatsappSession ? <span className="badge gray inbox-mini-badge">{conversation.whatsappSession}</span> : null}
                    {conversation.aiAgentName ? <span className="badge blue inbox-mini-badge">{conversation.aiAgentName}</span> : null}
                  </div>
                </div>
              </div>
            </div>

            {!isOwner && (
              <div className="chat-actions" aria-label="Aksi percakapan">
                {canReturnToAI && (
                  <button className="btn btn-secondary btn-sm" disabled={busyKey === "conversation-return-to-ai"} onClick={() => runConversationAction(conversation.id, "return-to-ai", {})}>
                    Return to AI
                  </button>
                )}
                {(!isHumanMode || (canForceTakeover && conversation?.assignedToId !== auth?.user?.id)) && (
                  <button className="btn btn-warning btn-sm" disabled={busyKey === "conversation-takeover"} onClick={() => runConversationAction(conversation.id, "takeover", { note: takeoverNote || "" })}>
                    Take Over
                  </button>
                )}
                <button className="btn btn-success btn-sm" disabled={busyKey === "conversation-resolve"} onClick={() => runConversationAction(conversation.id, "resolve", {})}>
                  Resolve
                </button>
              </div>
            )}

            <div 
              className="message-thread chat-messages"
              ref={messageThreadRef}
            >
              {messages.map((message, index) => {
                const type = senderClass(message);
                if (type === "system") {
                  return (
                    <div key={index} className="system-event">
                      {message.text || "System update"}
                    </div>
                  );
                }
                const hasRenderedMedia = Boolean(message?.rawPayload?.media?.url || message?.rawPayload?.image?.url);
                const label = senderLabel(type);
                return (
                  <div key={index} className={`message-row msg ${type}`}>
                    <div className="message-stack">
                      {label ? <div className="msg-label">{label}</div> : null}
                      <div className={`message-bubble msg-bubble ${type}`}>
                        {renderMessageMedia(message)}
                        {String(message.text || "").trim() || !hasRenderedMedia ? (
                          <div className="chat-rich-text">{renderChatMessage(message.text)}</div>
                        ) : null}
                      </div>
                      <div className="msg-meta">{formatTime(message.createdAt)}</div>
                    </div>
                  </div>
                );
              })}
              {isAIThinking && (
                <div className="message-row agent" style={{ display: "flex", justifyContent: "flex-end" }}>
                  <div className="message-bubble ai typing-bubble">
                    <div className="typing-dots" aria-label="AI sedang mengetik">
                      <span />
                      <span />
                      <span />
                    </div>
                    <div style={{ fontSize: "10px", marginTop: "4px", opacity: 0.7, textAlign: "right" }}>AI thinking</div>
                  </div>
                </div>
              )}
            </div>

            <div className="chat-composer">
              <div style={{ background: "#f3f4f6", borderRadius: "12px", padding: "8px" }}>
                {!canSendManualMessage && (
                  <div style={{ padding: "8px 10px", marginBottom: "6px", borderRadius: "8px", background: "#fff7ed", color: "#9a3412", fontSize: "12px", fontWeight: 700 }}>
                    {composer.hint || "Take over conversation before replying manually."}
                  </div>
                )}
                {whatsappWindow ? (
                  <div className={`tickets-window-panel ${whatsappWindow.requiresTemplate ? "warn" : "ok"}`} style={{ margin: "0 0 8px 0" }}>
                    <strong>Aturan follow-up WhatsApp</strong>
                    <span>
                      {whatsappWindow.status === "open"
                        ? `Masih di bawah 24 jam. Free-form boleh dikirim; sisa sekitar ${whatsappWindow.hoursRemaining || 0} jam.`
                        : "Di atas 24 jam atau belum ada inbound tercatat. WhatsApp official wajib approved template; biaya Meta/BSP terpisah dari credit AI."}
                    </span>
                  </div>
                ) : null}
                <textarea
                  ref={manualTextareaRef}
                  style={{ width: "100%", background: "transparent", border: "none", outline: "none", padding: "8px", fontSize: "14px", minHeight: "80px", resize: "none", opacity: canSendManualMessage ? 1 : 0.55 }}
                  placeholder={canSendManualMessage ? "Type a message..." : "Take over first to reply..."}
                  value={manualMessage}
                  onChange={(e) => setManualMessage(e.target.value)}
                  onKeyDown={handleManualMessageKeyDown}
                  disabled={!canSendManualMessage}
                />
                {manualMedia ? (
                  <div className="manual-media-chip">
                    <span>{manualMedia.kind === "image" ? "Image" : "File"}: {manualMedia.fileName}</span>
                    <button type="button" onClick={() => setManualMedia(null)} disabled={!canSendManualMessage}>Remove</button>
                  </div>
                ) : null}
                <div className="composer-toolbar">
                  <div style={{ display: "flex", gap: "10px", color: "var(--text-dim)", alignItems: "center", flexWrap: "wrap" }}>
                    <label className={`composer-attach-button ${canSendManualMessage ? "" : "disabled"}`} title="Lampirkan gambar atau dokumen">
                      {Icons.upload}
                      <span>Attach File</span>
                      <input
                        type="file"
                        accept="image/*,.pdf,.doc,.docx,.txt"
                        style={{ display: "none" }}
                        disabled={!canSendManualMessage}
                        onChange={handleManualMediaChange}
                      />
                    </label>
                    <div className="emoji-picker-shell">
                      <button
                        type="button"
                        className={`composer-icon-button emoji-trigger ${canSendManualMessage ? "" : "disabled"}`}
                        title="Emoji"
                        aria-label="Open emoji picker"
                        aria-expanded={emojiPickerOpen}
                        disabled={!canSendManualMessage}
                        onClick={() => setEmojiPickerOpen((open) => !open)}
                      >
                        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10" /><path d="M8 14s1.5 2 4 2 4-2 4-2" /><line x1="9" y1="9" x2="9.01" y2="9" /><line x1="15" y1="9" x2="15.01" y2="9" /></svg>
                      </button>
                      {emojiPickerOpen && (
                        <div className="emoji-picker" role="menu" aria-label="Emoji picker">
                          {composerEmojis.map((emoji) => (
                            <button
                              key={emoji}
                              type="button"
                              role="menuitem"
                              onMouseDown={(event) => {
                                event.preventDefault();
                                appendManualEmoji(emoji);
                              }}
                              onClick={() => appendManualEmoji(emoji)}
                            >
                              {emoji}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                  <button
                    className="btn btn-primary"
                    style={{ padding: "8px 20px" }}
                    disabled={!canSubmitManualMessage || busyKey === "conversation-manual-message"}
                    onClick={sendManualMessage}
                  >
                    Kirim
                  </button>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <EmptyState title="Pilih percakapan" copy="Pilih kontak di kiri untuk mulai chat." />
        )}
      </section>

      <aside className="ops-column ops-side detail-panel-v2">
        {conversation ? (
          <>
            <div className="mobile-detail-nav">
              <button className="mobile-pane-btn" type="button" onClick={() => setMobilePane("chat")}>
                Kembali ke chat
              </button>
              <strong>Detail percakapan</strong>
            </div>
            <section className="detail-section-v2">
              <div className="detail-label-v2">Kontak</div>
              <div className="customer-mini-profile">
                <div className={`avatar avatar-lg ${conversationAvatarColor}`}>
                  {conversationInitials}
                </div>
                <div>
                  <div className="customer-mini-name">{conversation.contactName || "Contact"}</div>
                  <div className="customer-mini-phone">{conversation.phone || "-"}</div>
                </div>
              </div>
              <div className="crm-actions-row" style={{ marginTop: "12px" }}>
                <button
                  className="btn btn-secondary btn-sm"
                  type="button"
                  onClick={() => openContactFromConversation?.(conversation)}
                  disabled={!conversation.contactId}
                >
                  Buka kontak
                </button>
                {canUseProspects ? (
                  <button
                    className="btn btn-primary btn-sm"
                    type="button"
                    onClick={() => createDealFromConversation?.(conversation)}
                    disabled={busyKey === "deal-from-conversation"}
                  >
                    Buat follow-up
                  </button>
                ) : null}
              </div>
              {canUseProspects && openDeals.length ? (
                <div className="crm-list-stack" style={{ marginTop: "12px" }}>
                  {openDeals.slice(0, 3).map((deal) => (
                    <button className="crm-note-item deal-activity-item as-button" type="button" key={deal.id} onClick={() => openDealFromConversation?.(deal)}>
                      <strong>{deal.title || "Deal aktif"}</strong>
                      <span>{deal.stageName || "Pipeline"} {deal.valueAmount ? `- ${formatCurrencyIDR(deal.valueAmount)}` : ""}</span>
                      <small>{deal.ownerName || "Unassigned"}</small>
                    </button>
                  ))}
                </div>
              ) : null}
            </section>

            <section className="detail-section-v2">
              <div className="detail-label-v2">Status Chat</div>
              <div className="detail-field"><span className="detail-key">Mode</span><span>{modeBadge(conversation.mode)}</span></div>
              <div className="detail-field"><span className="detail-key">WhatsApp</span><span className="detail-val">{conversation.whatsappSession || "-"}</span></div>
              <div className="detail-field"><span className="detail-key">AI Agent</span><span className="detail-val">{conversation.aiAgentName || "-"}</span></div>
              <div className="detail-field"><span className="detail-key">Assigned To</span><span className="detail-val">{conversation.assignedToName || "AI Assistant"}</span></div>
              <div className="detail-field"><span className="detail-key">Priority</span><span className={`badge ${conversation.priority === "urgent" ? "red" : conversation.priority === "high" ? "orange" : "blue"}`}>{conversation.priority || "normal"}</span></div>
              <div className="detail-field"><span className="detail-key">SLA</span><span className="detail-val">{conversation.slaDueAt ? formatDateTime(conversation.slaDueAt) : "-"}</span></div>
              <div className="detail-field"><span className="detail-key">Last inbound</span><span className="detail-val">{conversation.lastCustomerMessageAt ? formatDateTime(conversation.lastCustomerMessageAt) : "-"}</span></div>
              {conversation.escalationReason ? <div className="detail-field"><span className="detail-key">Eskalasi</span><span className="detail-val">{formatTime(conversation.lastMessageAt)}</span></div> : null}
              {conversation.escalationReason ? <div className="detail-field"><span className="detail-key">Reason</span><span className="detail-val text-sm">{conversation.escalationReason}</span></div> : null}
            </section>

            <section className="detail-section-v2 conversation-workflow-panel">
              <div className="detail-label-v2">Workflow CRM</div>
              <div className="workflow-grid">
                <label>Status
                  <select className="form-select" value={workflowDraft.status} onChange={(event) => setWorkflowDraft((current) => ({ ...current, status: event.target.value }))}>
                    <option value="open">Terbuka</option>
                    <option value="pending_human">Menunggu admin</option>
                    <option value="resolved">Selesai</option>
                  </select>
                </label>
                <label>Priority
                  <select className="form-select" value={workflowDraft.priority} onChange={(event) => setWorkflowDraft((current) => ({ ...current, priority: event.target.value }))}>
                    <option value="low">Low</option>
                    <option value="normal">Normal</option>
                    <option value="high">High</option>
                    <option value="urgent">Urgent</option>
                  </select>
                </label>
                <label>Assign
                  <select className="form-select" value={workflowDraft.assignedToId} onChange={(event) => setWorkflowDraft((current) => ({ ...current, assignedToId: event.target.value }))}>
                    <option value="">Unassigned</option>
                    {teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}
                  </select>
                </label>
                <label>SLA due
                  <input className="form-input" type="datetime-local" value={workflowDraft.slaDueAt} onChange={(event) => setWorkflowDraft((current) => ({ ...current, slaDueAt: event.target.value }))} />
                </label>
              </div>
              <textarea className="form-textarea workflow-note" value={workflowDraft.internalNote} onChange={(event) => setWorkflowDraft((current) => ({ ...current, internalNote: event.target.value }))} placeholder="Internal note untuk tim..." />
              <button className="btn btn-secondary btn-sm" type="button" onClick={submitWorkflow} disabled={busyKey === "conversation-workflow"}>Simpan workflow</button>
            </section>

            <section className="detail-section-v2">
              <div className="detail-label-v2">Timeline</div>
              <div className="timeline-list-v2">
                {timelineItems.length ? timelineItems.map((item) => (
                  <div className="timeline-item-v2" key={item.id}>
                    <span className={`timeline-dot-v2 ${item.tone}`} />
                    <div>
                      <div className="timeline-event-text">{item.label}</div>
                      <div className="timeline-event-time">{formatTime(item.at)}</div>
                    </div>
                  </div>
                )) : (
                  <EmptyState title="Belum ada timeline" copy="Aktivitas chat akan muncul setelah percakapan berjalan." compact />
                )}
              </div>
            </section>

            {selectedConversation?.retrieval?.matches?.length ? (
              <section className="detail-section-v2">
                <div className="detail-label-v2">AI Context</div>
                <div className="context-match-list-v2">
                  {selectedConversation.retrieval.matches.slice(0, 3).map((match, i) => (
                    <div className="context-match-v2" key={i}>
                      <strong>{match.sourceTitle || "FAQ"}</strong>
                      <p>{truncateText(match.preview, 96)}</p>
                    </div>
                  ))}
                </div>
              </section>
            ) : null}
          </>
        ) : (
          <EmptyState title="Belum ada yang dipilih" copy="Detail muncul setelah kamu memilih chat." />
        )}
      </aside>

    </div>
    </>
  );
}
