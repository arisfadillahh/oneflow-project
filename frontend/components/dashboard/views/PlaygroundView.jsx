import { useEffect, useMemo, useRef } from "react";
import { formatNumber, formatTime, renderChatMessage, truncateText } from "../../../lib/dashboard-core";

// Messages stored before migration 052 can still carry this inbound sender value.
const legacyInboundSenderType = "candidate";

function aiModeLabel(mode) {
  if (mode === "read") return "baca data";
  if (mode === "draft") return "siapkan draf";
  if (mode === "action") return "bantu otomatis";
  return "belum aktif";
}

function installedBusinessTools(tools = []) {
  return tools.filter((tool) => Boolean(tool?.installed ?? tool?.enabled));
}

function decisionClass(decision) {
  if (decision === "escalate") return "escalate";
  if (decision === "error") return "error";
  return "answer";
}

function decisionLabel(decision) {
  if (decision === "escalate") return "Dialihkan ke tim";
  if (decision === "error") return "Belum berhasil";
  return "Dijawab AI";
}

function usageCredits(message) {
  return message?.usage?.creditsUsed ?? message?.usage?.credits_used ?? message?.credits ?? null;
}

function messageText(message) {
  return message?.text || message?.answerText || message?.answer_text || "";
}

function messageConfidence(message) {
  const value = Number(message?.confidence ?? message?.confidenceScore ?? message?.confidence_score ?? 0);
  return Number.isFinite(value) ? value : 0;
}

function expectedBehaviorHint(text) {
  const value = String(text || "").toLowerCase();
  if (!value) return "";
  if (value.includes("eskalasi") || value.includes("handoff") || value.includes("admin") || value.includes("tim")) return "escalate";
  if (value.includes("jawab") || value.includes("knowledge") || value.includes("faq")) return "answer";
  return "";
}

function buildAnswerCuration(message, expectedBehavior = "") {
  if (!message) return null;
  const decision = message.decision || (message.status === "error" ? "error" : "");
  const answer = messageText(message).trim();
  const confidence = messageConfidence(message);
  const retrieval = message.retrieval || {};
  const matchedDocuments = retrieval?.matches ?? retrieval?.documents ?? [];
  const businessAction = businessActionFromRetrieval(retrieval);
  const expectedHint = expectedBehaviorHint(expectedBehavior);
  const issues = [];
  const passes = [];

  if (decision === "error" || message.status === "error") {
    issues.push("AI belum berhasil menjawab. Cek koneksi AI service, kredit, atau data agent.");
  }
  if (!answer) {
    issues.push("Jawaban kosong. Tambahkan instruksi fallback atau cek response AI.");
  }
  if (decision === "answer") {
    if (confidence > 0 && confidence < 0.55) {
      issues.push("Confidence rendah. Tambahkan FAQ/dokumen yang lebih eksplisit.");
    }
    if (!matchedDocuments.length && !businessAction) {
      issues.push("Tidak terlihat knowledge atau tool bisnis yang dipakai. Pastikan jawaban tidak mengarang.");
    } else {
      passes.push("Jawaban punya sumber dari Knowledge atau Alat Bisnis.");
    }
  }
  if (decision === "escalate") {
    passes.push("AI memilih panggil tim, cocok untuk kasus yang butuh keputusan manusia.");
    if (expectedHint === "answer") {
      issues.push("Shortcut ini kelihatannya diharapkan dijawab AI. Tambahkan knowledge jika seharusnya tidak handoff.");
    }
  }
  if (expectedHint === "escalate" && decision === "answer") {
    issues.push("Expected behavior mengarah ke eskalasi, tapi AI menjawab sendiri. Perketat aturan handoff.");
  }
  if (businessAction) {
    if (businessAction.tone === "green") {
      passes.push(`Aksi plugin terbaca: ${businessAction.label}.`);
    } else if (businessAction.tone === "orange") {
      issues.push(`Plugin belum jalan penuh: ${businessAction.label}. Pastikan data customer sudah lengkap.`);
    } else if (businessAction.tone === "red") {
      issues.push(`Aksi plugin gagal: ${businessAction.label}. Cek data Alat Bisnis.`);
    }
  }
  if (answer.length > 900) {
    issues.push("Jawaban terlalu panjang untuk chat. Ringkas prompt agar AI lebih mudah dibaca customer.");
  }
  if (!issues.length && answer) {
    passes.push("Format jawaban cukup aman untuk lanjut diuji ke customer.");
  }

  let tone = "green";
  let label = "Aman";
  if (decision === "error" || issues.length >= 2) {
    tone = "red";
    label = "Perlu revisi";
  } else if (issues.length === 1) {
    tone = "orange";
    label = "Perlu dicek";
  }
  return {
    tone,
    label,
    issues,
    passes,
    nextAction: issues.length
      ? "Perbaiki Knowledge, instruksi agent, atau aturan handoff lalu test ulang."
      : "Simpan sebagai contoh jawaban aman, lalu lanjut tes shortcut lain.",
  };
}

