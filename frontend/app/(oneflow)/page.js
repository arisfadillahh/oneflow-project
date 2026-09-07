import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { resolveServerApiProxyTarget } from '../../lib/runtime-config.mjs';

const shellTag = '<main class="main">';
const source = readFileSync(join(process.cwd(), 'content', 'oneflow-shell.html'), 'utf8');
const bodyStart = source.indexOf('<body>') + '<body>'.length;
const runtimeStart = source.indexOf('<script src="/_next', bodyStart);
const bodyMarkup = source.slice(bodyStart, runtimeStart);

if (!bodyMarkup.startsWith(shellTag) || !bodyMarkup.endsWith('</main>')) {
  throw new Error('The legacy Oneflow landing snapshot could not be parsed.');
}

const renderComparisonIcon = (name) =>
  readFileSync(
    join(process.cwd(), 'node_modules', 'lucide-static', 'icons', `${name}.svg`),
    'utf8',
  )
    .replace(/<!--[\s\S]*?-->\s*/, '')
    .replace('class="lucide ', 'class="oneflow-library-icon lucide ')
    .replace('stroke-width="2"', 'stroke-width="2.25"')
    .replace('<svg\n', '<svg\n  aria-hidden="true"\n  focusable="false"\n');

const comparisonIcons = {
  alert: renderComparisonIcon('triangle-alert'),
  bot: renderComparisonIcon('bot'),
  check: renderComparisonIcon('check'),
  clipboard: renderComparisonIcon('clipboard-list'),
  clipboardPlus: renderComparisonIcon('clipboard-plus'),
  message: renderComparisonIcon('message-circle'),
  messages: renderComparisonIcon('messages-square'),
  user: renderComparisonIcon('user-round'),
};

const setupIcons = {
  rocket: renderComparisonIcon('rocket'),
  shield: renderComparisonIcon('shield-check'),
};

export const dynamic = 'force-dynamic';

// Preserve the approved Webflow-derived markup while supplying product imagery.
const rawLegacyMarkup = bodyMarkup.slice(shellTag.length, -'</main>'.length);
const heroStart = rawLegacyMarkup.indexOf('<section data-load="display" class="hero-section');
const heroEnd = rawLegacyMarkup.indexOf('</section>', heroStart) + '</section>'.length;

if (heroStart < 0 || heroEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow hero section could not be parsed.');
}

const heroMarkup = rawLegacyMarkup
  .slice(heroStart, heroEnd)
  .replace(
    '<div class="hero-caption-title"><h1 list-item="show" class="hero-title">AI CS</h1><img src="/images/oneflow_icon.png" loading="lazy" list-item="show" alt="Oneflow AI icon" class="hero-icon"/><h1 list-item="show" class="hero-title">WhatsApp</h1></div>',
    '<div class="hero-caption-title oneflow-hero-headline"><h1 list-item="show" class="hero-title oneflow-hero-title"><span class="oneflow-sr-only">Ubah chat jadi closing, penjualan, booking, order masuk, pembelian, dan transaksi</span><span aria-hidden="true" class="oneflow-hero-visible-title"><span class="oneflow-hero-static">Ubah chat jadi</span><span class="oneflow-hero-word-window"><span class="oneflow-hero-word is-first">closing</span><span class="oneflow-hero-word is-second">penjualan</span><span class="oneflow-hero-word is-third">booking</span><span class="oneflow-hero-word is-fourth">order masuk</span><span class="oneflow-hero-word is-fifth">pembelian</span><span class="oneflow-hero-word is-sixth">transaksi</span></span></span></h1></div>',
  )
  .replace(
    '/oneflow_assets/hero-ai-cs-whatsapp.svg',
    '/oneflow_assets/hero-ai-cs-whatsapp.svg?v=inbox-tools-3',
  )
  .replace(
    'Oneflow AI CS WhatsApp dashboard with inbox, sales pipeline, plugin cards, and role access',
    'Oneflow.id AI CS workspace dengan inbox WhatsApp, handoff admin, dan alat bisnis',
  )
  .replaceAll(
    '/oneflow_assets/parallax-foreground.svg',
    '/oneflow_assets/hero-workflow-foreground.svg?v=outcome-copy',
  )
  .replace(/<img[^>]*class="hero-decoration(?: _02| _03)?"\/>/g, '')
  .replace('<div class="hero-decoration-center"></div>', '')
  .replace(
    '<div class="hero-bg-item"><img class="hero-bg-image" src="/oneflow_assets/parallax-backdrop.svg" alt="Oneflow AI CS WhatsApp flow background" loading="lazy" fetchPriority="high"/></div>',
    '<div class="hero-bg-item oneflow-hero-interactive-bg"><canvas class="oneflow-hero-canvas" aria-hidden="true"></canvas></div>',
  )
  .replace(
    'Kelola chat customer, AI CS, dan pipeline sales WhatsApp dalam satu workspace operasional.',
    'Semua percakapan pelanggan di satu tempat — AI yang follow-up otomatis, konfirmasi booking, dan catat order, tanpa kamu perlu pindah-pindah aplikasi.',
  )
  .replace(
    'href="/request-a-demo" class="button-primary',
    'data-oneflow-scroll="setup" href="#how-it-works" class="button-primary',
  )
  .replace(
    '<div class="button-title">Lihat Fitur</div>',
    '<div class="button-title">Lihat setup mudah</div>',
  )
  .replace(
    '<div class="list-title">Template agent bisnis</div>',
    '<div class="list-title">Template agent bisnis</div>',
  )
  .replace(
    '<div class="list-title">Plugin operasional</div>',
    '<div class="list-title">Plugin operasional</div>',
  )
  .replace(
    '<div class="list-title">Role akses akun</div>',
    '<div class="list-title">Role akses akun</div>',
  );

