# Audit Kesiapan Production Oneflow.id

Tanggal: 5 September 2026.

## Keputusan

**NO-GO untuk rilis umum dengan seluruh fitur yang dipromosikan saat ini.** Aplikasi memiliki fondasi yang berjalan, tetapi masih ada risiko pada kapasitas server, booking, konfigurasi agent, dan janji follow-up otomatis. Status container healthy dan build lulus tidak cukup untuk menyatakan alur bisnis siap production.

Laporan ini mengevaluasi source kandidat frontend/backend lokal serta pemeriksaan langsung domain oneflow.id dan server Ubuntu. Source lokal memiliki perubahan belum di-commit; temuan source tidak otomatis membuktikan bahwa setiap baris tersebut sama dengan image production. Tidak dilakukan deploy, transaksi pembayaran, pengiriman WhatsApp, atau pemanggilan model berbayar.

## Temuan Prioritas

### 1. [P1, runtime] Disk server hampir habis

Root filesystem berukuran 20 GB mencapai 100% saat dua container pengujian Go mengompilasi dependensi, menghasilkan `write /dev/stdout: no space left on device`. Pengujian terpisah tersebut ikut meningkatkan penggunaan disk; penggunaan sebelum kompilasi tidak dicatat, sehingga angka 100% tidak boleh dianggap kondisi awal.

Setelah container pengujian keluar otomatis dan file audit sementara dibersihkan, filesystem masih 98% terpakai dengan sekitar 505 MB tersisa. Container aplikasi tetap berjalan; pemeriksaan akhir `/` dan `/health` menghasilkan 200.

**Dampak:** pertumbuhan database, log, upload knowledge, atau build berikutnya dapat kehabisan ruang. Ini blocker operasional paling mendesak.

**Perbaikan:** tambah kapasitas disk, ukur pertumbuhan data, terapkan rotasi log dan retensi image yang mempertahankan rollback, serta pindahkan kompilasi ke CI/build runner. Jangan melakukan prune volume secara massal.

**Syarat lulus:** ruang bebas mencukupi pertumbuhan dan satu siklus deploy/rollback; warning sebelum disk penuh; upload, write database, dan restart terkontrol berhasil setelah penanganan kapasitas.

### 2. [P1, source] Booking berpotensi menerima dua pelanggan pada slot sama

`createBookingAppointment` memanggil pemeriksaan konflik lalu melakukan INSERT sebagai operasi terpisah. Jalur update melakukan pola yang sama. Tidak ditemukan exclusion constraint atau penguncian slot dalam migration booking yang diperiksa.

Bukti: [booking.go:421](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/booking.go:421>), [booking.go:446](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/booking.go:446>), [migration booking:29](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/db/migrations/037_booking_plugin.sql:29>).

**Skenario:** request A dan B memeriksa slot sebelum salah satunya INSERT; keduanya melihat slot kosong dan dapat berhasil. Ini temuan race condition dari inspeksi kode, belum direproduksi dengan database uji pada audit ini.

**Perbaikan:** tegakkan invariant slot di database atau gunakan transaksi dengan penguncian yang benar untuk create maupun reschedule. Kembalikan konflik yang dapat dipahami UI.

**Syarat lulus:** dua request paralel untuk slot yang bertabrakan menghasilkan tepat satu keberhasilan; slot berurutan yang valid tetap diterima; tenant berbeda tidak saling mengunci.

### 3. [P1, source] Jam operasional dan buffer booking belum ditegakkan

Layanan menyimpan `availability` dan `buffer_minutes`, tetapi pengambilan schedule untuk membuat appointment tidak mengambil availability. Buffer dibaca tetapi tidak dipakai dalam rentang konflik. Validasi create/update yang diperiksa juga tidak menolak waktu lampau.

Bukti: [booking.go:596](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/booking.go:596>), [booking.go:606](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/booking.go:606>).