function numberValue(value) {
  return Number(value ?? 0) || 0;
}

function walletCreditRemaining(wallet) {
  return (
    numberValue(wallet?.monthlyCreditsRemaining ?? wallet?.monthlyRemaining) +
    numberValue(wallet?.additionalCreditsRemaining ?? wallet?.additionalRemaining)
  );
}

function playgroundCreditNotice(wallet) {
  if (!wallet) return null;
  const remaining = walletCreditRemaining(wallet);
  if (remaining <= 0) {
    return {
      tone: "danger",
      title: "Kredit test habis",
      text: "AI belum bisa berjalan, jadi tool tidak akan membuat pesanan atau booking sampai kredit ditambah.",
    };
  }
  if (remaining <= 5) {
    return {
      tone: "warn",
      title: `${formatNumber(remaining)} Cr test tersisa`,
      text: "Playground memakai kredit dan Mode Aksi bisa membuat data test real di dashboard.",
    };
  }
  return null;
}

function businessActionStatusLabel(tool, action, toolCalled) {
  if (!toolCalled || action === "not_run") return "Belum ada tool yang jalan";
  if (tool === "create_order_draft" && action === "draft_created") return "Order draft dibuat";
  if (tool === "update_order_draft" && action === "updated") return "Order draft diupdate";
  if (tool === "confirm_order_draft" && action === "confirmed") return "Order dikonfirmasi";
  if (tool === "cancel_order_draft" && action === "cancelled") return "Order dibatalkan";
  if (tool === "create_booking" && action === "appointment_created") return "Booking dibuat";
  if (tool === "confirm_booking" && action === "confirmed") return "Booking dikonfirmasi";
  if (tool === "cancel_booking" && action === "cancelled") return "Booking dibatalkan";
  if (tool === "reschedule_booking" && action === "rescheduled") return "Booking dipindahkan";
  if (action === "read") return "Data tool dibaca";
  if (String(action || "").startsWith("needs_")) return "Belum ada aksi: butuh data lagi";
  if (action === "blocked_ai_mode") return "Belum ada aksi: mode belum mengizinkan";
  if (action === "blocked") return "Aksi gagal";
  if (action === "no_match") return "Data tidak ditemukan";
  return action || "Status tool tidak diketahui";
}

function businessActionFromRetrieval(retrieval) {
  const orchestrator = retrieval?.orchestrator || {};
  const businessTools = retrieval?.businessTools || {};
  if (orchestrator?.queryType !== "business_tool" && !businessTools?.result) return null;
  const action = orchestrator.action || "-";
  const tool = orchestrator.tool || "-";
  const result = businessTools.result || {};
  const toolCalled = orchestrator.toolCalled !== false;
  let label = businessActionStatusLabel(tool, action, toolCalled);
  let tone = "gray";
  if (!toolCalled || action === "not_run") {
    tone = "gray";
  } else if (result.error || action === "blocked") {
    tone = "red";
  } else if (toolCalled && ["draft_created", "appointment_created", "updated", "confirmed", "cancelled", "rescheduled", "read"].includes(action)) {
    tone = "green";
  } else if (toolCalled && ["blocked", "needs_product_selection", "needs_service_selection", "needs_time", "no_match", "blocked_ai_mode"].includes(action)) {
    tone = "orange";
  }
  return { action, tool, result, toolCalled, label, tone };
}