const heroUpdatedMarkup =
  rawLegacyMarkup.slice(0, heroStart) + heroMarkup + rawLegacyMarkup.slice(heroEnd);
const featureStart = heroUpdatedMarkup.indexOf('<section id="feature"');
const featureEnd = heroUpdatedMarkup.indexOf('</section>', featureStart) + '</section>'.length;

if (featureStart < 0 || featureEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow feature section could not be parsed.');
}

const featureMarkup = heroUpdatedMarkup
  .slice(featureStart, featureEnd)
  .replace(
    'AI CS WhatsApp + CRM operasional',
    'AI menjawab, admin menangani yang penting',
  )
  .replace(
    'Mulai dari template agent, plugin bisnis, inbox WhatsApp, sampai akses role untuk tim operasional.',
    'AI menjawab dari informasi bisnis yang kamu setujui. Kasus penting langsung diteruskan ke admin.',
  )
  .replace(/<div class="bento-button">[\s\S]*?<\/a><\/div>/, '')
  .replace('AI CS terarah', 'Inbox + kontrol AI')
  .replace(
    '/oneflow_assets/usecases-indonesia.svg',
    '/oneflow_assets/feature-ai-guidance.svg?v=simple-ui',
  )
  .replace(
    'Template agent Oneflow untuk bisnis Indonesia',
    'Inbox Oneflow dengan jawaban AI dan handoff admin',
  )
  .replace('Plugin bisnis', 'Alur otomatis')
  .replace(
    '/oneflow_assets/plugin-suite.svg',
    '/oneflow_assets/feature-plugin-automation.svg',
  )
  .replace(
    /<div class="feature-item-bottom"><img[^>]+class="feature-bottom-image _01"\/><img[^>]+class="feature-bottom-image _02"\/><\/div>/,
    '<div class="feature-item-bottom"><img src="/oneflow_assets/feature-crm-pipeline.svg?v=simple-ui" loading="lazy" alt="Pipeline CRM dari chat masuk hingga follow-up admin" class="oneflow-crm-feature-image"/></div>',
  )
  .replace(
    '<div class="bento-item-title text-white">AI CS</div><div class="bento-box">',
    '<div class="bento-item-title text-white">Jawaban AI</div><img src="/oneflow_assets/feature-ai-reply.svg" loading="lazy" alt="Jawaban AI berdasarkan knowledge dengan status siap kirim" class="oneflow-ai-reply-image"/><div class="bento-box">',
  )
  .replace(
    'Jawaban otomatis dari knowledge bisnis.',
    'Jawaban disusun dari knowledge, lalu admin mengambil alih bila diperlukan.',
  )
  .replace('>Reminder</div>', '>Handoff + pengingat</div>')
  .replace(
    '<div class="feature-alert-card">',
    '<div class="feature-alert-card"><img src="/oneflow_assets/feature-handoff-reminder.svg?v=simple-ui" loading="lazy" alt="Reminder follow-up untuk handoff admin" class="oneflow-reminder-image"/>',
  );

const featureUpdatedMarkup =
  heroUpdatedMarkup.slice(0, featureStart) +
  featureMarkup +
  heroUpdatedMarkup.slice(featureEnd);
const productStart = featureUpdatedMarkup.indexOf('<section id="product"');
const productEnd =
  featureUpdatedMarkup.indexOf('</section>', productStart) + '</section>'.length;

if (productStart < 0 || productEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow product comparison section could not be parsed.');
}

