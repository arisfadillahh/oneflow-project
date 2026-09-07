# Prompt GPT Untuk Menjelaskan Project Bisnis Oneflow.id

Copy prompt di bawah ini ke GPT kalau mau minta bantuan strategi, copywriting, pitch deck, roadmap, pricing, analisis kompetitor, atau ide pengembangan produk untuk Oneflow.id.

```text
Gue lagi bangun Oneflow.id, sebuah platform AI customer operations untuk bisnis di Indonesia. Tolong pahami project ini sebagai produk SaaS B2B/B2SMB, lalu bantu gue sesuai request gue setelah konteks ini.

Cara jawab:
- Jawab dalam Bahasa Indonesia yang jelas, praktis, dan cocok buat founder/operator.
- Jangan mengarang fitur di luar konteks ini. Kalau ada asumsi, tulis sebagai asumsi.
- Kalau info bisnis belum cukup, tanya maksimal 5 pertanyaan paling penting.
- Saat bikin output bisnis, fokus ke positioning, value proposition, target customer, monetisasi, operasional, risiko, dan prioritas eksekusi.
- Bedakan fitur yang sudah ada, fitur yang planned, dan ide tambahan.

Ringkasan produk:
Oneflow.id adalah platform multi-tenant untuk mengelola AI customer service dan operasional customer lewat WhatsApp. Tiap bisnis bisa punya organisasi sendiri, AI CS sendiri, knowledge base sendiri, session WhatsApp sendiri, billing/credit sendiri, dan plugin operasional seperti follow-up sales, produk/stok/pesanan, booking, dan payment planned.

Positioning:
Oneflow.id bukan cuma chatbot. Oneflow.id adalah dashboard operasional customer yang menggabungkan AI CS WhatsApp, inbox, handoff manusia, knowledge bisnis, CRM ringan, commerce/order, booking, billing, analytics, dan kontrol tim dalam satu tempat.

Target customer awal:
- UMKM, toko online, retail, jasa appointment, klinik administratif, kursus, properti, travel/hospitality, dan B2B sales/SaaS.
- Bisnis yang banyak menerima chat WhatsApp, sering telat follow-up, data customer tercecer, admin kewalahan, atau butuh AI CS yang tetap bisa dikontrol manusia.
- Tim kecil sampai menengah yang butuh dashboard operasional tanpa harus membangun CRM/CS stack sendiri.

Core value proposition:
- Balasan customer lebih cepat lewat AI CS WhatsApp.
- Admin tetap punya kontrol melalui inbox, handoff, dan notifikasi.
- Knowledge bisnis menjadi sumber jawaban AI.
- Data operasional seperti produk, stok, order, slot booking, dan prospek tidak disimpan di prompt bebas, tapi di plugin bisnis yang terstruktur.
- Billing berbasis paket, limit, credit, dan usage log supaya biaya AI bisa dikontrol.
- Multi-tenant dan role-based access supaya tiap bisnis terpisah dan aman.

Fitur utama:

1. AI CS dan agent configuration
- Bisnis bisa membuat beberapa AI agent/customer service.
- Agent punya konfigurasi persona, model, instruksi, fallback, guardrail, source of truth, escalation rule, dan template notifikasi.
- Agent bisa dikaitkan ke session WhatsApp tertentu.
- Template agent tersedia:
  1. CS Utama
  2. Toko Online & Retail
  3. Jasa Appointment
  4. Klinik & Healthcare Admin
  5. Kursus & Edukasi
  6. Properti & Real Estate
  7. Travel & Hospitality
  8. B2B Sales & SaaS
  9. Custom Kosong
- Model default:
  - Basic: openai/gpt-4.1-mini
  - Advance: anthropic/claude-haiku-4.5
- Owner bisa mengatur model availability, alias model, dan formula kredit/model pricing tanpa restart service.
- AI service memakai OpenRouter untuk model calls.
- Persona bisnis harus berasal dari konfigurasi dashboard, bukan hardcoded di runtime AI.

2. WhatsApp operations
- WhatsApp adalah channel utama kerja.
- Support mock atau real WhatsApp gateway.
- Fitur session lifecycle: connect, QR, QR image, pair phone, status, disconnect, send text, send media, dan chat presence.
- Dashboard bisa membuat session WhatsApp dengan label perangkat dan agent yang dipilih.
- Session WhatsApp bisa soft-delete/rename/disconnect.
- Inbound WhatsApp masuk ke backend, lalu diteruskan ke AI service untuk keputusan balasan.
- Outbound reply dikirim melalui gateway setelah backend menyimpan conversation, message, AI run, dan credit usage.
- Ada grup notifikasi/escalation group untuk menerima notifikasi AI CS dan plugin bisnis.
- Admin bisa mengatur notification rules per plugin seperti AI CS, Produk & Pesanan, dan Booking.

3. Conversation inbox dan human handoff
- Inbox untuk memantau percakapan customer.
- Admin bisa melihat chat, status eskalasi, histori, kontak, dan metadata.
- Mendukung AI-to-human handoff, takeover manusia, manual message, dan return-to-AI.
- Ada conversation events, manual message event, dan audit log.
- Owner/platform view menjaga privasi dengan fokus ke agregat, billing, dan health tanpa membuka isi chat customer.

4. Knowledge base
- Knowledge scoped per organisasi dan bisa juga per AI agent.
- Jenis knowledge:
  - FAQ
  - Dokumen
  - Knowledge chunks/embedding
  - Structured records
  - Job positions atau info produk/layanan lama untuk bisnis tanpa plugin
- Dokumen bisa upload, download, publish/draft, diproses worker untuk chunking dan embedding.
- AI retrieval menggabungkan vector retrieval, lexical scoring, source priority, intent analysis, source judgment, reranking, dan final answer verification.
- Kalau embedding OpenRouter disabled, sistem fallback ke local hash embeddings.
- Knowledge templates:
  - Jam operasional
  - Harga/paket
  - Cara daftar/order
  - Profil bisnis dan cara kerja
  - Kebijakan layanan
  - Info produk/layanan
- Untuk data real-time seperti stok, harga aktif, order, slot, jadwal, dan booking, source of truth harus plugin bisnis, bukan FAQ bebas.

5. AI Playground
- Playground dipakai admin untuk mengetes AI sebagai customer tester.
- Jalur playground memakai backend-to-AI billing/logging yang sama, tapi tidak membuat chat WhatsApp dan tidak mengirim pesan ke WhatsApp.
- Mendukung shortcut test, expected behavior, run history, retrieval metadata, dan debug metadata.
- Cocok untuk validasi sebelum agent diaktifkan ke customer.

6. Business tools/plugin system
- Tiap organisasi bisa install alat bisnis tambahan.
- Plugin yang ready:
  - Prospek & Follow-up
  - Produk, Stok & Pesanan
  - Booking / Jadwal
- Plugin planned:
  - Payment
- Setiap plugin punya Mode AI:
  - off: AI tidak memakai data plugin.
  - read: AI hanya membaca data.
  - draft: AI boleh membuat draft, admin review.
  - action: AI boleh melakukan aksi yang diizinkan saat data sudah lengkap.
- Prinsip keamanan AI tools:
  - AI hanya melakukan structured extraction dari chat.
  - Backend/tool executor melakukan validasi deterministik terhadap database.
  - AI tidak boleh mengarang ID produk, harga, stok, slot booking, refund, status lunas, atau success state.
  - State change berisiko butuh konfirmasi atau validasi yang jelas.
  - Tool result metadata disimpan di retrieval/debug metadata.

7. Prospek & Follow-up
- Mengubah chat menjadi prospek/deal.
- Kelola pipeline prospek, stage, owner, prioritas, due date, dan aktivitas closing.
- Mendukung deal pipelines, deal stages, deal activities, follow-up tasks, contact tags, message templates, dan broadcast.
- AI intents untuk plugin prospects:
  - create_prospect
- Mode AI:
  - read: baca konteks prospek untuk follow-up.
  - draft/action: membuat prospek baru dari chat customer sesuai izin.

8. Produk, Stok & Pesanan
- Kelola katalog produk aktif per bisnis.
- Data produk: nama, SKU, deskripsi, harga, status, stok tersedia, dan stok reserved.
- Kelola draft pesanan dari produk aktif.
- Confirm pesanan untuk reserve stok.
- Cancel/update pesanan sesuai status.
- Kelola order pipeline, order stage, fulfillment type, alamat, recipient, notes, dan status.
- Draft pesanan belum mengunci stok; stok reserved hanya saat admin confirm.
- AI intents untuk commerce:
  - list_products
  - check_stock
  - create_order
  - update_order
  - confirm_order
  - cancel_order
- AI tidak boleh mengubah harga atau stok. AI bisa membaca produk/stok, membuat draft order, atau membuat draft dari chat sesuai Mode AI.

9. Booking / Jadwal
- Kelola layanan booking, durasi, buffer, harga, timezone, hari/jam tersedia, status layanan, dan deskripsi.
- Kelola appointment customer: nama, nomor WhatsApp, service, jadwal, location type, alamat, notes, dan status.
- Sistem menolak appointment yang bentrok pada layanan yang sama.
- AI intents untuk booking:
  - list_booking_services
  - create_booking
  - reschedule_booking
  - confirm_booking
  - cancel_booking
- Mode draft lebih aman untuk awal operasional. Mode action boleh menjadwalkan saat layanan, nama customer, nomor WhatsApp, tanggal, dan jam sudah jelas.

10. Payment dan billing produk
- Payment sebagai business plugin masih planned untuk provider payment per bisnis, payment link setelah order matang, webhook sukses bayar, dan instruksi bayar.
- AI tidak boleh refund, mengubah nominal, atau menandai lunas tanpa bukti payment provider.
- Untuk billing Oneflow.id sendiri, sistem sudah punya paket, wallet, credit, purchase, payment flow, dan Midtrans notification.
- Metode pembayaran dashboard:
  - Virtual Account, rekomendasi, fee + Rp4.440
  - QRIS, fee + 0,7%
  - Kartu, belum tersedia untuk checkout saat ini
- Ada payment status, purchase history, request pembelian, approval/verification, dan cancel purchase.

11. Paket, credit, dan monetisasi
- Model bisnis SaaS dengan paket bulanan/tahunan, credit usage, dan top-up/add-on.
- Pricing public page:
  - Starter: 1.000 credit/bulan, 1 sesi WhatsApp, 2 AI agent, 3 human user, fitur Inbox WhatsApp, Contact 360, Pipeline deal, Reminder follow-up.
  - Growth: 3.000 credit/bulan, 2 sesi WhatsApp, 5 AI agent, 10 human user, semua Starter plus Broadcast segment, Dashboard CRM, Multi-agent workflow.
  - Business: 7.000 credit/bulan, 5 sesi WhatsApp, 15 AI agent, 25 human user, semua Growth plus lebih banyak sesi, tim lebih besar, workflow operasional penuh.
- Pricing page memakai toggle bulanan/tahunan dan annual discount 10%.
- Sistem billing mendukung:
  - Wallet
  - Monthly credit limit
  - Monthly used/remaining
  - Additional/top-up credit
  - Usage logs
  - Billing analytics
  - Credit adjustments
  - Plan limits
  - Trial credits
  - Monthly reset checks
  - Package management oleh owner

12. Team, organization, dan access control
- Multi-tenant organization.
- User bisa login/register, punya organisasi aktif, dan bisa switch organization.
- Role akses utama:
  - owner
  - super_admin
  - admin
  - operator
- Fitur team management:
  - Tambah anggota
  - Invite anggota
  - Accept invite
  - Reset password
  - Nonaktifkan anggota
  - Role-based access
- Role mempengaruhi nav dan izin, termasuk siapa yang boleh melihat health, pricing, billing control, business tools, dan team management.

13. Analytics, dashboard, dan health
- Dashboard summary untuk ringkasan operasional.
- Performance dashboard untuk statistik performa AI, human agent, volume chat, dan escalation.
- Analytics overview untuk volume percakapan, eskalasi, penyelesaian, dan performa operasional sesuai role.
- User analytics untuk agregat pemakaian AI lintas user/bisnis dengan privasi chat.
- Health panel untuk memantau wa-gateway, worker, app-backend, system alerts, dan status operasional.
- Ada notifications, support report, dan issue report via WhatsApp support.

14. Account dan legal surfaces
- Account settings untuk nama akun dan password.
- Public pages: homepage, pricing, login, contact/kontak, about, terms/syarat-ketentuan, privacy/kebijakan-privasi, refund/refund-policy, WhatsApp page, escalation page.

15. Arsitektur teknis
- oneflow-frontend: repository frontend terpisah untuk public site, login/register, owner/admin dashboard, agent setup, knowledge management, pricing/model settings, playground, billing UI, dan business-tool UI.
- apps/app-backend: Go API untuk auth/RBAC, tenant isolation, migrations, dashboard APIs, WhatsApp inbound ownership, AI playground, AI run logging, billing/credit deduction, dan business-tool CRUD.
- apps/ai-service: FastAPI AI runtime untuk decisioning, OpenRouter calls, knowledge retrieval/reranking, answer generation, tool-intent extraction, dan AI-side guardrails.
- apps/wa-gateway: Go WhatsApp gateway untuk mock/real transport, QR/session lifecycle, inbound forwarding ke backend, outbound send-text/media boundary, dan presence.
- apps/worker: Python worker untuk scheduled jobs, knowledge document hydration/chunking, embedding sync, dan wallet reset checks.
- db/migrations: SQL source of truth untuk schema, defaults, plan/model/billing/business-tool tables.

Runtime flow:
1. User/admin bekerja di frontend Oneflow.
2. Frontend memanggil app-backend.
3. Backend enforce auth, organization membership, role permissions, dan plan limits.
4. WhatsApp inbound masuk ke wa-gateway, lalu backend route internal WA inbound.
5. Backend menyimpan conversation/message/event lalu memanggil ai-service /api/decide.
6. AI service mengambil organization/agent settings, selected model, memory, knowledge, dan business-tool state.
7. AI service mengembalikan DecisionResponse: answer, escalate, tool response, metadata, usage.
8. Backend menyimpan ai_runs, cost steps, credit usage logs, wallet changes, lalu mengirim outbound WhatsApp reply jika applicable.

Hal penting yang harus dijaga:
- Tenant isolation wajib: semua data organisasi harus scoped by organization_id dan jika relevan ai_agent_id.
- Billing safety wajib: AI calls, playground, dan WhatsApp AI reply harus tercatat usage/cost/credit.
- Prompt/persona bisnis harus dikontrol dari dashboard per agent/organization.
- Jangan hardcode bahasa, negara, salutation, channel, atau persona bisnis di AI runtime.
- Natural-language business-tool actions harus structured extraction plus deterministic validation.
- AI tidak boleh menjadi sumber kebenaran untuk stok, harga, booking slot, order status, payment status, refund, atau success state.

Saat gue minta bantuan, pakai konteks Oneflow.id ini. Kalau gue minta pitch, landing page, deck, business plan, roadmap, GTM, pricing, UX copy, investor narrative, kompetitor, atau strategi fitur, hasilkan output yang spesifik untuk Oneflow.id, bukan jawaban generik.
```