function businessToolLabel(tool) {
  if (tool === "create_order_draft") return "Draft pesanan";
  if (tool === "confirm_order_draft") return "Confirm pesanan";
  if (tool === "cancel_order_draft") return "Cancel pesanan";
  if (tool === "update_order_draft") return "Update pesanan";
  if (tool === "check_stock") return "Cek produk/stok";
  if (tool === "create_booking") return "Buat booking";
  if (tool === "confirm_booking") return "Confirm booking";
  if (tool === "cancel_booking") return "Cancel booking";
  if (tool === "reschedule_booking") return "Reschedule booking";
  return tool || "-";
}

function PlaygroundMessage({ message, customerName }) {
  const isUser = message.role === "user" || ["customer", legacyInboundSenderType].includes(message.senderType);
  const isSystem = message.role === "system";
  const credits = usageCredits(message);
  const businessAction = !isUser ? businessActionFromRetrieval(message.retrieval) : null;

  if (isSystem) {
    return <div className="pg-system-note">{messageText(message)}</div>;
  }

  return (
    <div className={`pg-message-block ${isUser ? "user" : decisionClass(message.decision) === "error" ? "error" : "ai"}`}>
      <div className={`pg-label ${isUser ? "user-label" : "ai-label"}`}>{isUser ? customerName || "Pelanggan Simulasi" : "Jawaban AI"}</div>
      <div className={`pg-msg ${isUser ? "user" : decisionClass(message.decision) === "error" ? "error" : "ai"}`}>
        <div className="chat-rich-text">{renderChatMessage(messageText(message))}</div>
        {!isUser && message.decision && message.decision !== "error" && (
          <div><span className={`pg-decision ${decisionClass(message.decision)}`}>{decisionLabel(message.decision)}</span></div>
        )}
        {!isUser && message.decision && message.decision !== "error" && (
          <div>
            <span className={`pg-decision tool ${businessAction ? businessAction.tone : "gray"}`}>
              {businessAction ? businessAction.label : "Belum ada tool yang jalan"}
            </span>
          </div>
        )}
      </div>
      {!isUser && credits !== null && credits !== undefined && (
        <div className="pg-credit-row">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4" /><path d="M3 5v14a2 2 0 0 0 2 2h16v-5" /><path d="M18 12a2 2 0 0 0 0 4h4v-4Z" /></svg>
          <span>{credits} kredit</span>
        </div>
      )}
      <div className={`pg-time ${isUser ? "right" : ""}`}>{formatTime(message.createdAt)}</div>
    </div>
  );
}