const productMarkup = `
<section id="product" class="comparison-section oneflow-scroll-comparison">
  <div class="inner-container">
    <div class="oneflow-compare-sticky">
      <div class="oneflow-compare-frame" data-oneflow-compare style="--comparison-progress: 0%;">
        <div class="oneflow-compare-tabs" role="group" aria-label="Pilih tampilan perbandingan">
          <i class="oneflow-compare-tab-indicator"></i>
          <button type="button" class="is-before" data-comparison-view="before">Sebelum <strong>Oneflow</strong></button>
          <button type="button" class="is-after" data-comparison-view="after">Dengan <strong>Oneflow</strong><i class="oneflow-tab-spark" aria-hidden="true"></i></button>
        </div>
        <div class="oneflow-compare-viewport">
          <article class="oneflow-compare-layer is-before">
            <div class="oneflow-compare-copy">
              <p class="oneflow-compare-label">Alur manual</p>
              <h3>Chat rame,<br/>follow-up berantakan</h3>
              <p>Admin harus cek chat satu-satu, cari konteks lama, dan ingat sendiri mana yang harus dilanjutin.</p>
              <ul class="oneflow-before-list">
                <li><i class="is-alert" aria-hidden="true">${comparisonIcons.alert}</i>Lead lupa di-follow-up</li>
                <li><i class="is-note" aria-hidden="true">${comparisonIcons.clipboard}</i>Order &amp; booking masih dicatat manual</li>
                <li><i class="is-person" aria-hidden="true">${comparisonIcons.user}</i>Admin bingung mana yang harus diprioritaskan</li>
              </ul>
            </div>
            <div class="oneflow-before-board" aria-label="Masalah sebelum menggunakan Oneflow">
              <div class="oneflow-before-alert"><i aria-hidden="true">${comparisonIcons.alert}</i>Lambat respon</div>
              <div class="oneflow-before-route" aria-hidden="true">
                <svg viewBox="0 0 500 355" preserveAspectRatio="none">
                  <path class="oneflow-before-route-path" d="M112 48H72C37 48 20 67 20 102C20 133 39 148 75 148H418C460 148 478 165 478 196C478 226 459 240 420 240H78C39 240 20 258 20 285C20 305 35 316 61 316" />
                  <path class="oneflow-before-route-arrow" d="M56 311L62 318L68 311" />
                  <circle cx="441" cy="43" r="7" />
                  <circle cx="27" cy="177" r="7" />
                  <circle cx="450" cy="299" r="7" />
                </svg>
              </div>
              <div class="oneflow-before-row">
                <i class="is-chat" aria-hidden="true">${comparisonIcons.message}</i>
                <div><span>Chat customer</span><strong>Banyak chat belum dibalas</strong></div>
              </div>
              <div class="oneflow-before-row is-warning">
                <i class="is-person" aria-hidden="true">${comparisonIcons.user}</i>
                <div><span>Follow-up</span><strong>Prospek hilang karena lupa</strong></div>
              </div>
              <div class="oneflow-before-row is-warning is-order">
                <i class="is-note" aria-hidden="true">${comparisonIcons.clipboard}</i>
                <div><span>Order &amp; booking</span><strong>Data dicatat ulang manual</strong></div>
              </div>
              <div class="oneflow-before-note">Semua tersebar, admin sulit tahu mana yang harus dikerjakan dulu.</div>
            </div>
          </article>
          <article class="oneflow-compare-layer is-after">
            <div class="oneflow-compare-copy">
              <p class="oneflow-compare-label">Dengan Oneflow</p>
              <h3>Setiap chat langsung<br/>jadi <span class="oneflow-gradient-text">aksi bisnis</span></h3>
              <p>AI menjawab dari knowledge yang disetujui, membuat follow-up, dan menghubungkan order, booking, serta handoff admin dari satu flow.</p>
              <ul class="oneflow-after-list">
                <li><i aria-hidden="true">${comparisonIcons.check}</i>AI menjawab sesuai knowledge</li>
                <li><i aria-hidden="true">${comparisonIcons.check}</i>Owner, due date, dan reminder otomatis</li>
                <li><i aria-hidden="true">${comparisonIcons.check}</i>Order, booking, dan handoff terhubung</li>
                <li><i aria-hidden="true">${comparisonIcons.check}</i>Admin bisa mengambil alih saat dibutuhkan</li>
              </ul>
            </div>
            <div class="oneflow-after-board" aria-label="Alur kerja sesudah menggunakan Oneflow">
              <div class="oneflow-after-alert"><i aria-hidden="true">${comparisonIcons.check}</i>Aksi tercatat otomatis</div>
              <div class="oneflow-after-thread">
                <div class="is-customer"><i aria-hidden="true">${comparisonIcons.messages}</i><span class="oneflow-after-thread-copy"><strong>Chat masuk dari banyak channel</strong><span>WhatsApp, Instagram, Messenger, dan lainnya</span></span></div>
                <div class="is-ai"><i aria-hidden="true">${comparisonIcons.bot}</i><span class="oneflow-after-thread-copy"><strong>AI bantu jawab otomatis</strong><span>Jawaban mengikuti informasi bisnis</span></span></div>
                <div class="is-action"><i aria-hidden="true">${comparisonIcons.clipboardPlus}</i><span class="oneflow-after-thread-copy"><strong>AI buat tindak lanjut</strong><span>Follow-up, order, atau booking langsung dibuat</span></span></div>
              </div>
              <div class="oneflow-after-branch" aria-hidden="true">
                <i></i>
                <span class="is-left"></span>
                <span class="is-right"></span>
              </div>
              <div class="oneflow-after-metrics">
                <div><i class="is-escalation" aria-hidden="true">${comparisonIcons.user}</i><strong>Perlu bantuan admin</strong><span>Tim bisa ambil alih kalau dibutuhkan</span></div>
                <div><i class="is-active" aria-hidden="true">${comparisonIcons.clipboard}</i><strong>Langsung dicatat sistem</strong><span>Order &amp; booking tersimpan otomatis</span></div>
              </div>
            </div>
          </article>
        </div>
      </div>
    </div>
  </div>
</section>`;

const productUpdatedMarkup =
  featureUpdatedMarkup.slice(0, productStart) +
  productMarkup +
  featureUpdatedMarkup.slice(productEnd);
const useCaseStart = productUpdatedMarkup.indexOf('<section id="use-case"');
const useCaseEnd =
  productUpdatedMarkup.indexOf('</section>', useCaseStart) + '</section>'.length;

if (useCaseStart < 0 || useCaseEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow use case section could not be parsed.');
}

