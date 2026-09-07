# Recap Pekerjaan — 2026-06-01

Ringkasan detail semua perubahan dashboard yang dikerjakan hari ini. Fokus utama: membuat UX lebih ramah untuk user bisnis (termasuk yang berumur / non-teknis), sesuai tujuan produk "mempermudah membuat AI agent untuk bisnis".

Semua perubahan diverifikasi dengan: parse `esbuild` per file + `npm test` (12/12 pass) + cek brace balance CSS.

---

## 1. Optimasi wizard "Buat AI CS baru" (setup agent)

**File:** `components/dashboard/views/AgentsView.jsx`, `app/(application)/globals.css`

- Wizard pembuatan agent diubah dari 3 langkah (Template → Setup → Review) menjadi **4 langkah** yang lebih jelas:
  1. **Setup agent** — pilih template + isi Nama AI CS (nama dipindah ke sini)
  2. **Bisnis** — nama/jenis bisnis, cerita bisnis, produk, customer, "kapan panggil admin"
  3. **Alat bisnis** — kemampuan aktif + setting produk/booking
  4. **Review** — cek kesiapan lalu buat agent
- Bagian **"Kapan harus panggil admin"** diubah dari textarea bebas menjadi **checkbox kondisi (mengikuti template bisnis)** + **text box opsional** "Kondisi lain".
  - Ditambah `adminOptions` per template di `templateSetupCopy` (ecommerce, booking, klinik, edukasi, properti, travel, b2b, custom, fallback).
  - Data checkbox + teks digabung jadi `operatingNotes` lewat helper `composeOperatingNotes()` — alur ke business context AI tetap sama (tidak mengubah backend).
- State step di-rename `agent`/`business`/`tools`/`review`, gating per langkah, footer & progress bar 4 kolom.
- CSS: progress 4 kolom, layout class `agent-step`/`business-step`/`tools-step`, styling checkbox kondisi admin, dark mode + responsif.

## 2. AI CS — hapus tombol "Buat Agent" yang berlebihan

**File:** `components/dashboard/views/AgentsView.jsx`, `app/(application)/globals.css`

- Saat belum ada agent, sebelumnya muncul **3 tombol** "Buat AI CS/Agent Baru" sekaligus (header, kartu kiri, panel kanan).
- Diganti jadi **satu hero onboarding** terpusat ("Buat AI CS pertama kamu" + preview 4 langkah + 1 tombol). Tombol header disembunyikan saat 0 agent.
- Empty-state internal (hasil filter kosong / belum pilih agent) tidak lagi membawa tombol buat. Tambah CSS `.agents-onboarding`.

## 3. WhatsApp Integration — UX + langkah hubungkan grup

**File:** `components/dashboard/views/WhatsAppConnectionView.jsx`, `app/(application)/globals.css`

- Bagian "Notifikasi Grup" yang tadinya cuma kotak Binding Code + 1 baris hint, awalnya dirombak jadi **step-by-step state-aware** (judul diganti "Hubungkan Agent ke Grup Notifikasi").

## 4. Grup binding → popup wizard (seperti pembuatan agent)

**File:** `components/dashboard/views/WhatsAppConnectionView.jsx`, `app/(application)/globals.css`

- Atas permintaan, langkah hubungkan grup dijadikan **popup wizard 3 langkah** (mirip modal buat agent):
  1. **Siapkan grup** — tambahkan nomor gateway ke grup tim
  2. **Buat kode** — generate kode + salin
  3. **Tempel & selesai** — tempel kode di grup, tunggu konfirmasi (auto sukses saat terhubung)
- Di kartu: belum terhubung → CTA "Hubungkan Grup" (buka modal); sudah terhubung → panel sukses (nama grup, status, Kirim Test, Hapus Grup).
- State `bindModalOpen`/`bindWizardStep`, progress strip, tutup via X/backdrop/Esc, auto-advance saat status jadi terhubung.
- CSS: `.wa-bind-cta`, `.wa-bind-modal`, `.wa-bind-progress`, `.wa-bind-modal-step`, `.wa-bind-tips`, `.wa-bind-waiting` (+ `.wa-group-connected*`).

## 5. Perbaikan error hydration (`bis_skin_checked`)

**File:** `app/(oneflow)/layout.js`

- Error hydration `bis_skin_checked="1"` berasal dari **extension browser** (Bitdefender/antivirus), bukan bug kode.
- Layout dashboard `(application)` sudah punya `suppressHydrationWarning`. Ditambahkan juga ke layout marketing `(oneflow)` (`<html>` + `<body>`) untuk konsistensi.
- Catatan: `suppressHydrationWarning` tidak menurun ke anak-elemen, jadi warning pada `<div>` internal Next.js tidak bisa dihilangkan via kode — solusi sebenarnya matikan/whitelist extension atau pakai incognito.

