# UX Copy Audit — Dashboard Oneflow

**Tujuan produk:** mempermudah user (termasuk yang berumur / non-teknis) membuat AI agent untuk bisnisnya.
**Masalah inti:** terlalu banyak teks yang "menjelaskan diri sendiri" + jargon teknis + bahasa campur Inggris–Indonesia. User harus *membaca* sebelum tahu *harus klik apa*.

Dokumen ini = rencana review **sebelum** ada perubahan kode. Setiap baris berisi string asli dari `components/dashboard/views/` + usulan revisi. Tidak ada kode yang diubah sampai disetujui.

---

## Prinsip yang dipakai

1. **1 elemen = 1 pesan.** Kartu cukup judul + maksimal 1 baris aksi. Detail "kenapa/bagaimana" pindah ke ikon `?` (tooltip) atau "Pelajari".
2. **Mulai dari kata kerja**, bukan penjelasan. "Tambah dokumen", bukan "Upload file atau tulis dokumen manual…".
3. **Satu bahasa: Indonesia awam.** Tidak ada istilah teknis Inggris yang menclutter UI utama.
4. **Progressive disclosure.** Yang advanced disembunyikan/dilipat sampai dibutuhkan.
5. **Hapus alasan teknis.** User tidak perlu tahu alasan arsitektur ("karena setiap draft mengambil produk aktif…").

---

## 1. Glosarium jargon → bahasa awam (berlaku global)

Cari-ganti terkontrol di seluruh view. Ini dampak tertinggi & risiko terendah.

| Istilah sekarang | Ganti jadi |
|---|---|
| human handoff / handoff | diteruskan ke admin |
| fallback message | pesan tunggu |
| guardrail / policy | aturan aman |
| escalation / eskalasi | minta bantuan admin |
| session (WhatsApp) | koneksi WhatsApp |
| legacy | cara lama |
| binding code | kode penghubung ✅ (sudah dipakai di alur grup) |
| gateway / webhook / AI gateway | (sembunyikan dari user; hanya untuk halaman Status Sistem) |
| embedding | proses belajar AI (atau cukup "diproses") |
| knowledge | pengetahuan / materi |
| long-term memory | ingat pelanggan |
| credit / run | pemakaian / penggunaan |
| lifecycle | tahap pelanggan |
| broadcast | kirim massal |
| draft (pesanan) | draf |
| queue | antrian |
| SLA | batas waktu |
| PIC | penanggung jawab |
| triage | pemilahan otomatis |
| staging | tahap |

---

## 2. Bug konsistensi: string Inggris yang tersisa (InboxView)

Masih ada copy Inggris di tengah dashboard berbahasa Indonesia. Wajib diterjemahkan.

| Sekarang | Usulan |
|---|---|
| "Choose a contact from the left to start chatting." | "Pilih kontak di kiri untuk mulai chat." |
| "Details will appear here once you select a chat." | "Detail muncul setelah kamu memilih chat." |
| "Try adjusting your filters or search." | "Coba ubah filter atau pencarian." |
| "Select a conversation" | "Pilih percakapan" |
| "Attach image or document" | "Lampirkan gambar atau dokumen" |

---

## 3. Per-view: before → after