const useCaseMarkup = `
<section id="use-case" data-scroll="load" class="oneflow-template-section section-spacing-bottom">
  <div class="container">
    <div class="oneflow-template-header">
      <div scroll-item="show" class="pre-center"><div data-wf--pre-title--variant="base" class="pre-title">Template agent</div></div>
      <h2 scroll-item="show" class="use-case-title">Pilih template sesuai alur bisnis kamu</h2>
      <p scroll-item="show" class="oneflow-template-description">Pilih alur yang paling dekat dengan bisnis kamu, lalu sesuaikan informasi dan aturan jawabannya.</p>
    </div>
    <div scroll-item="show" class="oneflow-template-showcase">
      <figure class="oneflow-template-visual">
        <img src="/oneflow_assets/usecases-workflow-board.svg" loading="lazy" alt="Alur Oneflow dari chat masuk, jawaban AI, handoff admin, hingga pipeline" class="oneflow-template-image"/>
      </figure>
      <div class="oneflow-template-grid">
        <article class="oneflow-template-card is-retail">
          <div class="oneflow-template-card-label"><span></span>Retail</div>
          <h3>Toko Online</h3>
          <p>Cek stok dan siapkan draft pesanan dari chat customer.</p>
          <div class="oneflow-template-flow"><span class="oneflow-template-flow-source">Tanya stok</span><span class="oneflow-template-flow-line"></span><span class="oneflow-template-flow-result">Draft order</span></div>
        </article>
        <article class="oneflow-template-card is-clinic">
          <div class="oneflow-template-card-label"><span></span>Klinik</div>
          <h3>Klinik &amp; Kesehatan</h3>
          <p>Arahkan booking dan handoff kasus yang perlu admin.</p>
          <div class="oneflow-template-flow"><span class="oneflow-template-flow-source">Minta jadwal</span><span class="oneflow-template-flow-line"></span><span class="oneflow-template-flow-result">Booking</span></div>
        </article>
        <article class="oneflow-template-card is-course">
          <div class="oneflow-template-card-label"><span></span>Kursus</div>
          <h3>Edukasi</h3>
          <p>Kirim jadwal kelas dan tindak lanjuti calon murid.</p>
          <div class="oneflow-template-flow"><span class="oneflow-template-flow-source">Tanya kelas</span><span class="oneflow-template-flow-line"></span><span class="oneflow-template-flow-result">Reminder</span></div>
        </article>
        <article class="oneflow-template-card is-sales">
          <div class="oneflow-template-card-label"><span></span>B2B Sales</div>
          <h3>B2B &amp; Demo</h3>
          <p>Kualifikasi lead dan simpan next action otomatis.</p>
          <div class="oneflow-template-flow"><span class="oneflow-template-flow-source">Minta demo</span><span class="oneflow-template-flow-line"></span><span class="oneflow-template-flow-result">Follow-up</span></div>
        </article>
      </div>
    </div>
    <div scroll-item="show" class="oneflow-template-more">
      <strong>Template lainnya</strong>
      <div><span>Properti</span><span>Travel</span><span>SOP Custom</span></div>
    </div>
    <div class="oneflow-template-more oneflow-template-plugins">
      <strong>Plugin saat dibutuhkan</strong>
      <div><span>Follow-up</span><span>Stok &amp; order</span><span>Booking</span><span>Notifikasi admin</span></div>
    </div>
  </div>
</section>`;

const useCaseUpdatedMarkup =
  productUpdatedMarkup.slice(0, useCaseStart) +
  useCaseMarkup +
  productUpdatedMarkup.slice(useCaseEnd);
const setupStart = useCaseUpdatedMarkup.indexOf('<section id="how-it-works"');
const setupEnd =
  useCaseUpdatedMarkup.indexOf('</section>', setupStart) + '</section>'.length;

if (setupStart < 0 || setupEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow setup section could not be parsed.');
}

const setupMarkup = `
<section id="how-it-works" class="oneflow-setup-section section-spacing-bottom">
  <div class="container">
    <div class="oneflow-setup-pin" data-oneflow-setup>
      <div class="oneflow-setup-grid">
        <div class="oneflow-setup-copy">
          <div class="pre-left"><div data-wf--pre-title--variant="base" class="pre-title">Cara kerja</div></div>
          <h2 class="step-title">Setup AI CS<br/>dalam 3 langkah</h2>
          <p class="step-description">Pilih template siap pakai, masukkan knowledge bisnis, lalu aktifkan plugin yang diperlukan. Admin tetap bisa mengawasi dan mengambil alih.</p>
          <div class="oneflow-setup-benefits">
            <article class="oneflow-setup-benefit">
              <i aria-hidden="true">${setupIcons.shield}</i>
              <h3>Terkontrol</h3>
          <p>Aturan jawaban dicek sebelum AI aktif</p>
            </article>
            <article class="oneflow-setup-benefit">
              <i aria-hidden="true">${setupIcons.rocket}</i>
          <h3>Lebih cepat mulai</h3>
              <p>Tidak perlu membangun alur dari nol</p>
            </article>
          </div>
        </div>
        <div class="oneflow-setup-board">
          <div class="oneflow-setup-progress" aria-label="Progres setup AI CS">
            <span class="oneflow-setup-progress-line" aria-hidden="true"><i></i></span>
            <span class="oneflow-setup-progress-item is-active" data-setup-step="0" aria-current="step">Step-01</span>
            <span class="oneflow-setup-progress-item" data-setup-step="1">Step 02</span>
            <span class="oneflow-setup-progress-item" data-setup-step="2">Step 03</span>
          </div>
          <div class="oneflow-setup-cards">
            <article class="oneflow-setup-card is-active" data-setup-panel="0">
              <span class="oneflow-setup-number">01</span>
              <img src="/oneflow_assets/step-template.svg" loading="lazy" alt="Pilih template agent sesuai jenis bisnis" class="oneflow-setup-image"/>
              <h3>Pilih template agent</h3>
              <p>Mulai dari template siap pakai sesuai jenis bisnis.</p>
            </article>
            <article class="oneflow-setup-card" data-setup-panel="1">
              <span class="oneflow-setup-number">02</span>
              <img src="/oneflow_assets/step-knowledge.svg" loading="lazy" alt="Validasi knowledge bisnis untuk AI CS" class="oneflow-setup-image"/>
              <h3>Validasi knowledge</h3>
              <p>Masukkan FAQ, produk, layanan, aturan fallback, dan notifikasi admin.</p>
            </article>
            <article class="oneflow-setup-card" data-setup-panel="2">
              <span class="oneflow-setup-number">03</span>
              <img src="/oneflow_assets/step-plugin.svg" loading="lazy" alt="Aktifkan plugin operasional yang dibutuhkan" class="oneflow-setup-image"/>
              <h3>Aktifkan plugin</h3>
              <p>Tambahkan follow-up, order, stok, booking, dan role akses sesuai kebutuhan.</p>
            </article>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>`;