export function PlaygroundView({
  playgroundForm,
  setPlaygroundForm,
  playgroundMessages,
  aiAgents = [],
  playgroundShortcuts,
  playgroundBatchResults,
  sendPlaygroundMessage,
  clearPlaygroundChat,
  runPlaygroundBatch,
  sendAnswerCurationNotification,
  answerCurationNotificationRule,
  isNotificationGroupBound = false,
  notificationGroupName = "",
  busyKey,
  businessTools = [],
  wallet,
}) {
  const threadRef = useRef(null);

  useEffect(() => {
    if (threadRef.current) {
      threadRef.current.scrollTop = threadRef.current.scrollHeight;
    }
  }, [playgroundMessages, busyKey]);

  const lastDebugMessage = [...playgroundMessages].reverse().find((message) => message.role === "assistant" && (message.decision || message.retrieval || message.usage));
  const lastUserMessage = [...playgroundMessages].reverse().find((message) => message.role === "user");
  const selectedAIAgent = aiAgents.find((agent) => agent.id === playgroundForm.aiAgentId);
  const lastDebug = lastDebugMessage ? {
    decision: lastDebugMessage.decision,
    creditsUsed: usageCredits(lastDebugMessage) ?? "-",
    latencyMs: lastDebugMessage.latencyMS ?? lastDebugMessage.latencyMs ?? "-",
    matchedDocuments: lastDebugMessage.retrieval?.matches ?? lastDebugMessage.retrieval?.documents ?? [],
    businessAction: businessActionFromRetrieval(lastDebugMessage.retrieval),
  } : null;
  const lastCuration = buildAnswerCuration(lastDebugMessage);
  const answerCurationEnabled = answerCurationNotificationRule?.isEnabled !== false;
  const isSendingCuration = busyKey === "answer-curation-notification";

  const sessionTotals = useMemo(() => {
    const assistantMessages = playgroundMessages.filter((message) => message.role === "assistant");
    return {
      messages: playgroundMessages.filter((message) => message.role !== "system").length,
      credits: assistantMessages.reduce((sum, message) => sum + (Number(usageCredits(message)) || 0), 0),
      escalations: assistantMessages.filter((message) => message.decision === "escalate").length,
    };
  }, [playgroundMessages]);
  const activeBusinessTools = useMemo(() => installedBusinessTools(businessTools), [businessTools]);
  const creditNotice = playgroundCreditNotice(wallet);
  const sendLastCurationToGroup = () => {
    if (!lastCuration || !lastDebugMessage) return;
    sendAnswerCurationNotification?.({
      aiAgentName: selectedAIAgent?.name || "AI Agent",
      customerName: playgroundForm.customerName || "Pelanggan Test",
      question: messageText(lastUserMessage),
      answerText: messageText(lastDebugMessage),
      decision: decisionLabel(lastDebugMessage.decision),
      tone: lastCuration.tone,
      statusLabel: lastCuration.label,
      issues: lastCuration.issues,
      passes: lastCuration.passes,
      nextAction: lastCuration.nextAction,
      confidence: messageConfidence(lastDebugMessage),
      creditsUsed: usageCredits(lastDebugMessage),
    });
  };

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Coba AI</div>
          <div className="page-desc">Coba AI seperti pelanggan. Tidak mengirim WhatsApp.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" onClick={clearPlaygroundChat}>Bersihkan Chat</button>
          <button className="btn btn-primary" onClick={runPlaygroundBatch} disabled={!playgroundForm.aiAgentId || busyKey === "playground-batch"}>Tes Semua Shortcut</button>
        </div>
      </div>

      {activeBusinessTools.length ? (
        <div className="business-ai-banner playground-ai-banner">
          <div>
            <strong>Izin AI untuk alat bisnis</strong>
            <span>
              Playground mengikuti izin Alat Bisnis:{" "}
              {activeBusinessTools.map((tool) => `${tool.name || tool.key}: ${aiModeLabel(tool.aiMode || "off")}`).join(", ")}.
              {" "}Tes ini memakai kredit dan bisa membuat data real di dashboard jika izin bantu otomatis aktif.
            </span>
          </div>
        </div>
      ) : null}

      {creditNotice ? (
        <div className={`playground-credit-notice ${creditNotice.tone}`}>
          <strong>{creditNotice.title}</strong>
          <span>{creditNotice.text}</span>
        </div>
      ) : null}

      <div className="playground-layout">
        <div className="pg-panel">
          <div className="card pg-scroll-card">
            <div className="card-title mb-12">Konfigurasi Test</div>
            <div className="form-group">
              <label className="form-label">AI Agent</label>
              <select
                className="form-select"
                value={playgroundForm.aiAgentId || ""}
                onChange={(event) => setPlaygroundForm({ ...playgroundForm, aiAgentId: event.target.value })}
              >
                <option value="">Pilih AI Agent</option>
                {aiAgents.filter((agent) => agent.isActive !== false).map((agent) => (
                  <option key={agent.id} value={agent.id}>{agent.name}</option>
                ))}
              </select>
            </div>
            <div className="form-group">
              <label className="form-label">Nama Pelanggan Simulasi</label>
              <input className="form-input" value={playgroundForm.customerName || ""} onChange={(event) => setPlaygroundForm({ ...playgroundForm, customerName: event.target.value })} />
            </div>
            <div className="form-group">
              <label className="form-label">Arah Respons</label>
              <select className="form-select" value={playgroundForm.forceDecision || ""} onChange={(event) => setPlaygroundForm({ ...playgroundForm, forceDecision: event.target.value })}>
                <option value="">Otomatis</option>
                <option value="answer">Paksa dijawab AI</option>
                <option value="escalate">Paksa dialihkan ke tim</option>
              </select>
            </div>
            <div className="divider" />
            <div className="pg-section-label">Shortcut Pertanyaan</div>
            <div className="pg-shortcuts">
              {playgroundShortcuts.map((shortcut, index) => {
                const text = shortcut.question || shortcut.text || "";
                return (
                  <button key={shortcut.id || index} className="btn btn-secondary btn-sm pg-shortcut-btn" onClick={() => setPlaygroundForm({ ...playgroundForm, messageText: text })}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="13 17 18 12 13 7" /><polyline points="6 17 11 12 6 7" /></svg>
                    <span>{shortcut.label || truncateText(text, 28)}</span>
                  </button>
                );
              })}
              {!playgroundShortcuts.length && <div className="text-xs text-muted">Belum ada shortcut aktif.</div>}
            </div>
          </div>
        </div>

        <div className="pg-panel">
          <div ref={threadRef} className="pg-messages">
            {playgroundMessages.map((message, index) => (
              <PlaygroundMessage key={message.id || index} message={message} customerName={playgroundForm.customerName} />
            ))}
            {busyKey === "playground" && (
              <div className="pg-message-block ai">
                <div className="pg-label ai-label">Jawaban AI</div>
                <div className="pg-msg ai"><div className="typing-dots"><span></span><span></span><span></span></div></div>
              </div>
            )}
            {!playgroundMessages.length && <div className="pg-system-note">Mulai percakapan sebagai pelanggan</div>}
          </div>
          <div className="pg-composer">
            <textarea
              data-tour="playground"
              data-tour-id="agent-playground-composer"
              className="form-textarea pg-input"
              rows="2"
              placeholder="Ketik pesan sebagai pelanggan..."
              value={playgroundForm.messageText || ""}
              onChange={(event) => setPlaygroundForm({ ...playgroundForm, messageText: event.target.value })}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  sendPlaygroundMessage();
                }
              }}
            />
            <button
              className="pg-send-icon-btn"
              type="button"
              aria-label="Kirim ke AI"
              title="Kirim ke AI"
              onClick={sendPlaygroundMessage}
              disabled={!playgroundForm.aiAgentId || !String(playgroundForm.messageText || "").trim() || busyKey === "playground"}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="22" y1="2" x2="11" y2="13" /><polygon points="22 2 15 22 11 13 2 9 22 2" /></svg>
            </button>
          </div>
        </div>

        <div className="pg-panel">
          <div className="card pg-scroll-card">
            <div className="card-title mb-12">Ringkasan Respons</div>
            <div className="pg-debug-stack">
              <div className="pg-debug-card">
                <div className="pg-debug-title">RESPONS TERAKHIR</div>
                {lastDebug ? (
                  <>
                    <div className="flex-between text-sm mb-8"><span className="text-muted">Hasil</span><span className={`badge ${lastDebug.decision === "escalate" ? "orange" : lastDebug.decision === "error" ? "red" : "green"}`}>{decisionLabel(lastDebug.decision)}</span></div>
                    {lastDebug.businessAction ? (
                      <>
                        <div className="flex-between text-sm mb-8"><span className="text-muted">Aksi Bisnis</span><span className={`badge ${lastDebug.businessAction.tone}`}>{lastDebug.businessAction.label}</span></div>
                        <div className="flex-between text-sm mb-8"><span className="text-muted">Tool</span><strong>{businessToolLabel(lastDebug.businessAction.tool)}</strong></div>
                      </>
                    ) : (
                      <div className="flex-between text-sm mb-8"><span className="text-muted">Aksi Bisnis</span><span className="badge gray">Belum ada tool yang jalan</span></div>
                    )}
                    <div className="flex-between text-sm mb-8"><span className="text-muted">Kredit Dikonsumsi</span><strong className="pg-credit-text">{lastDebug.creditsUsed} kredit</strong></div>
                    <div className="flex-between text-sm"><span className="text-muted">Waktu Respons</span><strong>{lastDebug.latencyMs}ms</strong></div>
                  </>
                ) : (
                  <div className="text-xs text-muted">Jalankan percakapan untuk melihat response.</div>
                )}
              </div>

              <div className="pg-debug-card">
                <div className="pg-debug-title">TOTAL SESI</div>
                <div className="flex-between text-sm mb-8"><span className="text-muted">Pesan</span><strong>{sessionTotals.messages}</strong></div>
                <div className="flex-between text-sm mb-8"><span className="text-muted">Total Kredit</span><strong className="pg-credit-text">{sessionTotals.credits} kredit</strong></div>
                <div className="flex-between text-sm"><span className="text-muted">Dialihkan ke Tim</span><strong className="pg-credit-text">{sessionTotals.escalations}</strong></div>
              </div>

              <div className="pg-debug-card">
                <div className="pg-debug-title">KNOWLEDGE TERPAKAI</div>
                {lastDebug?.matchedDocuments?.length ? lastDebug.matchedDocuments.map((document, index) => (
                  <div key={document.id || index} className="pg-knowledge-match">
                    <strong>{document.title || document.documentTitle || `Knowledge ${index + 1}`}</strong>
                    <div>{truncateText(document.contentSnippet || document.snippet || document.text || "", 80)}</div>
                  </div>
                )) : <div className="text-xs text-muted">Tidak ada context ter-match.</div>}
              </div>

              <div className={`pg-curation-card ${lastCuration?.tone || "gray"}`}>
                <div className="pg-curation-head">
                  <div>
                    <div className="pg-debug-title">KURASI JAWABAN</div>
                    <strong>{lastCuration ? lastCuration.label : "Belum dites"}</strong>
                  </div>
                  <span className={`badge ${lastCuration?.tone || "gray"}`}>{lastCuration?.label || "Menunggu"}</span>
                </div>
                {lastCuration ? (
                  <>
                    <div className="pg-curation-list">
                      {(lastCuration.issues.length ? lastCuration.issues : lastCuration.passes).slice(0, 3).map((item, index) => (
                        <div key={`${item}-${index}`} className={lastCuration.issues.length ? "issue" : "pass"}>{item}</div>
                      ))}
                    </div>
                    <p>{lastCuration.nextAction}</p>
                    <div className="pg-curation-actions">
                      <span>{isNotificationGroupBound ? `Grup: ${notificationGroupName || "siap"}` : "Grup notifikasi belum terhubung"}</span>
                      <button
                        className="btn btn-secondary btn-sm"
                        type="button"
                        onClick={sendLastCurationToGroup}
                        disabled={!sendAnswerCurationNotification || !isNotificationGroupBound || !answerCurationEnabled || isSendingCuration}
                        title={!answerCurationEnabled ? "Aktifkan event Kurasi jawaban AI di Notifikasi Grup" : undefined}
                      >
                        {isSendingCuration ? "Mengirim..." : "Kirim kurasi ke grup"}
                      </button>
                    </div>
                  </>
                ) : (
                  <p>Jalankan satu pertanyaan dulu. Panel ini akan bantu cek apakah jawaban AI aman, kurang data, atau perlu handoff.</p>
                )}
              </div>

              <div>
                <div className="pg-debug-title">HASIL TES SHORTCUT</div>
                <div className="pg-batch-list">
                  {playgroundBatchResults.map((result, index) => {
                    const decision = result.decision || result.expectedBehavior || "-";
                    const credits = result.creditsUsed ?? result.credits_used ?? result.usage?.creditsUsed ?? result.usage?.credits_used ?? "-";
                    const curation = buildAnswerCuration(result, result.expectedBehavior);
                    return (
                      <div key={result.id || index} className="pg-batch-row">
                        <div>
                          <span>{result.label || result.question || `Tes ${index + 1}`}</span>
                          {result.expectedBehavior ? <small>{truncateText(result.expectedBehavior, 64)}</small> : null}
                        </div>
                        <span className={`pg-decision ${decisionClass(decision)}`}>{decisionLabel(decision)}</span>
                        <span className={`badge ${curation?.tone || "gray"}`}>{curation?.label || "Cek"}</span>
                        <strong>{credits} kr</strong>
                      </div>
                    );
                  })}
                  {!playgroundBatchResults.length && <div className="text-xs text-muted">Belum ada hasil tes.</div>}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
