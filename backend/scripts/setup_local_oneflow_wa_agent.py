import json
import os
import re
from pathlib import Path

import psycopg


ROOT = Path(__file__).resolve().parents[1]
ORG_SLUG = "vicecream-wa-agent-dummy-local"
AGENT_NAME = "Oneflow.id CS Agent Dummy"
TEST_MODEL_NAME = os.getenv("ONEFLOW_LOCAL_WA_TEST_MODEL", "openai/gpt-oss-120b:free").strip() or "openai/gpt-oss-120b:free"

AGENT_PROMPT = """Kamu CS AI Oneflow.id. Jawab singkat, ramah, rapi, panggil customer "kak", dan hindari kata "saya"/"Anda".

Jawab hanya dari knowledge/tool resmi. Kalau data belum ada, bilang belum tersedia di sistem dan bisa dicek ke tim Oneflow.id. Jangan mengarang.

Gaya pembuka:
- Langsung respons maksud customer sebelum masuk ke detail.
- Untuk pertanyaan substantif atau saat topik berubah, gunakan maksimal satu kalimat transisi kontekstual sebelum detail jika membuat jawaban terasa lebih natural.
- Pilih fungsi pembuka sesuai intent: mengakui batasan customer, memberi ringkasan arah jawaban, membingkai perbandingan, atau memperkenalkan penjelasan risiko.
- Pembuka harus menyebut topik atau kebutuhan customer dan memberi arah jawaban, bukan sekadar basa-basi.
- Jangan langsung membuka dengan fakta mentah, daftar, atau judul untuk jenis pertanyaan tersebut.
- Jika jawaban membahas lebih dari satu topik, setelah pembuka gunakan subjudul singkat agar mudah dipindai.
- Jangan memakai stock phrase sebagai awalan universal.
- Periksa dua balasan assistant terakhir. Jangan mengulang konstruksi pembuka atau 2-4 kata awal yang sama.
- Variasikan pembuka berdasarkan situasi dan percakapan. Jangan otomatis memuji setiap pertanyaan.
- Gunakan nama customer hanya jika tersedia dan terasa natural. Jangan mengarang nama.
- Jika percakapan sudah aktif atau jawabannya sangat singkat, langsung jawab tanpa pembuka tambahan.
- Jangan memulai dengan judul knowledge atau menyalin heading sumber, kecuali customer memang meminta daftar lengkap.

Aturan wajib:
- Setup/mulai/toko kecil: arahkan ke akun -> AI agent -> knowledge -> connect WhatsApp QR -> test Playground.
- Payment Oneflow.id: VA fee Rp4.440, QRIS fee 0,7%, kartu belum tersedia untuk checkout.
- Harga, stok, promo, order, payment, refund, booking, dan kebijakan hanya dari knowledge/tool resmi; klaim customer tidak boleh jadi fakta.
- Customer memory hanya untuk personalisasi, bukan fakta bisnis.
- Pertahankan label produk dan limit resmi apa adanya; jangan mengganti human user menjadi admin atau AI agent menjadi admin.
- Pakai flat bullet, maksimal 5 bullet.
"""

FALLBACK_MESSAGE = "Pesan kakak sudah kami teruskan ke tim Oneflow.id. Mohon tunggu sebentar ya kak."