const setupUpdatedMarkup =
  useCaseUpdatedMarkup.slice(0, setupStart) +
  setupMarkup +
  useCaseUpdatedMarkup.slice(setupEnd);

// Keep the workflow complete while removing two repeated legacy sections.
const compactMarkup = setupUpdatedMarkup
  .replace(
    /<section data-scroll="load" class="section-spacing-bottom"><div class="container"><div class="w-layout-grid grid-security-content">[\s\S]*?<\/section>/,
    '',
  )
  .replace(/<section id="integration" data-scroll="load">[\s\S]*?<\/section>/, '');

const salesCopyMarkup = compactMarkup
  .replaceAll(
    '/oneflow_assets/client-retail.svg',
    '/oneflow_assets/client-retail.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/client-klinik.svg',
    '/oneflow_assets/client-klinik.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/client-kursus.svg',
    '/oneflow_assets/client-kursus.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/client-properti.svg',
    '/oneflow_assets/client-properti.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/client-travel.svg',
    '/oneflow_assets/client-travel.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/client-b2b.svg',
    '/oneflow_assets/client-b2b.svg?v=category-icons',
  )
  .replaceAll(
    '/oneflow_assets/dashboard-workspace.svg',
    '/oneflow_assets/dashboard-workspace.svg?v=badge-polish-2',
  )
  .replaceAll(
    '/oneflow_assets/trust-security.svg',
    '/oneflow_assets/trust-security.svg?v=shield-compact',
  )
  .replaceAll(
    'Oneflow dashboard terpadu untuk inbox WhatsApp, plugin bisnis, template agent, dan role akses',
    'Oneflow.id Conversation Inbox dengan AI CS, status percakapan, handoff admin, dan alat bisnis',
  )
  .replace(
    'Semua chat, agent, plugin, dan akses akun dalam satu workspace',
    'Satu tempat untuk menjawab, menindaklanjuti, dan mengawasi',
  )
  .replace(
    'Kelola inbox WhatsApp, pipeline sales, order, booking, dan handoff admin tanpa pindah alat.',
    'Inbox WhatsApp, order, booking, dan handoff admin tetap terhubung tanpa pindah alat.',
  )
  .replace(/<div scroll-item="show" class="overview-button">[\s\S]*?<\/a><\/div>/, '')
  .replace(
    'Aktifkan alat operasional yang kamu perlukan',
    'Mulai sederhana, tambah plugin saat dibutuhkan',
  )
  .replace(
    'Follow-up, produk, stok, pesanan, booking, dan notifikasi admin tetap terhubung ke chat dan AI.',
    'Aktifkan follow-up, stok, order, booking, atau notifikasi admin sesuai alur bisnis kamu.',
  )
  .replace(
    /<div scroll-item="show" class="integration-button"><a button-lg="" data-wf--button-arrow--variant="base-button" href="https:\/\/oneflow\.id\/integrations"[\s\S]*?<div class="button-title">Lihat plugin<\/div>[\s\S]*?<\/a><\/div>/,
    '',
  )
  .replace(
    'Mulai dari alur WhatsApp bisnis kamu',
    'Pilih paket sesuai volume chat kamu',
  )
  .replace(
    '<h2 scroll-item="show" class="pricing-title">Pilih paket sesuai volume chat kamu</h2>',
    '<h2 scroll-item="show" class="pricing-title">Pilih paket sesuai volume chat kamu</h2><p class="oneflow-pricing-intro">Semua paket menampilkan batas credit AI, sesi WhatsApp, agent, dan anggota tim. Biaya Meta/BSP mengikuti kebijakan Meta dan dipisahkan dari credit AI Oneflow.</p>',
  )
  .replace(
    'Jawaban cepat tentang AI CS WhatsApp, plugin bisnis, akses akun, dan kontrol keamanan.',
    'Jawaban singkat soal setup, credit AI, aturan WhatsApp Meta, dan cara kerja Oneflow.',
  )
  .replace(
    'Bisa dipakai untuk lebih dari satu jenis bisnis?',
    'Bagaimana aturan follow-up WhatsApp?',
  )
  .replace(
    'Bisa. Mulai dari template retail, klinik, kursus, properti, travel, B2B, atau template kosong yang disesuaikan.',
    'Dalam 24 jam sejak pesan masuk dari customer, free-form dapat digunakan. Setelah lewat 24 jam, WhatsApp resmi memerlukan approved template dan biaya Meta/BSP terpisah dari credit AI Oneflow.',
  )
  .replace(
    'AI diarahkan dengan knowledge, prompt, fallback, dan guardrail. Saat pertanyaan tidak aman atau butuh keputusan manusia, percakapan bisa di-handoff ke admin.',
    'AI hanya menggunakan informasi bisnis dan aturan yang kamu setujui. Kalau konteksnya belum jelas atau butuh keputusan manusia, percakapan dapat diteruskan ke admin.',
  )
  .replace(
    'AI menggunakan knowledge bisnis, aturan jawaban, fallback, dan konteks chat. Admin tetap bisa mengambil alih saat dibutuhkan.',
    'AI membaca informasi bisnis, aturan jawaban, fallback, dan konteks percakapan sebelum merespons. Admin bisa mengambil alih saat dibutuhkan.',
  )
  .replace(
    'Bisa. Kamu bisa mulai dari demo untuk melihat inbox, template agent, plugin, dan role akses sebelum menentukan paket.',
    'Kamu bisa mulai dari dashboard untuk melihat setup, inbox, template agent, dan role akses sebelum menentukan paket.',
  )
  .replace(
    'Bisa. Oneflow mendukung role admin, supervisor, dan agent untuk tim yang mulai ramai.',
    'Bisa. Gunakan role admin, supervisor, dan agent agar tugas, handoff, dan tindak lanjut tim lebih jelas.',
  )
  .replace(
    'Paket Bisnis',
    'Butuh Paket Khusus?',
  )
  .replace(
    /<div class="cta-top"><div class="cta-list">[\s\S]*?<div class="cta-bottom">/,
    '<div class="cta-bottom">',
  )
  .replace(
    '<a data-wf--button-primary--variant="base" href="/contact" class="button-primary w-inline-block"><div class="button-primary-wrap"><div class="button-title">Hubungi sales</div></div></a>',
    '<a button-lg="" href="/contact" class="button-arrow oneflow-contact-button w-inline-block"><div class="button-arrow-content"><div button-icon-left="" class="button-icon-wrap left"><img src="/oneflow_vendor/73_69b2a2adca3cdacc51788ea9_68af63ddc6d3c51cfece3d1ab6a0fedd_button-dark-arrow.svg" loading="eager" alt="icon" class="button-icon"/></div><div button-text="" class="button-arrow-title-wrap right"><div class="button-title">Hubungi kami</div></div><div button-icon-right="" class="button-icon-wrap r"><img src="/oneflow_vendor/73_69b2a2adca3cdacc51788ea9_68af63ddc6d3c51cfece3d1ab6a0fedd_button-dark-arrow.svg" loading="eager" alt="icon" class="button-icon"/></div></div><div class="button-arrow-border"></div></a>',
  )
  .replace(
    /<div scroll-item="show" class="integration-center">[\s\S]*?<img class="integration-bottom-image" src="\/oneflow_assets\/plugin-suite\.svg" alt="[^"]*"\/>/,
    '<div scroll-item="show" class="integration-center oneflow-plugin-visual"><picture class="oneflow-plugin-picture"><source media="(max-width: 767px)" srcset="/oneflow_assets/integration-plugin-board-mobile.svg?v=lucide-library-1"/><source media="(max-width: 991px)" srcset="/oneflow_assets/integration-plugin-board-tablet.svg?v=lucide-library-1"/><img src="/oneflow_assets/integration-plugin-board.svg?v=lucide-library-1" loading="lazy" alt="Plugin bisnis Oneflow untuk follow-up, order, stok, booking, dan notifikasi admin" class="oneflow-plugin-board"/></picture></div>',
  );