**Dampak:** pengaturan jadwal terlihat tersedia, tetapi appointment dapat dibuat di luar jam layanan atau tanpa jeda yang sudah diatur.

**Perbaikan:** satu validator jadwal untuk dashboard dan aksi AI: timezone, hari aktif, jam buka/tutup, buffer, durasi, dan waktu lampau. Bila admin boleh override, buat tindakan eksplisit dan tercatat.

**Syarat lulus:** jam tutup, hari libur, overlap buffer, dan waktu lampau ditolak; timezone dan reschedule diuji.

### 4. [P1, source] Setup agent baru dapat mengubah izin tool seluruh organisasi

Setup agent melakukan PATCH `/api/business-tools/{key}` dengan `enabled: true` dan mode pilihan/default `draft`. Backend menyimpan mode pada `organization_modules`, menggunakan kunci organisasi dan modul, bukan agent.

Bukti: [DashboardApp.jsx:1344](<C:/Users/Aris Fadillah/Downloads/oneflow-frontend/components/dashboard/DashboardApp.jsx:1344>), [business_tools.go:539](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/business_tools.go:539>).

**Skenario:** tool commerce agent lama sudah `action`; pengguna membuat agent baru dengan default `draft`; izin commerce organisasi ikut menjadi `draft`. Sebaliknya, modul yang sebelumnya dimatikan dapat diaktifkan saat setup agent lain.

**Perbaikan:** putuskan kepemilikan konfigurasi secara eksplisit. Jika izin bersifat per-agent, simpan mapping agent-tool. Jika memang global, onboarding harus mempertahankan konfigurasi lama dan menjelaskan dampak perubahan lintas-agent.

**Syarat lulus:** membuat agent B tidak mengubah perilaku agent A tanpa tindakan perubahan workspace yang eksplisit.

### 5. [P1, gap fungsi] Klaim follow-up otomatis belum didukung dispatcher yang ditemukan

Hero production menyebut AI melakukan follow-up otomatis. Source memiliki tugas follow-up, due date, status, dan ringkasan overdue, tetapi pencarian referensi `follow_up_tasks` hanya menemukan CRUD dan reporting. Loop worker berisi heartbeat, reset bulanan, knowledge hydration, chunking, embeddings, dan alerting; tidak ditemukan pengambilan tugas jatuh tempo untuk dikirim sebagai pesan.