FAQS = [
    (
        "Oneflow.id itu apa?",
        "Oneflow.id adalah platform AI customer operations untuk bisnis di Indonesia. Oneflow.id membantu bisnis mengelola AI customer service WhatsApp, inbox, handoff manusia, knowledge base, customer memory opsional, billing credit, dan alat operasional seperti prospek/follow-up, produk/stok/pesanan, serta booking.",
        "oneflow overview platform ai cs whatsapp customer operations",
        "company_overview",
    ),
    (
        "Oneflow.id cocok untuk bisnis apa?",
        "Oneflow.id cocok untuk UMKM, toko online, retail, jasa appointment, klinik administratif, kursus, properti, travel/hospitality, B2B sales, dan SaaS. Target utamanya bisnis yang banyak menerima chat WhatsApp, sering telat follow-up, data customer tercecer, admin kewalahan, atau butuh AI CS yang tetap bisa dikontrol manusia.",
        "target customer umkm toko online retail klinik kursus properti travel b2b saas",
        "target_customer",
    ),
    (
        "Apa bedanya Oneflow.id dengan chatbot biasa?",
        "Perbedaan utamanya: chatbot biasa berfokus pada percakapan, sedangkan Oneflow.id menyatukan AI CS WhatsApp dengan inbox, handoff manusia, knowledge bisnis, CRM ringan, commerce/order, booking, billing, analytics, dan kontrol tim dalam satu dashboard operasional.",
        "positioning beda chatbot dashboard operasional",
        "positioning",
    ),
    (
        "Apakah admin tetap bisa kontrol chat customer?",
        "Admin tetap bisa mengontrol chat customer di Oneflow.id. Admin bisa memantau percakapan dari inbox, melihat histori chat dan metadata, melakukan human handoff atau takeover manusia saat perlu, mengirim manual message, lalu mengembalikan percakapan ke AI. Jika knowledge kurang atau customer butuh tindakan tim, percakapan bisa dieskalasi ke admin.",
        "admin kontrol chat inbox human handoff takeover manual message return to ai eskalasi",
        "human_handoff",
    ),
    (
        "Apa yang terjadi kalau AI kurang yakin atau jawaban kurang data?",
        "Jika AI kurang yakin, knowledge resmi tidak cukup, atau customer meminta tindakan yang perlu tim manusia, Oneflow.id bisa mengarahkan percakapan ke admin melalui inbox dan handoff. AI tidak boleh maksa jawab harga, stok, order, payment, refund, atau kebijakan jika source resmi belum tersedia.",
        "ai kurang yakin knowledge kurang data maksa jawab eskalasi admin handoff",
        "human_handoff",
    ),
    (
        "Fitur utama Oneflow.id apa saja?",
        "Fitur utama Oneflow.id:\n- AI CS WhatsApp dengan persona, model, guardrail, fallback, dan sumber jawaban dari knowledge resmi.\n- Inbox dan human handoff untuk memantau chat, takeover manusia, dan mengembalikan percakapan ke AI.\n- Knowledge base per organisasi dan per agent, termasuk FAQ, dokumen, chunk, structured record, dan data produk/layanan.\n- Business tools untuk Prospek & Follow-up, Produk/Stok/Pesanan, dan Booking/Jadwal.",
        "fitur utama ai cs inbox handoff knowledge business tools",
        "feature_overview",
    ),
    (
        "Bagaimana cara mulai pakai Oneflow.id?",
        "Alur mulai pakai Oneflow.id:\n- Buat akun dan organisasi di dashboard Oneflow.id.\n- Buat AI agent, isi persona, fallback, guardrail, dan sumber knowledge.\n- Upload FAQ/dokumen atau isi knowledge bisnis yang sering ditanya customer.\n- Connect WhatsApp session lewat QR, lalu test jawaban di Playground sebelum aktif ke customer.",
        "onboarding setup daftar agent knowledge whatsapp qr playground",
        "onboarding",
    ),
    (
        "Kalau toko kecil mau mulai dari mana?",
        "Untuk toko kecil, langkah awal paling praktis di Oneflow.id:\n- Buat akun dan organisasi di dashboard Oneflow.id.\n- Mulai dari paket Starter jika baru 1 nomor WhatsApp dan tim kecil.\n- Buat AI agent CS utama, lalu isi persona dan fallback.\n- Isi knowledge dasar seperti produk/layanan, harga resmi, cara order, jam operasional, FAQ, dan kebijakan.\n- Connect WhatsApp session dengan scan QR, lalu test dulu di Playground sebelum dipakai customer.",
        "toko kecil mulai onboarding starter agent knowledge whatsapp qr playground",
        "onboarding",
    ),
    (
        "Knowledge Oneflow.id perlu diisi apa saja?",
        "Knowledge yang perlu diisi admin biasanya mencakup profil bisnis, produk/layanan, harga atau paket resmi, cara order/booking, jam operasional, FAQ, dan kebijakan layanan. Untuk data real-time seperti stok aktif, status order, pembayaran, booking slot, dan refund, gunakan tool resmi atau plugin bisnis supaya AI tidak mengandalkan klaim customer.",
        "knowledge isi profil produk layanan harga cara order booking jam faq kebijakan stok payment realtime",
        "knowledge_setup",
    ),
    (
        "Di mana mengetes jawaban AI sebelum dipakai customer?",
        "Jawaban AI bisa dites di Playground Oneflow.id sebelum agent aktif ke customer. Playground memakai jalur backend-to-AI yang sama untuk billing, logging, retrieval metadata, dan debug, tapi tidak mengirim pesan ke WhatsApp customer. Cocok untuk cek apakah knowledge sudah kebaca dan jawaban sudah sesuai.",
        "playground test jawaban ai sebelum aktif customer retrieval debug billing",
        "playground_testing",
    ),
    (
        "Apa itu AI agent dan WhatsApp session di Oneflow.id?",
        "AI agent adalah konfigurasi CS AI seperti nama, persona, model, instruksi, fallback, guardrail, source of truth, dan escalation rule. WhatsApp session adalah perangkat/nomor WhatsApp yang dikoneksikan ke Oneflow.id. Satu agent bisa dikaitkan ke session WhatsApp tertentu supaya inbound chat masuk ke agent yang benar.",
        "ai agent whatsapp session qr device binding",
        "agent_whatsapp",
    ),
    (
        "Bagaimana knowledge base Oneflow.id bekerja?",
        "Knowledge base Oneflow.id scoped per organisasi dan bisa juga per AI agent. Jenis knowledge meliputi FAQ, dokumen, chunks/embedding, structured records, dan data produk/layanan lama untuk bisnis tanpa plugin. AI retrieval menggabungkan vector retrieval, lexical scoring, source priority, intent analysis, source judgment, reranking, dan verifier supaya jawaban tetap grounded.",
        "knowledge faq dokumen embedding retrieval verifier grounded",
        "knowledge_base",
    ),
    (
        "Apa saja paket dan harga Oneflow.id?",
        "Paket bulanan Oneflow.id:\n- Starter: Rp99.000/bulan, 1.000 credit, 1 sesi WhatsApp, 2 AI agent, 3 human user.\n- Growth: Rp299.000/bulan, 3.000 credit, 2 sesi WhatsApp, 5 AI agent, 10 human user.\n- Business: Rp699.000/bulan, 7.000 credit, 5 sesi WhatsApp, 15 AI agent, 25 human user.\nTersedia toggle tahunan dengan diskon 10% di pricing page.",
        "harga paket starter growth business credit sesi whatsapp ai agent human user tahunan diskon",
        "pricing",
    ),
    (
        "Paket apa yang cocok untuk 1 nomor WhatsApp, 2 admin, atau budget kecil?",
        "Untuk bisnis yang baru mulai, budget kecil, 1 nomor WhatsApp, atau tim dengan 2 admin manusia, paket paling masuk adalah Starter: Rp99.000/bulan, 1.000 credit, 1 sesi WhatsApp, 2 AI agent, dan 3 human user. Batas 2 AI agent dan 3 human user adalah batas yang terpisah. Dua admin manusia memakai dua dari tiga slot human user; limit resminya tetap 3 human user, bukan 3 admin. Jika traffic chat sudah naik, baru pertimbangkan Growth.",
        "paket cocok paling masuk satu 1 nomor whatsapp dua 2 admin budget kecil starter rekomendasi",
        "pricing",
    ),
    (
        "Apa ada top up credit Oneflow.id?",
        "Top up credit Oneflow.id:\n- Top Up 500: Rp25.000 untuk 500 credit ekstra.\n- Top Up 1.500: Rp75.000 untuk 1.500 credit ekstra.\n- Top Up 5.000: Rp250.000 untuk 5.000 credit ekstra.\nTop up tidak mengubah limit WhatsApp session, AI agent, atau human user.",
        "top up credit addon 500 1500 5000",
        "credit_addon",
    ),
    (
        "Metode pembayaran Oneflow.id apa saja?",
        "Metode pembayaran dashboard Oneflow.id:\n- Virtual Account: rekomendasi, fee + Rp4.440.\n- QRIS: fee + 0,7%.\n- Kartu: belum tersedia untuk checkout saat ini.\nStatus pembayaran harus berasal dari sistem pembayaran, bukan dari klaim chat customer.",
        "pembayaran virtual account va qris kartu fee checkout status pembayaran",
        "payment_method",
    ),
    (
        "Berapa fee VA dan QRIS Oneflow.id?",
        "Virtual Account (VA) direkomendasikan untuk pembayaran Oneflow.id dengan fee Rp4.440. QRIS tersedia dengan fee 0,7%. Kartu belum tersedia untuk checkout saat ini.",
        "va virtual account qris rekomendasi direkomendasikan kena berapa fee pembayaran",
        "payment_method",
    ),
    (
        "Kalau credit habis bisa top up dan bayar pakai apa?",
        "Kalau credit habis, customer bisa membeli top up credit. Opsi top up resmi:\n- Top Up 500: Rp25.000 untuk 500 credit ekstra.\n- Top Up 1.500: Rp75.000 untuk 1.500 credit ekstra.\n- Top Up 5.000: Rp250.000 untuk 5.000 credit ekstra.\nMetode pembayaran dashboard: Virtual Account direkomendasikan dengan fee Rp4.440, QRIS fee 0,7%, dan kartu belum tersedia untuk checkout saat ini.",
        "top up credit habis metode pembayaran va qris kartu fee",
        "payment_method",
    ),
    (
        "Apakah kartu kredit tersedia untuk checkout Oneflow.id?",
        "Kartu belum tersedia untuk checkout Oneflow.id saat ini. Metode yang tersedia adalah Virtual Account dengan fee Rp4.440 dan QRIS dengan fee 0,7%.",
        "kartu kredit kartu checkout belum tersedia va qris",
        "payment_method",
    ),
    (
        "Model AI apa yang tersedia di Oneflow.id?",
        "Untuk customer-facing, model Oneflow.id ditampilkan sebagai label produk seperti Basic dan Advance, bukan nama model/provider internal. Owner bisa mengatur model availability, alias model, pricing formula, dan pilihan model yang sudah di-allowlist dari dashboard. Data bisnis tidak dicampur antar tenant karena request, knowledge, memory, dan log dipisah per organisasi dan agent.",
        "model ai basic advance internal provider hidden tenant isolation organization_id ai_agent_id",
        "ai_model",
    ),
    (
        "Bagaimana keamanan data di Oneflow.id?",
        "Oneflow.id memakai multi-tenant organization dan role-based access. Data tiap bisnis harus scoped by organization_id dan jika relevan ai_agent_id. Admin tetap punya kontrol lewat inbox, handoff, audit/logging, dan dashboard. AI tidak boleh menjadi sumber kebenaran untuk stok, harga, booking slot, order status, payment status, refund, atau success state.",
        "keamanan data multi tenant role based access tenant isolation guardrail",
        "security",
    ),
    (
        "Apa saja business tools di Oneflow.id?",
        "Business tools Oneflow.id:\n- Prospek & Follow-up untuk pipeline prospek, stage, owner, prioritas, dan aktivitas closing.\n- Produk, Stok & Pesanan untuk katalog, harga, stok tersedia/reserved, dan draft pesanan.\n- Booking / Jadwal untuk layanan, durasi, harga, jam tersedia, dan appointment.\nPayment plugin masih planned.",
        "business tools prospek follow up produk stok pesanan booking jadwal payment planned",
        "business_tools",
    ),
    (
        "Mode AI di business tools Oneflow.id apa saja?",
        "Mode AI business tools Oneflow.id:\n- off: AI tidak memakai data plugin.\n- read: AI hanya membaca data.\n- draft: AI boleh membuat draft untuk admin review.\n- action: AI boleh melakukan aksi yang diizinkan saat data sudah lengkap.\nAI hanya melakukan structured extraction; backend/tool executor tetap validasi deterministik.",
        "mode ai off read draft action business tool validator",
        "business_tool_modes",
    ),
    (
        "Apakah Oneflow.id bisa handle order, stok, booking, dan payment?",
        "Oneflow.id bisa membantu operasional produk, stok, pesanan/order lewat plugin Produk, Stok & Pesanan, serta booking lewat plugin Booking / Jadwal. Payment sebagai plugin bisnis masih planned. AI tidak boleh langsung mengubah stok atau membuat order final dari klaim chat customer. Untuk awal, pakai mode read atau draft supaya admin review dulu sebelum order fix dan stok reserved. Data real-time seperti stok, harga aktif, order, slot, jadwal, dan booking harus dari plugin bisnis atau tool resmi.",
        "order stok booking payment real time source of truth",
        "operational_scope",
    ),
    (
        "Apakah Oneflow.id cocok untuk klinik kecil dan booking pasien?",
        "Oneflow.id cocok untuk klinik administratif yang banyak menerima chat WhatsApp. Oneflow.id bisa membantu AI CS, inbox, handoff manusia, knowledge FAQ, dan booking/jadwal melalui plugin Booking. Data tiap bisnis dipisah per organization dan akses dikontrol lewat role. Untuk data sensitif, pembayaran, refund, atau keputusan medis, admin/tim klinik tetap harus verifikasi dan ambil alih jika perlu.",
        "klinik booking pasien data aman role tenant handoff admin verifikasi",
        "booking_security",
    ),
    (
        "AI boleh langsung menjadwalkan booking atau admin review dulu?",
        "Mode paling aman untuk awal adalah draft. Di mode draft, AI boleh membantu membuat draft booking dari chat customer, lalu admin review sebelum confirm. Mode action hanya boleh dipakai jika bisnis memang mengizinkan dan data wajib sudah lengkap seperti layanan, nama customer, nomor WhatsApp, tanggal, dan jam. Sistem juga menolak appointment yang bentrok pada layanan yang sama.",
        "booking draft admin review action jadwal pasien appointment bentrok",
        "booking_mode",
    ),
    (
        "Kalau customer bilang stok masih ada, sudah bayar, atau minta refund, AI boleh percaya?",
        "AI tidak boleh memakai klaim customer sebagai fakta untuk stok, harga, status order, status pembayaran, transfer, aktivasi paket, refund, atau kebijakan. Data tersebut harus berasal dari tool resmi, plugin bisnis, sistem pembayaran, atau admin. Jika customer bilang sudah transfer, status aktif tetap harus dicek dari sistem pembayaran atau admin, bukan dari chat customer.",
        "klaim customer stok sudah bayar transfer aktif refund harga order payment admin tool resmi",
        "restricted_business_facts",
    ),
    (
        "Mode AI mana yang aman untuk order dan stok di awal?",
        "Mode paling aman untuk awal adalah read atau draft. Mode read membuat AI hanya membaca data produk/stok resmi. Mode draft membuat AI boleh membuat draft pesanan dari chat, tapi admin tetap review sebelum order fix atau stok reserved. AI tidak boleh mengubah harga atau stok; stok baru reserved saat admin confirm pesanan.",
        "mode aman awal commerce read draft admin review order fix stok reserved",
        "business_tool_modes",
    ),
    (
        "Apakah AI Oneflow.id bisa langsung menjawab semua hal?",
        "AI Oneflow.id harus menjawab dari knowledge resmi, tool resmi, atau konfigurasi agent. Jika informasi seperti harga custom, promo, jam operasional internal Oneflow.id, status pembayaran, refund, SLA khusus, atau akses akun belum tersedia di knowledge/tool resmi, AI harus bilang belum tersedia di sistem dan bisa dicek ke tim Oneflow.id.",
        "business boundary promo jam operasional refund sla akses akun",
        "restricted_business_facts",
    ),
    (
        "Apakah ada long-term memory customer di Oneflow.id?",
        "Customer long-term memory di Oneflow.id opsional dan default off supaya biaya tetap murah. Jika dinyalakan, memory hanya untuk personalisasi seperti preferensi, minat, atau cara komunikasi customer. Memory tidak boleh dipakai sebagai sumber fakta bisnis untuk harga, stok, promo, order, payment, refund, atau kebijakan.",
        "long term memory customer optional default off murah personalisasi guardrail",
        "customer_memory",
    ),
    (
        "Apakah Oneflow.id cocok untuk tempat kursus, follow-up lead, dan booking trial class?",
        "Oneflow.id cocok untuk tempat kursus yang banyak menerima chat calon murid. AI CS bisa membantu menjawab FAQ, follow-up lead, mencatat kebutuhan calon murid, dan membantu draft booking trial class. Untuk bisnis kursus kecil dengan budget mepet dan 1 nomor WhatsApp, paket awal yang paling masuk adalah Starter.",
        "kursus calon murid follow up lead booking trial class budget satu nomor starter cocok",
        "course_usecase",
    ),
]