const statStart = salesCopyMarkup.indexOf(
  '<section class="stat-section">',
);
const statEnd =
  salesCopyMarkup.indexOf('</section>', statStart) + '</section>'.length;

if (statStart < 0 || statEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow stat section could not be parsed.');
}

const conciseMarkup =
  salesCopyMarkup.slice(0, statStart) + salesCopyMarkup.slice(statEnd);
const testimonialStart = conciseMarkup.indexOf(
  '<section data-scroll="load" class="testimonial-section section-spacing">',
);
const testimonialEnd =
  conciseMarkup.indexOf('</section>', testimonialStart) + '</section>'.length;

if (testimonialStart < 0 || testimonialEnd < '</section>'.length) {
  throw new Error('The legacy Oneflow testimonial section could not be parsed.');
}

const legacyMarkup = (
  conciseMarkup.slice(0, testimonialStart) +
  conciseMarkup.slice(testimonialEnd)
)
  .replaceAll(
    '/oneflow_vendor/133_69b527a96a3f4aeae3c081fd_decoration%2002.svg',
    '/oneflow_vendor/decoration-02.svg',
  )
  .replaceAll(
    '/oneflow_vendor/29_69b527a9ba5e05b459d1cfae_decoration%2001.svg',
    '/oneflow_vendor/decoration-01.svg',
  )
  .replaceAll(
    '/oneflow_vendor/93_69b52a8c0aa5261e009a3e65_07c05cb10b6214e07d3dd217a2539f1f_bellll%201.png',
    '/oneflow_vendor/notification-bell.png',
  )
  .replace(
    '<a href="/about" class="nav-link w-nav-link">Fitur</a>',
    '<a data-oneflow-scroll="setup" href="#feature" class="nav-link w-nav-link">Fitur</a>',
  )
  .replace(
    '<a href="/feature" class="nav-link w-nav-link">Plugin Bisnis</a>',
    '<a data-oneflow-scroll="setup" href="#use-case" class="nav-link w-nav-link">Plugin Bisnis</a>',
  )
  .replace(
    '<a href="/pricing" class="nav-link w-nav-link">Harga</a>',
    '<a data-oneflow-scroll="setup" href="#pricing" class="nav-link w-nav-link">Harga</a>',
  )
  .replace(
    /<div data-hover="true" data-delay="0" class="dropdown w-dropdown">[\s\S]*?<\/nav><\/div>/,
    '<a href="/about" class="nav-link w-nav-link">Tentang Kami</a>',
  )
  .replace(
    '<a button-sm="" href="/contact" class="button-arrow-sm w-inline-block">',
    '<a button-sm="" href="/dashboard" class="button-arrow-sm w-inline-block">',
  )
  .replace(
    '<div class="button-title button-dark-sm-6">Demo</div>',
    '<div class="button-title button-dark-sm-6">Coba Gratis</div>',
  )
  .replace(
    /<div class="footer-top">[\s\S]*?<div scroll-item="show" class="footer-bottom">/,
    '<div scroll-item="show" class="footer-bottom">',
  )
  .replace(
    '<div class="footer-page-list"><div class="footer-links"><a href="/" aria-current="page" class="footer-link w--current">Home</a><a href="/about" class="footer-link">Fitur</a><a href="/feature" class="footer-link">Plugin Bisnis</a><a href="/blog" class="footer-link">Use Case</a><a href="/pricing" class="footer-link">Harga</a></div><div class="footer-links"><a href="/faqs" class="footer-link">FAQ</a><a href="/contact" class="footer-link">Kontak</a><a href="/request-a-demo" class="footer-link">Akses Akun</a><a href="/waitlist" class="footer-link">Login</a></div></div>',
    '<div class="footer-links"><a data-oneflow-scroll="setup" href="#how-it-works" class="footer-link">Alur CRM</a><a data-oneflow-scroll="setup" href="#feature" class="footer-link">Fitur</a><a data-oneflow-scroll="setup" href="#pricing" class="footer-link">Pricing</a></div>',
  )
  .replace(
    '<div class="footer-links"><a href="/utility-pages/style-guide" class="footer-link">Syarat &amp; Ketentuan</a><a href="/utility-pages/instructions" class="footer-link">Kebijakan Privasi</a><a href="/utility-pages/changelog" class="footer-link">Status</a><a href="/utility-pages/licenses" class="footer-link">Refund</a><a href="/privacy-policy" class="footer-link">Kebijakan Privasi</a><a href="/contact" class="footer-link">Kontak</a><a href="/waitlist" class="footer-link">Login</a></div>',
    '<div class="footer-links"><a href="/about" class="footer-link">Tentang Kami</a><a href="/terms" class="footer-link">Syarat &amp; Ketentuan</a><a href="/privacy" class="footer-link">Kebijakan Privasi</a></div></div><div class="footer-item"><h2 class="footer-item-title">Bantuan</h2><div class="footer-links"><a href="/contact" class="footer-link">Pusat Bantuan</a><a href="/contact" class="footer-link">Kontak</a><a href="/refund-policy" class="footer-link">Refund &amp; Pembatalan</a><a href="/data-deletion" class="footer-link">Penghapusan Data</a></div>',
  )
  .replace('<h2 class="footer-item-title">Legal</h2>', '<h2 class="footer-item-title">Perusahaan</h2>')
  .replace(
    'class="w-layout-grid grid-footer-right"',
    'class="w-layout-grid grid-footer-right oneflow-official-footer-nav"',
  )
  .replaceAll('href="/request-a-demo"', 'href="/dashboard"')
  .replaceAll('href="/utility-pages/style-guide"', 'href="/terms"')
  .replaceAll('href="/utility-pages/instructions"', 'href="/privacy"')
  .replaceAll('href="/utility-pages/licenses"', 'href="/refund-policy"')
  .replaceAll(
    '<a href="/privacy-policy" class="dropdown-link w-dropdown-link">Kebijakan Privasi</a>',
    '<a href="/data-deletion" class="dropdown-link w-dropdown-link">Penghapusan Data</a>',
  )
  .replaceAll(
    '<a href="/privacy-policy" class="footer-link">Kebijakan Privasi</a>',
    '<a href="/data-deletion" class="footer-link">Penghapusan Data</a>',
  )
  .replace(
    'Dibuat untuk operasional bisnis Indonesia. <a href="https://oneflow.id/" target="_blank" class="copyright-link">Oneflow.id</a>',
    '&copy; 2026 PT ONEFLOW TEKNOLOGI NUSANTARA. All rights reserved.',
  )
  .replace(/<div[^>]*class="footer-social-list"[^>]*>[\s\S]*?<\/div>/, '');