### 3a. AI CS / Agents (`AgentsView.jsx`) — jalur INTI
| Lokasi | Sekarang | Usulan |
|---|---|---|
| page-desc | "Buat dan kelola AI agent untuk CS, sales, booking, reminder, edukasi, dan lainnya." | "AI yang membalas chat pelanggan otomatis." |
| hint sistem prompt | "Ini panduan utama AI. Jangan hapus batasan penting; cukup ubah bagian yang diberi tanda." | "Panduan utama AI. Ubah bagian bertanda saja." + ikon `?` untuk sisanya |
| banner instruksi | "Ganti bagian yang masih bertanda [ubah bagian ini: ...]. Kalau ragu, cukup isi nama bisnis, jenis bisnis, data yang perlu dikumpulkan, dan kapan AI harus panggil admin." | "Isi bagian bertanda. Minimal: nama & jenis bisnis, data yang dikumpulkan, kapan panggil admin." |
| long-term memory | "Long-term memory menambah proses baca, ekstraksi, dan penyimpanan konteks customer. Ini bisa menambah pemakaian credit…" | Toggle **"Ingat pelanggan"** + ikon `?`: "Menambah biaya. Aktifkan kalau perlu personalisasi." |
| hapus agent | "Knowledge, instruksi, dan konfigurasi khusus agent ini akan dihapus. Riwayat conversation tetap tersimpan…" | "Materi & pengaturan agent ini dihapus. Riwayat chat tetap aman." |
| EmptyState | "Klik AI CS di daftar untuk melihat instruksi, knowledge, dan koneksi WhatsApp." | "Pilih AI CS di daftar untuk lihat detailnya." |

### 3b. WhatsApp (`WhatsAppConnectionView.jsx`) — jalur INTI
| Lokasi | Sekarang | Usulan |
|---|---|---|
| card-subtitle koneksi | "Gunakan koneksi resmi untuk nomor WhatsApp Business yang tetap aktif di aplikasi ponsel." | "Tambah nomor WhatsApp untuk AI kamu." |
| pilihan provider | "Legacy - QR device lama" / "Resmi - WhatsApp Business tetap aktif" | "Scan QR (cara cepat)" / "WhatsApp Business resmi" |
| card-subtitle grup | "Sambungkan satu grup WhatsApp tim sebagai tujuan notifikasi saat AI CS butuh bantuan atau ada pesanan/booking baru." | "Grup tim yang menerima notifikasi dari AI." |
| notif paket trial | "Akun trial bisa pakai Playground untuk tes AI. Pilih paket untuk membuka koneksi WhatsApp production." | "Aktifkan paket untuk menghubungkan WhatsApp. Sebelum itu, coba AI di Coba AI." |

> Catatan: alur "Hubungkan Grup" yang baru (wizard 3 langkah) sudah sesuai prinsip. Pertahankan polanya.

### 3c. Knowledge → "Materi / Pengetahuan" (`KnowledgeView.jsx`) — jalur INTI
| Lokasi | Sekarang | Usulan |
|---|---|---|
| page-desc | "Kelola knowledge terpisah per AI agent." | "Materi yang dipakai AI untuk menjawab." |
| hint FAQ | "FAQ cocok untuk jawaban pendek yang sering ditanyakan customer." | "FAQ: jawaban pendek yang sering ditanya." |
| hint dokumen | "Dokumen cocok untuk info panjang: profil bisnis, kebijakan, SOP, syarat layanan, dan alur order." | "Dokumen: info panjang (profil, kebijakan, SOP)." |
| hint real-time | "Untuk data stok, slot, order, atau booking real-time, pakai Alat Bisnis." | pindah ke ikon `?` |
| subtitle | "Upload file atau tulis dokumen manual langsung untuk agent ini." | "Tambah dokumen" |

### 3d. Playground → "Coba AI" (`PlaygroundView.jsx`) — jalur INTI
| Lokasi | Sekarang | Usulan |
|---|---|---|
| page-desc | "Test AI sebagai pelanggan. Tidak mengirim WhatsApp, tapi AI bisa membuat data test seperti pesanan atau booking kalau izinnya aktif." | "Coba AI seperti pelanggan. Tidak mengirim WhatsApp." |

### 3e. AI Settings → "Aturan AI" (`AISettingsView.jsx`)
| Lokasi | Sekarang | Usulan |
|---|---|---|
| page-desc | "Atur batas perilaku AI, fallback human handoff, dan guardrail berbasis policy bisnis." | "Atur batas jawaban AI dan kapan diteruskan ke admin." |

### 3f. Alat Bisnis (`BusinessToolsView.jsx`)
| Lokasi | Sekarang | Usulan |
|---|---|---|
| page-desc | "Pasang alat tambahan sesuai kebutuhan bisnis. Fitur utama AI CS tetap aktif." | "Tambahkan kemampuan AI: produk, pesanan, booking." |