## 6. Analisa & audit teks dashboard

**File:** `docs/ux-copy-audit.md` (dokumen rencana)

- Audit menyeluruh string UI di `components/dashboard/views/`. Temuan utama: bukan jumlah teks, tapi **jargon teknis + bahasa campur Inggris-Indonesia** yang membuat user tersesat.
- Berisi: glosarium jargon→bahasa awam, daftar string Inggris yang tersisa, tabel before→after per view, usulan struktur nav, catatan aksesibilitas, dan rencana rollout berprioritas.

## 7. Eksekusi perapian copy (de-jargon + terjemah + padatkan)

**File:** `InboxView`, `AgentsView`, `WhatsAppConnectionView`, `KnowledgeView`, `PlaygroundView`, `AISettingsView`, `BusinessToolsView`, `CommerceViews`, `HealthView`, `ContactsView`, `TicketsView`, `AnalyticsViews`

- **§2 — String Inggris → Indonesia** (InboxView): "No conversations", "Select a conversation", "No selection", "Attach image or document", "Try adjusting…", "Choose a contact…".
- **§3 — Padatkan + de-jargon (~30 string)** jadi 1 baris aksi berbahasa awam:
  - AI CS: page-desc, hint sistem prompt, banner instruksi, "Long-term memory"→**"Ingat pelanggan"**, copy hapus agent, empty state.
  - WhatsApp: page-desc & subtitle (buang "legacy/session"), opsi provider, notif trial.
  - Pengetahuan AI (dulu "Knowledge Base"), Coba AI (Playground), Aturan AI (buang "human handoff/guardrail/policy"), Alat Bisnis.
  - Sekunder: Pesanan, Status Sistem (dulu "System Health"), Kontak (dulu "Contact 360"), Tiket (buang "SLA/PIC/triage/queue"), Analytics, Platform Overview.

## 8. Penyatuan bahasa navigasi + regroup menu 3 tingkat

**File:** `lib/dashboard-core.jsx`, `components/dashboard/shell/AppShell.jsx`

- **Label sidebar** disatukan ke Indonesia: Overview→Ringkasan, Billing Control→Kredit, User Analytics→Laporan Pemakaian, Health→Status Sistem, Pricing→Model & Harga, User Management→Tim, Playground→Coba AI, Analytics→Laporan, Billing→Tagihan, Follow-up→Tindak Lanjut, Ticketing→Tiket.
- **`viewMeta`** (23 entri): judul + deskripsi disamakan & dibuang jargon.
- **Label mobile** (AppShell) yang masih Inggris (Test, Billing, Usage, Health, Pricing, Data, Tools) → Indonesia singkat.
- **Judul halaman hardcoded** di view disamakan dengan nav (Agents→AI CS, "WhatsApp Integration"→WhatsApp, "AI Settings"→Aturan AI, dll, termasuk yang pakai ternary).
- **Regroup sidebar 3 tingkat** lewat metadata `group`:
  - Utama (AI CS, WhatsApp, Coba AI) · Operasional (Inbox, Kontak, Alat Bisnis) · Lainnya (Laporan, Tim, Tagihan) · Alat Terinstall (tools terpasang).
  - Render section dinamis + auto-hide jika kosong. Urutan array sumber tidak diubah → halaman default tetap sama.

## 9. Panel "Mulai di sini" (onboarding 3 langkah)

**File:** `components/dashboard/views/AgentsView.jsx`, `app/(application)/globals.css`

- Panel di halaman AI CS: **1 Buat AI CS → 2 Hubungkan WhatsApp → 3 Coba AI**, tiap kartu klik langsung ke aksinya.
- State dinamis (centang hijau saat selesai), **hilang otomatis** setelah punya agent aktif **dan** WhatsApp terhubung.
- CSS `.agents-startguide*` + dark mode + responsif.

## 10. Aksesibilitas (§5)

**File:** `app/(application)/globals.css`

- `page-desc` 13→14px, `card-subtitle` & `form-field-hint` 12→13px + line-height lebih lega.
- Kontras teks abu naik: `--gray-500` → `--gray-600`.
- Label section sidebar 10→11px, kontras naik.

---

## Catatan tindak lanjut

- Verifikasi visual final sebaiknya dilakukan di app yang berjalan (`npm run dev`) — beberapa perubahan layout/aksesibilitas paling pas dicek langsung.
- Detail before→after lengkap ada di `docs/ux-copy-audit.md`.
- Error `bis_skin_checked` tidak bisa hilang lewat kode; minta user uji di incognito/matikan extension.