const pricingCardMarker =
  /<div class="pricing-info"><div class="w-layout-grid grid-pricing">[\s\S]*?<\/div><\/div><div class="pricing-bottom-list">[\s\S]*?<\/div><\/div>(?=<div class="pricing-bottom-info">)/;
const pricingIconSource =
  '/oneflow_vendor/137_69b7d1cbd6e96116d3445209_icon-02.svg';
const pricingBenefitsMarkup =
  '<div class="pricing-bottom-list"><div class="pricing-bottom-title">Credit mengikuti pemakaian AI</div><div class="pricing-dot"></div><div class="pricing-bottom-title">Biaya Meta/BSP terpisah</div><div class="pricing-dot"></div><div class="pricing-bottom-title">Bulanan atau tahunan</div></div>';

function escapeMarkup(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function formatIDR(value) {
  return `Rp${new Intl.NumberFormat('id-ID').format(Math.round(Number(value) || 0))}`;
}

function formatCount(value) {
  return new Intl.NumberFormat('id-ID').format(Math.max(0, Number(value) || 0));
}

function normalizeAnnualDiscountPercent(value) {
  const percentage = Number(value);
  return Number.isFinite(percentage) && percentage >= 0 && percentage < 100
    ? percentage
    : 10;
}

function formatPercentage(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(value);
}

function renderPricingListItem(label, dark = false) {
  const variant = dark
    ? ' w-variant-eab022e8-50a1-dce2-3e61-170b665a720a'
    : ' w-variant-1edcb659-3341-815e-63a2-10493468499c';
  const listVariant = dark ? 'sm-dark-bg' : 'sm';

  return `<div data-wf--list-item--variant="${listVariant}" class="list-item"><div class="list-icon-wrap${variant}"><img loading="lazy" src="${pricingIconSource}" alt="Arrow Icon" class="list-icon${variant}"/></div><div class="list-title${variant}">${escapeMarkup(label)}</div></div>`;
}

function renderPublicPlanCard(plan, annualDiscountPercent) {
  const dark = plan.isPopular === true;
  const textClass = dark ? ' text-white' : '';
  const descriptionClass = dark ? ' text-light' : '';
  const title = escapeMarkup(plan.name);
  const annualTotal = Math.round(Number(plan.price) * 12 * (1 - annualDiscountPercent / 100));
  const features = [
    `${formatCount(plan.creditAmount)} credit AI per bulan`,
    `${formatCount(plan.maxWhatsAppSessions)} koneksi WhatsApp`,
    `${formatCount(plan.maxAiAgents)} agent AI`,
    `${formatCount(plan.maxHumanUsers)} anggota tim`,
  ];
  const titleContent = `<div class="pricing-title-info"><div class="pricing-top-title${textClass}">${title}</div><p class="pricing-top-description${descriptionClass}">${escapeMarkup(plan.description)}</p></div>`;
  const badge = dark && plan.isPopular
    ? '<div class="offer-badge-info"><div class="price-offer-badge">Populer</div></div>'
    : '';

  return `<div class="pricing-item${dark ? ' dark' : ''}">${dark ? `<div class="price-content">${titleContent}${badge}</div>` : titleContent}<div class="pricing-amount oneflow-package-price"><div class="pricing-yearly oneflow-monthly-price"><div class="price-text-wrap"><h2 class="pricing-price${textClass}">${formatIDR(plan.price)}</h2></div><div class="pricing-text${textClass}">/ bulan</div></div><div class="pricing-yearly oneflow-yearly-price"><div class="price-text-wrap"><h2 class="pricing-price${textClass}">${formatIDR(annualTotal)}</h2></div><div class="pricing-text${textClass}">/ tahun</div></div></div><a data-wf--button-primary--variant="${dark ? 'white' : 'base'}" href="/contact" class="button-primary${dark ? ' w-variant-9cd3687a-f271-3144-935b-b543c06b439e' : ''} w-inline-block"><div class="button-primary-wrap"><div class="button-title">Tanya paket</div></div></a><div class="pricing-list">${features.map((feature) => renderPricingListItem(feature, dark)).join('')}</div></div>`;
}

function renderEmptyPublicPricingCard() {
  return '<div class="pricing-item oneflow-empty-plan"><div class="pricing-title-info"><div class="pricing-top-title">Paket sedang disiapkan</div><p class="pricing-top-description">Hubungi tim Oneflow untuk mendapatkan rekomendasi paket yang sesuai.</p></div><a data-wf--button-primary--variant="base" href="/contact" class="button-primary w-inline-block"><div class="button-primary-wrap"><div class="button-title">Tanya tim Oneflow</div></div></a></div>';
}

function applyPublicPricing(markup, plans, annualDiscountPercent) {
  const discount = normalizeAnnualDiscountPercent(annualDiscountPercent);
  const cards = plans.length
    ? plans.map((plan) => renderPublicPlanCard(plan, discount)).join('')
    : renderEmptyPublicPricingCard();

  return markup
    .replace(
      '<div class="pricing-top-item">',
      '<input class="oneflow-period-toggle-input" type="checkbox" id="oneflow-period-toggle"/><div class="pricing-top-item">',
    )
    .replace(
      '<div class="pricing-toggle"><div class="pricing-toggle-circle"></div></div>',
      '<label for="oneflow-period-toggle" class="oneflow-pricing-toggle" aria-label="Ganti periode harga"><div class="pricing-toggle-circle"></div></label>',
    )
    .replace(
      pricingCardMarker,
      `<div class="pricing-info oneflow-pricing-card-shell"><div class="w-layout-grid grid-pricing oneflow-public-pricing-grid">${cards}</div>${pricingBenefitsMarkup}</div></div>`,
    )
    .replace(
      '<div class="pricing-toggle-title">Tahunan</div><div class="pricing-offer-badge">fleksibel</div>',
      `<div class="pricing-toggle-title">Tahunan</div><div class="pricing-offer-badge">Hemat ${formatPercentage(discount)}%</div>`,
    );
}

async function getPublicPricing() {
  const apiTarget = resolveServerApiProxyTarget({ env: process.env });
  try {
    const response = await fetch(`${apiTarget}/api/public/billing/packages`, {
      cache: 'no-store',
    });
    if (!response.ok) return { plans: [], annualDiscountPercent: 10 };

    const payload = await response.json();
    return {
      plans: Array.isArray(payload.items)
        ? payload.items.filter((item) => item && Number(item.price) > 0)
        : [],
      annualDiscountPercent: normalizeAnnualDiscountPercent(payload.annualDiscountPercent),
    };
  } catch {
    return { plans: [], annualDiscountPercent: 10 };
  }
}

export default async function Home() {
  const publicPricing = await getPublicPricing();

  return (
    <main
      className="main"
      dangerouslySetInnerHTML={{ __html: applyPublicPricing(legacyMarkup, publicPricing.plans, publicPricing.annualDiscountPercent) }}
    />
  );
}