### 3g. View sekunder/lanjutan (padatkan, prioritas lebih rendah)
| View | Sekarang | Usulan |
|---|---|---|
| Pesanan (Commerce) | "Pesanan ada di dalam Produk & Stok karena setiap draft harus mengambil produk aktif dari katalog." | (hapus — alasan teknis) |
| Ticketing | "Problem customer, status staging, SLA, PIC, dan AI triage dalam satu queue." | "Keluhan pelanggan & penanganannya dalam satu daftar." |
| Health → "Status Sistem" | "Monitoring stabilitas server, webhook, dan AI gateway." | "Status sistem & koneksi." |
| Analytics | "Pantau volume percakapan, eskalasi, penyelesaian, dan performa operasional." | "Ringkasan percakapan & performa AI." |
| Kontak | "Profil customer, riwayat chat, catatan internal, tags, dan follow-up dalam satu halaman." | "Data & riwayat pelanggan." |
| Follow-up (Deals) | "Ringkasan sales dan follow-up yang perlu ditangani tim." | "Peluang penjualan yang perlu ditindaklanjuti." |
| Billing | "Ringkasan kredit, transaksi, kesehatan sistem, dan pemakaian AI." | "Ringkasan pemakaian & tagihan." |

---

## 4. Navigasi: satukan bahasa + kelompokkan

**Bahasa nav** (saat ini campur): Overview→**Ringkasan**, Health→**Status Sistem**, Pricing→**Harga**, Billing→**Tagihan**, Playground→**Coba AI**, Follow-up→**Tindak Lanjut**, Ticketing→**Tiket**, Analytics→**Laporan**.

**Pengelompokan (progressive disclosure):**
```
UTAMA
  AI CS · WhatsApp · Coba AI · Materi/Pengetahuan
OPERASIONAL
  Inbox · Kontak · Alat Bisnis · Produk · Pesanan · Booking
LANJUTAN (lipat, default tertutup)
  Laporan · Tagihan · Tiket · Tindak Lanjut · Status Sistem · Harga
```

**Jalur "Mulai di sini"** (untuk akun baru, di Ringkasan): 1 Buat AI → 2 Hubungkan WhatsApp → 3 Coba AI. (Bisa pakai pola progress-strip yang sudah ada.)

---

## 5. Aksesibilitas untuk user berumur

- Naikkan ukuran font dasar untuk body/copy sekunder (banyak yang 11–12px → minimal 13–14px).
- Perbesar target klik tombol kecil (`btn-sm`) di area onboarding.
- Tambah whitespace antar kartu; kurangi kartu yang berdempetan.
- Pastikan kontras teks abu-abu (`--gray-500`) memenuhi WCAG AA, terutama di subtitle.

---

## 6. Rencana rollout (urut prioritas)

| # | Pekerjaan | Dampak | Usaha | File utama |
|---|---|---|---|---|
| 1 | Glosarium jargon → bahasa awam (§1) + terjemahkan string Inggris (§2) | Tinggi | Rendah | semua view, InboxView |
| 2 | Padatkan page-desc & card-subtitle jalur inti (§3a–3f) | Tinggi | Sedang | Agents, WhatsApp, Knowledge, Playground, AISettings, BusinessTools |
| 3 | Satukan bahasa nav + kelompokkan Utama/Operasional/Lanjutan (§4) | Tinggi | Sedang | dashboard-core (nav), DashboardApp |
| 4 | Jalur "Mulai di sini" di Ringkasan (§4) | Sedang | Sedang | DashboardApp / Overview |
| 5 | Detail panjang → tooltip `?` (§3) | Sedang | Sedang | komponen tooltip + view |
| 6 | Aksesibilitas font/kontras/whitespace (§5) | Sedang | Rendah | globals.css |

**Saran mulai:** langkah 1 dulu (cari-ganti terkontrol, paling cepat terasa), lalu langkah 2 untuk jalur inti.