Bukti: [crm.go:1280](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/crm.go:1280>), [worker main.py:789](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/worker/app/main.py:789>), [hero production](https://oneflow.id/).

**Dampak:** pelanggan dapat mengira membuat tugas berarti pesan akan terkirim otomatis. Penilaian ini berlaku pada layanan/source yang diperiksa; dispatcher eksternal belum dibuktikan.

**Perbaikan:** tentukan tugas pengingat admin versus pesan otomatis sebagai dua perilaku berbeda. Untuk pesan otomatis diperlukan job persisten, claim/locking, retry terbatas, deduplikasi, pembatalan ketika customer membalas, opt-out, validasi window/template saat eksekusi, serta pencatatan biaya dan hasil kirim. Aturan channel dan harga harus diverifikasi lagi ke dokumentasi provider saat implementasi.

**Syarat lulus:** job bertahan setelah restart, tidak mengirim ganda, batal ketika tidak relevan, tidak melewati kebijakan channel, dan biaya tidak dicatat ganda saat retry.

### 6. [P1, verification] CI dapat hijau ketika pengujian database terlewati

Helper test backend memanggil `t.Skip` jika `APP_BACKEND_TEST_DSN` tidak tersedia. Workflow CI menjalankan `go test ./...` tanpa menyediakan database/DSN. Test tenant isolation menggunakan helper tersebut. CI Python hanya melakukan compile, bukan menjalankan test persona.

Bukti: [server_test.go:81](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/server_test.go:81>), [tenant_isolation_test.go:36](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/app-backend/internal/httpapi/tenant_isolation_test.go:36>), [backend-ci.yml:26](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/.github/workflows/backend-ci.yml:26>).

**Dampak:** badge hijau dapat memberi keyakinan keliru untuk tenant isolation, transaksi, dan alur yang membutuhkan database. Audit ini tidak menyatakan ada kebocoran tenant; yang terbukti adalah celah verifikasi.

**Perbaikan:** sediakan database terisolasi dan migrations dalam CI, jalankan suite integrasi wajib tanpa skip, serta unit test AI. Tambahkan browser E2E untuk alur bisnis penting; test frontend saat ini sebagian besar memeriksa pola source/CSS.

**Syarat lulus:** laporan CI membedakan unit/integrasi/E2E; test kritis tidak boleh skip; gagal bila DB tidak siap.

### 7. [P1, runtime] Alert eksternal belum dikonfigurasi; pemulihan data belum terbukti

Environment container worker menunjukkan `ALERT_WEBHOOK_URL`, `ALERT_EMAIL_SMTP_HOST`, dan `ALERT_EMAIL_TO` kosong. Timer systemd dan crontab pengguna yang diperiksa tidak menunjukkan backup aplikasi. Ditemukan file backup pemulihan lama, tetapi itu bukan bukti backup terjadwal atau restore berhasil.

Bukti kode dukungan alert: [worker main.py:25](<C:/Users/Aris Fadillah/Downloads/oneflow-backend/apps/worker/app/main.py:25>).

**Dampak:** gangguan dapat diketahui dari keluhan pengguna. Keberadaan backup di jalur lain, root cron, atau layanan eksternal belum diverifikasi; jangan menganggap tidak ada backup sama sekali.

**Perbaikan:** aktifkan notifikasi ke kanal operasional, monitor dari luar host untuk tunnel/domain, jadwalkan backup database dan object storage ke lokasi terpisah, tetapkan target kehilangan data dan waktu pemulihan, lakukan restore drill.

**Syarat lulus:** simulasi alert benar-benar diterima; backup baru terverifikasi; restore ke lingkungan kosong menghasilkan data yang dapat dibaca aplikasi.

### 8. [P2, source + UI] Pemulihan password belum berfungsi

Login production menampilkan "Lupa password?" sebagai teks. Source menggunakan `<span>` tanpa link/handler; route yang ditemukan adalah reset password oleh admin, bukan recovery mandiri. Form registrasi juga belum meminta email pemulihan.

Bukti: [AppShell.jsx:97](<C:/Users/Aris Fadillah/Downloads/oneflow-frontend/components/dashboard/shell/AppShell.jsx:97>).

**Perbaikan:** sediakan kanal recovery terverifikasi, token sekali pakai dengan masa berlaku, pembatasan percobaan, dan pesan netral. Sampai tersedia, arahkan secara jujur ke dukungan dengan prosedur verifikasi kepemilikan.

**Syarat lulus:** pemilik bisnis yang lupa password dapat pulih tanpa memberikan akses ke orang lain; token kedaluwarsa/dipakai ulang ditolak.

### 9. [P2, source] Setup agent dapat selesai sebagian dan kehilangan draf

Setelah agent tersimpan, modal ditutup dan draf di-reset sebelum konfigurasi tool selesai. Konfigurasi tool berjalan lewat promise yang tidak ditunggu. Fungsi konfigurasi memberi warning bila gagal, tetapi perubahan beberapa modul/data awal tidak menjadi satu transaksi dan draf input sudah hilang.

Bukti: [AgentsView.jsx:1713](<C:/Users/Aris Fadillah/Downloads/oneflow-frontend/components/dashboard/views/AgentsView.jsx:1713>), [DashboardApp.jsx:1336](<C:/Users/Aris Fadillah/Downloads/oneflow-frontend/components/dashboard/DashboardApp.jsx:1336>).

**Dampak:** pengguna pemula melihat agent dibuat tetapi harus menyelesaikan alat secara manual; retry dapat mengulang sebagian operasi dan pengguna harus mengisi ulang detail.

**Perbaikan:** simpan status setup dan draf, tampilkan langkah yang berhasil/gagal, serta retry hanya bagian yang belum selesai. Hindari label selesai sebelum seluruh langkah wajib tersimpan.

**Syarat lulus:** putuskan koneksi pada setiap langkah simpan; pengguna dapat melanjutkan tanpa agent/produk duplikat dan tanpa kehilangan data yang diisi.

### 10. [P2, UI + source] CTA demo menuju harga, bukan pengalaman demo

Tombol "Coba Demo" menuju `/pricing`, dan route tersebut redirect ke `/#pricing`. Ini dikonfirmasi di browser production. Pengguna tidak memperoleh demo yang dijanjikan tombol.

Bukti: [pricing/page.js:4](<C:/Users/Aris Fadillah/Downloads/oneflow-frontend/app/(oneflow)/pricing/page.js:4>).

**Perbaikan:** sambungkan ke demo/sandbox nyata atau beri label sesuai tujuan. Animasi hero tidak perlu diubah untuk memperbaiki navigasi ini.

**Syarat lulus:** CTA membawa pengguna ke pengalaman yang sesuai label dan langkah selanjutnya jelas.

## Cakupan Fungsional dan Gate yang Masih Terbuka

| Area | Bukti audit saat ini | Yang wajib dibuktikan sebelum rilis |
| --- | --- | --- |
| Landing, login, daftar | UI publik dapat dibuka; daftar menampilkan form | CTA demo, recovery, validasi registrasi, perjalanan hingga activation |
| Onboarding agent | Source alur percakapan dan penyimpanan diperiksa | Create/edit/retry/reload; izin agent A/B; pengguna pemula menyelesaikan tanpa bantuan |
| Knowledge | Worker hydration/chunking ada di source | PDF/DOCX valid/rusak, status gagal, edit/hapus/reindex, isolasi knowledge tiap agent/tenant |
| AI dan playground | 41 test persona/fallback offline lulus | Kualitas model default aktual, 3 bisnis, fakta salah, mixed-scope, model timeout, hitungan credit |
| WhatsApp | Gateway healthy; endpoint internal terlindungi | Pair/reconnect, inbound, retry webhook, delivery failure, media, sesi tenant lain |
| Inbox dan handoff | API auth gate lulus | Dua admin membuka chat sama, AI berhenti saat takeover, realtime reconnect, pesan tidak ganda |
| CRM, deals, tickets | CRUD/source dan tugas follow-up ditemukan | Stage custom, owner, izin role, status SLA, query/filter/pagination, job otomatis |
| Commerce | Executor draft dan idempotency tersedia di source | Stok terakhir dipesan bersamaan, cancel/confirm berulang, harga sumber resmi, recovery crash |
| Booking | Race dan validasi schedule ditemukan | Konflik paralel, buffer, jam buka, timezone, reschedule |
| Billing | Source webhook memakai signature dan row lock | Sandbox payment sampai credit, callback duplikat/terlambat, cancel/expiry, upgrade, invoice, refund |
| Owner dan tim pelanggan | Test tenant isolation tersedia | Role owner/platform vs org owner vs staff; invite, revoke, logout, account switch; integrasi wajib |
| Operasional | SSH/tunnel/container aktif; disk kritis; alert kosong | Backup/restore, monitoring eksternal, headroom disk, versi image, rollback dan restart drill |

Baris yang belum diuji E2E adalah **gate terbuka**, bukan klaim bug. Pemeriksaan UI authenticated seluruh dashboard, uji usability mobile, pembayaran sandbox, serta database concurrency belum selesai dalam audit ini. Jadi laporan ini bukan sertifikasi bahwa semua fitur sudah dites.

## Hasil Pengujian Aktual

- Frontend: 21 test lulus; split readiness lulus. Ini tidak mencakup browser E2E seluruh dashboard.
- Backend standalone verification: lulus.
- AI persona/fallback: 41 test lulus dengan network disabled di container sementara Ubuntu; tidak menggunakan OpenRouter.
- Smoke domain production: 16 dari 17 assertion lulus. Satu kegagalan `root_not_public` adalah asumsi script backend-only yang tidak cocok dengan domain gabungan frontend/backend; bukan bug landing page.
- Go API dan WA: dicoba dalam container sementara, gagal menyelesaikan pengujian karena disk penuh. Tidak diklaim lulus dan tidak tersedia hasil lengkap untuk dihitung.
- Runtime akhir: `/` 200, `/health` 200, `/api/me` tanpa login 401; container aplikasi utama tetap berjalan. Restart aplikasi tidak dilakukan.
- Artefak audit sementara di Ubuntu sudah dibersihkan. Image/data aplikasi yang sudah ada tidak dihapus.

## Rencana Perbaikan

### Tahap 1: Stabilkan lingkungan dan bukti rilis

Pemilik: backend/ops. Tangani disk dan monitoring terlebih dahulu. Buat database staging terisolasi serta CI integrasi dengan migrations dan test wajib tanpa skip. Jangan kompilasi suite besar lagi pada host production yang sempit.

Selesai bila backup/restore terbukti, alert diterima, kapasitas cukup untuk deploy/rollback, dan test database benar-benar berjalan.

### Tahap 2: Lindungi perubahan bisnis

Pemilik: backend + frontend. Dengan TDD, reproduksi bentrok booking, jam/buffer, perubahan izin agent lain, dan setup parsial. Perbaiki invariant database serta alur retry/draf. Jalankan kasus paralel dan lintas-tenant.

Selesai bila seluruh reproduksi gagal pada versi lama dan lulus pada perbaikan; membuat agent baru tidak mengubah agent lama tanpa tindakan eksplisit.

### Tahap 3: Tuntaskan janji produk

Pemilik: product + backend + frontend. Putuskan dan implementasikan batas pengingat versus pengiriman otomatis, recovery password, CTA demo, status readiness agent, serta langkah pengguna ketika knowledge/WhatsApp/credit belum siap.

Selesai bila pengguna baru mengerti apakah AI sudah siap membalas, mana yang masih draft, dan tindakan apa yang benar-benar otomatis.

### Tahap 4: Uji perjalanan nyata di staging

Gunakan tiga tenant bisnis terpisah: retail, layanan booking, dan B2B. Jalankan perjalanan daftar -> agent -> knowledge -> playground -> channel -> handoff -> aksi bisnis -> billing. Sertakan koneksi putus, duplikasi event, kredit habis, input ambigu, dan pergantian role. Gunakan payment sandbox dan channel tester; pemanggilan model nyata mengikuti izin dan batas biaya yang disepakati.

Untuk pengguna pemula, ukur apakah mereka bisa membuat agent, memperbaiki knowledge, mencoba jawaban, menghubungkan channel, dan mengambil alih chat tanpa bantuan. Catat titik berhenti dan salah paham; jangan menyimpulkan mudah hanya dari tampilan yang rapi.

### Tahap 5: Rilis terbatas lalu perluas

Pilih kandidat commit/image yang pasti. Deploy ke kelompok kecil setelah blocker selesai, pantau error dan hasil bisnis, lalu uji rollback. Rilis umum hanya setelah gate data/biaya/akses/pemulihan lulus dan tidak ada P1 terbuka pada fitur yang dipasarkan.

**Prioritas praktis:** kapasitas + backup/alert -> CI integrasi -> booking + izin tool -> setup dapat dilanjutkan -> follow-up/recovery/demo -> E2E tenant dan pembayaran -> pilot.