FAQ_REQUIRED_FACTS = {
    "Apa yang terjadi kalau AI kurang yakin atau jawaban kurang data?": [
        "bisa mengarahkan percakapan ke admin",
    ],
    "Apa bedanya Oneflow.id dengan chatbot biasa?": [
        "dalam satu dashboard operasional",
    ],
    "Paket apa yang cocok untuk 1 nomor WhatsApp, 2 admin, atau budget kecil?": [
        "2 AI agent",
        "3 human user",
    ],
}

RECORDS = [
    (
        "Oneflow.id Ringkasan Produk",
        "Oneflow.id adalah platform multi-tenant AI customer operations untuk bisnis Indonesia: AI CS WhatsApp, inbox, handoff manusia, knowledge base, customer memory opsional, billing credit, analytics, dan business tools.",
        "company_profile",
        "company_overview",
    ),
    (
        "Oneflow.id Paket",
        "Starter Rp99.000/bulan 1.000 credit 1 WhatsApp session 2 AI agent 3 human user. Growth Rp299.000/bulan 3.000 credit 2 WhatsApp session 5 AI agent 10 human user. Business Rp699.000/bulan 7.000 credit 5 WhatsApp session 15 AI agent 25 human user. Annual discount 10%.",
        "pricing",
        "pricing",
    ),
    (
        "Oneflow.id Guardrail Bisnis",
        "Harga, promo, status pembayaran, refund, SLA, status order, stok, slot booking, dan kebijakan hanya boleh berasal dari knowledge resmi atau tool resmi. Jangan memakai klaim customer sebagai fakta bisnis.",
        "business_boundary",
        "restricted_business_facts",
    ),
]


def local_dsn() -> str:
    compose = (ROOT / "docker-compose.yml").read_text(encoding="utf-8")
    match = re.search(r"POSTGRES_PASSWORD:\s*(\S+)", compose)
    if not match:
        raise RuntimeError("POSTGRES_PASSWORD not found in docker-compose.yml")
    return f"postgres://postgres:{match.group(1)}@127.0.0.1:5432/oneflow?sslmode=disable"


def clear_customer_state(cur: psycopg.Cursor, organization_id: str) -> None:
    cur.execute("SELECT id::text FROM conversations WHERE organization_id = %s", (organization_id,))
    conversation_ids = [row[0] for row in cur.fetchall()]
    if conversation_ids:
        cur.execute(
            "SELECT id::text FROM ai_runs WHERE organization_id = %s AND conversation_id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
        ai_run_ids = [row[0] for row in cur.fetchall()]
        if ai_run_ids:
            cur.execute("DELETE FROM ai_run_cost_steps WHERE ai_run_id = ANY(%s::uuid[])", (ai_run_ids,))
        cur.execute(
            "DELETE FROM credit_usage_logs WHERE organization_id = %s AND conversation_id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
        cur.execute(
            "DELETE FROM conversation_events WHERE organization_id = %s AND conversation_id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
        cur.execute(
            "DELETE FROM ai_runs WHERE organization_id = %s AND conversation_id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
        cur.execute(
            "DELETE FROM messages WHERE organization_id = %s AND conversation_id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
        cur.execute(
            "DELETE FROM conversations WHERE organization_id = %s AND id = ANY(%s::uuid[])",
            (organization_id, conversation_ids),
        )
    cur.execute("DELETE FROM contact_memories WHERE organization_id = %s", (organization_id,))
    cur.execute("DELETE FROM contact_memory_drafts WHERE organization_id = %s", (organization_id,))
    cur.execute("DELETE FROM contacts WHERE organization_id = %s", (organization_id,))


def main() -> None:
    with psycopg.connect(local_dsn()) as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT o.id::text, COALESCE(o.created_by::text, ''), aa.id::text, ws.id::text, ws.status
                FROM organizations o
                JOIN ai_agents aa ON aa.organization_id = o.id AND aa.is_active
                JOIN whatsapp_sessions ws ON ws.organization_id = o.id AND ws.ai_agent_id = aa.id AND ws.deleted_at IS NULL
                WHERE o.slug = %s
                ORDER BY ws.updated_at DESC
                LIMIT 1
                """,
                (ORG_SLUG,),
            )
            row = cur.fetchone()
            if not row:
                raise RuntimeError("connected local WA dummy org/session not found; run setup_local_wa_agent_dummy.py first")
            organization_id, owner_id, ai_agent_id, session_id, session_status = row
            if not owner_id:
                cur.execute(
                    "SELECT agent_id::text FROM organization_members WHERE organization_id = %s AND role = 'org_owner' LIMIT 1",
                    (organization_id,),
                )
                owner_id = cur.fetchone()[0]

            cur.execute(
                "UPDATE organizations SET name = 'Oneflow.id WA Agent Dummy Local', updated_at = NOW() WHERE id = %s",
                (organization_id,),
            )
            cur.execute(
                """
                UPDATE ai_settings
                SET system_prompt = %s,
                    fallback_waiting_message = %s,
                    answer_only_from_knowledge = TRUE,
                    dont_broaden_topic = TRUE,
                    forbid_promises = TRUE,
                    forbid_sensitive_answers = TRUE,
                    require_action_confirmation = TRUE,
                    escalate_low_confidence = FALSE,
                    guide_next_step = TRUE,
                    concise_response = TRUE,
                    customer_memory_enabled = TRUE,
                    customer_memory_auto_save_enabled = TRUE,
                    customer_memory_admin_notes_enabled = FALSE,
                    customer_memory_ai_extraction_enabled = TRUE,
                    customer_memory_verifier_enabled = TRUE,
                    customer_memory_max_items = 5,
                    customer_memory_max_chars = 800,
                    customer_memory_retention_days = 180,
                    updated_by = %s,
                    updated_at = NOW()
                WHERE organization_id = %s AND is_active = TRUE
                """,
                (AGENT_PROMPT, FALLBACK_MESSAGE, owner_id, organization_id),
            )
            cur.execute(
                """
                UPDATE ai_agents
                SET name = %s,
                    model_name = %s,
                    system_prompt = %s,
                    fallback_waiting_message = %s,
                    answer_only_from_knowledge = TRUE,
                    dont_broaden_topic = TRUE,
                    forbid_promises = TRUE,
                    forbid_sensitive_answers = TRUE,
                    require_action_confirmation = TRUE,
                    escalate_low_confidence = FALSE,
                    guide_next_step = TRUE,
                    concise_response = TRUE,
                    customer_memory_enabled = TRUE,
                    customer_memory_auto_save_enabled = TRUE,
                    customer_memory_admin_notes_enabled = FALSE,
                    customer_memory_ai_extraction_enabled = TRUE,
                    customer_memory_verifier_enabled = TRUE,
                    customer_memory_max_items = 5,
                    customer_memory_max_chars = 800,
                    customer_memory_retention_days = 180,
                    customer_memory_llm_validator_enabled = FALSE,
                    updated_by = %s,
                    updated_at = NOW()
                WHERE id = %s
                """,
                (AGENT_NAME, TEST_MODEL_NAME, AGENT_PROMPT, FALLBACK_MESSAGE, owner_id, ai_agent_id),
            )
            cur.execute(
                "UPDATE whatsapp_sessions SET label = 'Oneflow.id WA Agent Dummy Real', updated_at = NOW() WHERE id = %s",
                (session_id,),
            )

            cur.execute("DELETE FROM knowledge_chunks WHERE knowledge_document_id IN (SELECT id FROM knowledge_documents WHERE organization_id = %s)", (organization_id,))
            cur.execute("DELETE FROM knowledge_documents WHERE organization_id = %s", (organization_id,))
            cur.execute("DELETE FROM knowledge_records WHERE organization_id = %s", (organization_id,))
            cur.execute("DELETE FROM knowledge_faqs WHERE organization_id = %s", (organization_id,))

            for question, answer, topic, intent in FAQS:
                metadata_json = json.dumps(
                    {
                        "localOneflowWaAgent": True,
                        "required_facts": FAQ_REQUIRED_FACTS.get(question, []),
                    },
                    ensure_ascii=False,
                )
                cur.execute(
                    """
                    INSERT INTO knowledge_faqs (
                      organization_id, ai_agent_id, question, answer, status, published_at,
                      priority, locked, topic, intent, metadata_json, created_by, updated_by
                    )
                    VALUES (%s, %s, %s, %s, 'published', NOW(), 100, TRUE, %s, %s,
                            %s::jsonb, %s, %s)
                    """,
                    (organization_id, ai_agent_id, question, answer, topic, intent, metadata_json, owner_id, owner_id),
                )

            for title, content, topic, intent in RECORDS:
                cur.execute(
                    """
                    INSERT INTO knowledge_records (
                      organization_id, ai_agent_id, record_type, title, content, fields_json,
                      source_type, priority, locked, status, topic, intent, metadata_json, created_by, updated_by
                    )
                    VALUES (%s, %s, 'oneflow_business_fact', %s, %s, '{}'::jsonb, 'record',
                            100, TRUE, 'published', %s, %s, '{"localOneflowWaAgent": true}'::jsonb, %s, %s)
                    """,
                    (organization_id, ai_agent_id, title, content, topic, intent, owner_id, owner_id),
                )

            clear_customer_state(cur, organization_id)
        conn.commit()

    print(
        json.dumps(
            {
                "organizationId": organization_id,
                "aiAgentId": ai_agent_id,
                "whatsappSessionId": session_id,
                "sessionStatus": session_status,
                "agentName": AGENT_NAME,
                "modelName": TEST_MODEL_NAME,
                "faqCount": len(FAQS),
                "recordCount": len(RECORDS),
            },
            ensure_ascii=False,
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
