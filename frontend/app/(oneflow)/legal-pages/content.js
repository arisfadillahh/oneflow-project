export const legalLinks = [
  { href: '/terms', label: 'Syarat & Ketentuan' },
  { href: '/privacy', label: 'Kebijakan Privasi' },
  { href: '/refund-policy', label: 'Refund & Pembatalan' },
  { href: '/data-deletion', label: 'Penghapusan Data' },
];

export const informationLinks = [
  { href: '/about', label: 'Tentang Kami' },
  { href: '/contact', label: 'Kontak' },
];

export const footerGroups = [
  {
    title: 'Produk',
    links: [
      { href: '/#how-it-works', label: 'Alur CRM' },
      { href: '/#feature', label: 'Fitur' },
      { href: '/#pricing', label: 'Pricing' },
    ],
  },
  {
    title: 'Perusahaan',
    links: [
      { href: '/about', label: 'Tentang Kami' },
      { href: '/terms', label: 'Syarat & Ketentuan' },
      { href: '/privacy', label: 'Kebijakan Privasi' },
    ],
  },
  {
    title: 'Bantuan',
    links: [
      { href: '/contact', label: 'Pusat Bantuan' },
      { href: '/contact', label: 'Kontak' },
      { href: '/refund-policy', label: 'Refund & Pembatalan' },
      { href: '/data-deletion', label: 'Penghapusan Data' },
    ],
  },
];

export const informationDocuments = {
  about: {
    badge: 'Tentang Kami',
    title: 'Tentang Oneflow.id',
    description:
      'Oneflow.id membantu bisnis mengubah chat WhatsApp menjadi inbox tim, kontak, pipeline deal, reminder, broadcast segment, dan dashboard CRM yang rapi.',
    updatedAt: '11 Mei 2026',
    sections: [
      {
        id: 'tentang-oneflow',
        title: 'Tentang Oneflow.id',
        paragraphs: [
          'Oneflow.id membantu bisnis merapikan percakapan WhatsApp menjadi workflow sales dan operasional yang bisa dipantau. Fokusnya adalah AI response, inbox tim, Contact 360, pipeline deal, reminder follow-up, broadcast segment, dan dashboard CRM.',
          'Produk ini dirancang untuk tim sales, owner, admin operasional, dan bisnis yang ingin respons pelanggan lebih cepat tanpa kehilangan konteks percakapan.',
        ],
      },
      {
        id: 'cara-kerja',
        title: 'Cara kerja',
        bullets: [
          'Chat WhatsApp masuk ke inbox tim.',
          'Percakapan dapat dibantu AI atau diteruskan ke agent manusia.',
          'Kontak, deal, owner, due date, reminder, dan ringkasan aktivitas dicatat di CRM.',
          'Owner dan admin bisa memantau open deal, overdue follow-up, pending chat, dan aktivitas tim.',
        ],
      },
    ],
  },
  contact: {
    badge: 'Bantuan',
    title: 'Kontak & Legal Notice',
    description:
      'Kanal resmi untuk bantuan akun, billing, refund, pertanyaan legal, privasi, dan informasi administratif Oneflow.id.',
    updatedAt: '11 Mei 2026',
    sections: [
      {
        id: 'kontak-support',
        title: 'Kontak support',
        paragraphs: [
          'Email support: support@oneflow.id',
          'Gunakan kontak support untuk bantuan akun, onboarding, billing, paket, payment status, refund, bug report, atau kendala penggunaan dashboard.',
        ],
      },
      {
        id: 'kontak-legal-dan-privasi',
        title: 'Kontak legal dan privasi',
        paragraphs: [
          'Pertanyaan legal dan Syarat & Ketentuan: legal@oneflow.id',
          'Permintaan data pribadi dan privasi: privacy@oneflow.id',
        ],
      },
      {
        id: 'legal-notice',
        title: 'Legal notice',
        paragraphs: [
          'Pengelola layanan: Oneflow.id.',
          'Kategori layanan: Software as a Service (SaaS), AI chat automation, CRM WhatsApp, sales pipeline, dan workflow operasional untuk bisnis.',
          'Wilayah operasional utama: Indonesia.',
          'Data badan hukum, alamat korespondensi, NPWP, dan informasi administrasi resmi mengikuti invoice, kontrak, dokumen merchant, atau dokumen kerja sama yang disepakati dengan pelanggan.',
        ],
      },
      {
        id: 'jam-respons',
        title: 'Jam respons',
        paragraphs: [
          'Permintaan support diproses pada hari kerja Indonesia. Laporan gangguan kritis dan keamanan akan diprioritaskan sesuai tingkat dampaknya.',
        ],
      },
    ],
  },
  pricing: {
    badge: 'Pricing',
    title: 'Pilih paket yang pas untuk tim kamu',
    description:
      'Semua paket dirancang untuk bantu tim sales balas lebih cepat, rapi data prospek, dan tutup lebih banyak deal.',
    updatedAt: '11 Mei 2026',
    sections: [
      {
        id: 'starter',
        title: 'Starter - Rp 99 rb / bulan',
        paragraphs: [
          'Untuk mulai balas chat lebih cepat dan rapihin data prospek.',
          'Bayar fleksibel setiap bulan. Termasuk 1.000 credit / bulan.',
        ],
        bullets: [
          '1 sesi WhatsApp, 2 AI agent, dan 3 human user.',
          'Inbox WhatsApp, Contact 360, Pipeline deal, dan Reminder follow-up.',
        ],
        action: {
          href: '/dashboard',
          label: 'Pilih Starter',
        },
      },
      {
        id: 'growth',
        title: 'Growth - Rp 299 rb / bulan',
        paragraphs: [
          'Untuk tim yang mulai butuh pipeline, broadcast, dan follow-up terkontrol.',
          'Bayar fleksibel setiap bulan. Termasuk 3.000 credit / bulan.',
        ],
        bullets: [
          '2 sesi WhatsApp, 5 AI agent, dan 10 human user.',
          'Semua fitur Starter, ditambah Broadcast segment, Dashboard CRM, dan Multi-agent workflow.',
        ],
        action: {
          href: '/dashboard',
          label: 'Pilih Growth',
        },
      },
      {
        id: 'business',
        title: 'Business - Rp 699 rb / bulan',
        paragraphs: [
          'Untuk volume chat tinggi, multi-session, dan proses sales yang makin serius.',
          'Bayar fleksibel setiap bulan. Termasuk 7.000 credit / bulan.',
        ],
        bullets: [
          '5 sesi WhatsApp, 15 AI agent, dan 25 human user.',
          'Semua fitur Growth, ditambah lebih banyak sesi, tim lebih besar, dan workflow operasional penuh.',
        ],
        action: {
          href: '/dashboard',
          label: 'Pilih Business',
        },
      },
      {
        id: 'dukungan-paket',
        title: 'Dukungan paket',
        bullets: [
          'Aman & Terpercaya: data kamu dienkripsi dan disimpan dengan aman.',
          'Support Cepat: tim kami siap bantu kamu setiap hari kerja.',
          'Bisa Upgrade Kapan Saja: naik atau turun paket kapan pun sesuai kebutuhan.',
          'Tanpa Komitmen: berlangganan bulanan, batalkan kapan saja.',
        ],
      },
      {
        id: 'pertanyaan-umum',
        title: 'Pertanyaan yang sering ditanyakan',
        bullets: [
          'Apakah bisa ganti paket kapan saja?',
          'Apakah ada biaya setup?',
          'Apa itu credit?',
          'Bagaimana jika saya butuh lebih dari paket ini?',
        ],
      },
    ],
  },
};

export const legalDocuments = {
  refundPolicy: {
    badge: 'Kebijakan pembayaran',
    title: 'Kebijakan Refund & Pembatalan',
    description:
      'Ketentuan pembatalan paket, refund pembayaran, credit bulanan, dan proses pengembalian dana untuk layanan digital Oneflow.id.',
    updatedAt: '11 Mei 2026',
    sections: [
      {
        id: 'sifat-layanan',
        title: 'Sifat layanan',
        paragraphs: [
          'Oneflow.id adalah layanan digital/SaaS. Tidak ada pengiriman barang fisik, sehingga kebijakan pengembalian barang, ongkos kirim, dan retur fisik tidak berlaku.',
          'Akses paket, credit bulanan, AI agent, sesi WhatsApp, dan fitur CRM mulai berlaku setelah pembayaran diverifikasi dan akun diaktifkan.',
        ],
      },
      {
        id: 'pembatalan-subscription',
        title: 'Pembatalan subscription',
        paragraphs: [
          'Pelanggan dapat mengajukan pembatalan paket kapan saja melalui kanal support. Pembatalan berlaku untuk periode penagihan berikutnya.',
          'Untuk paket bulanan, akses tetap aktif sampai akhir periode berjalan. Untuk paket tahunan, akses tetap aktif sampai akhir periode tahunan yang sudah dibayar kecuali disepakati lain secara tertulis.',
          'Sisa credit atau kuota pada periode berjalan tidak dapat diuangkan, dipindahkan ke pihak lain, atau dikembalikan dalam bentuk kas.',
        ],
      },
      {
        id: 'refund-yang-dapat-dipertimbangkan',
        title: 'Refund yang dapat dipertimbangkan',
        bullets: [
          'Pembayaran ganda untuk invoice atau order yang sama.',
          'Kesalahan nominal tagihan yang dapat diverifikasi.',
          'Akun berbayar gagal diaktifkan karena masalah dari sisi Oneflow.id dan tidak dapat diselesaikan dalam waktu yang wajar.',
          'Transaksi tidak sah yang terbukti setelah proses verifikasi.',
          'Kesepakatan tertulis khusus antara pelanggan dan Oneflow.id.',
        ],
      },
      {
        id: 'refund-yang-umumnya-tidak-berlaku',
        title: 'Refund yang umumnya tidak berlaku',
        bullets: [
          'Perubahan keputusan setelah paket aktif dan layanan sudah digunakan.',
          'Sisa credit atau kuota yang tidak dipakai sampai akhir periode.',
          'Gangguan yang disebabkan koneksi internet, perangkat, kredensial, konfigurasi WhatsApp, kebijakan Meta/WhatsApp, atau layanan pihak ketiga di luar kendali Oneflow.id.',
          'Akun yang dihentikan karena spam, penyalahgunaan, pelanggaran hukum, pelanggaran hak pihak lain, atau pelanggaran Syarat & Ketentuan.',
        ],
      },
      {
        id: 'cara-mengajukan-refund',
        title: 'Cara mengajukan refund',
        paragraphs: [
          'Kirim permintaan ke support@oneflow.id dengan mencantumkan nama organisasi, email akun, invoice/order ID, tanggal pembayaran, nominal transaksi, metode pembayaran, dan alasan pengajuan refund.',
          'Oneflow.id akan meninjau permintaan dan dapat meminta data tambahan untuk verifikasi. Pengajuan refund sebaiknya dilakukan paling lambat 7 hari kalender sejak tanggal pembayaran untuk mempercepat penanganan.',
        ],
      },
      {
        id: 'proses-pengembalian-dana',
        title: 'Proses pengembalian dana',
        paragraphs: [
          'Jika refund disetujui, pengembalian dana akan diproses ke metode pembayaran awal apabila metode tersebut mendukung refund melalui payment gateway. Jika metode pembayaran tidak mendukung refund otomatis, Oneflow.id dapat memproses pengembalian secara manual setelah verifikasi.',
          'Waktu dana diterima pelanggan bergantung pada bank, penerbit kartu, e-wallet, payment gateway, dan metode pembayaran yang digunakan. Biaya administrasi, biaya payment gateway, atau biaya bank dapat mengurangi nominal refund jika biaya tersebut tidak dapat dikembalikan oleh penyedia pembayaran.',
        ],
      },
      {
        id: 'cancel-transaksi-belum-settlement',
        title: 'Cancel transaksi belum settlement',
        paragraphs: [
          'Untuk transaksi yang belum selesai atau belum settlement, pembatalan dapat mengikuti status transaksi pada payment gateway. Apabila transaksi sudah settlement, proses yang tersedia umumnya adalah refund sesuai metode pembayaran dan ketentuan penyedia pembayaran.',
        ],
      },
    ],
  },
  privacy: {
    badge: 'Privasi & data',
    title: 'Kebijakan Privasi',
    description:
      'Cara Oneflow.id mengumpulkan, menggunakan, menyimpan, melindungi, dan membagikan data untuk menyediakan layanan CRM WhatsApp berbasis AI.',
    updatedAt: '25 Mei 2026',
    sections: [
      {
        id: 'ringkasan',
        title: 'Ringkasan',
        paragraphs: [
          'Kebijakan Privasi ini menjelaskan bagaimana Oneflow.id mengumpulkan, menggunakan, menyimpan, melindungi, dan membagikan data saat kamu menggunakan situs, dashboard, fitur AI chat automation, CRM WhatsApp, dan layanan terkait.',
          'Oneflow.id memproses data untuk menyediakan layanan digital, mengelola akun, memproses pembayaran, menjaga keamanan, memberikan dukungan, dan memenuhi kewajiban hukum yang berlaku.',
        ],
      },
      {
        id: 'data-yang-kami-kumpulkan',
        title: 'Data yang kami kumpulkan',
        bullets: [
          'Data akun: nama, username, email, nomor telepon, nama organisasi, peran pengguna, dan kredensial yang diproses secara aman.',
          'Data layanan: konfigurasi WhatsApp, daftar kontak, chat, riwayat percakapan, tag, lifecycle, pipeline deal, task, reminder, broadcast segment, agent AI, dan knowledge base.',
          'Data pembayaran: paket, invoice, status pembayaran, metode pembayaran, order ID, tanggal transaksi, dan catatan administrasi.',
          'Data teknis: alamat IP, perangkat, browser, log aktivitas, cookies, token sesi, error log, dan metadata keamanan.',
          'Data komunikasi: pesan support, laporan masalah, feedback, dan korespondensi dengan tim Oneflow.id.',
        ],
      },
      {
        id: 'tujuan-pemrosesan',
        title: 'Tujuan pemrosesan',
        bullets: [
          'Menyediakan dashboard, inbox, CRM, AI response, routing human agent, reminder, pipeline, dan fitur operasional lain.',
          'Mengaktifkan paket, menghitung credit, mengelola subscription, invoice, payment status, refund, dan pembatalan.',
          'Menjaga keamanan akun, mencegah penyalahgunaan, melakukan audit teknis, dan memperbaiki gangguan layanan.',
          'Memberikan bantuan pelanggan, onboarding, update produk, notifikasi penting, dan komunikasi administratif.',
          'Menganalisis performa layanan secara agregat untuk peningkatan produk tanpa menjual data pribadi pelanggan.',
        ],
      },
      {
        id: 'dasar-pemrosesan',
        title: 'Dasar pemrosesan',
        paragraphs: [
          'Pemrosesan data dilakukan berdasarkan persetujuan, pelaksanaan kontrak layanan, kepentingan sah untuk menjaga keamanan dan kualitas layanan, serta kewajiban hukum atau administratif yang berlaku.',
          'Untuk data pelanggan akhir yang dimasukkan oleh pengguna ke Oneflow.id, pengguna bertanggung jawab memastikan dasar hukum, persetujuan, atau pemberitahuan privasi yang diperlukan telah dipenuhi.',
        ],
      },
      {
        id: 'pembagian-data-kepada-pihak-ketiga',
        title: 'Pembagian data kepada pihak ketiga',
        paragraphs: [
          'Oneflow.id dapat membagikan data secara terbatas kepada penyedia infrastruktur, payment gateway, penyedia komunikasi WhatsApp atau integrasi terkait, layanan email/support, keamanan, analytics, auditor, penasihat profesional, atau otoritas yang sah.',
          'Pihak ketiga hanya menerima data yang diperlukan untuk menjalankan fungsi terkait dan wajib menjaga kerahasiaan serta keamanan data sesuai peran mereka.',
        ],
      },
      {
        id: 'keamanan-dan-penyimpanan',
        title: 'Keamanan dan penyimpanan',
        paragraphs: [
          'Oneflow.id menerapkan kontrol keamanan yang wajar seperti pembatasan akses, enkripsi saat sesuai, pemantauan sistem, backup, audit teknis, dan pemisahan akses berdasarkan peran.',
          'Data disimpan selama akun aktif, selama diperlukan untuk menyediakan layanan, menyelesaikan kewajiban pembayaran, memenuhi kewajiban hukum, menangani sengketa, atau menjaga keamanan. Data dapat dihapus atau dianonimkan ketika tidak lagi diperlukan.',
        ],
      },
      {
        id: 'cookies-dan-teknologi-serupa',
        title: 'Cookies dan teknologi serupa',
        paragraphs: [
          'Situs dan dashboard dapat menggunakan cookies atau penyimpanan lokal untuk login, keamanan sesi, preferensi tampilan, analytics dasar, dan peningkatan pengalaman pengguna.',
          'Kamu dapat mengatur browser untuk menolak cookies tertentu, tetapi beberapa fitur dashboard mungkin tidak berjalan normal.',
        ],
      },
      {
        id: 'hak-pengguna',
        title: 'Hak pengguna',
        paragraphs: [
          'Kamu dapat meminta akses, koreksi, pembaruan, penghapusan, pembatasan pemrosesan, atau penarikan persetujuan atas data pribadi dengan menghubungi privacy@oneflow.id.',
          'Permintaan dapat ditolak atau dibatasi apabila data masih diperlukan untuk kewajiban hukum, keamanan, penyelesaian transaksi, penyelesaian sengketa, atau kepentingan sah lain yang diizinkan hukum.',
        ],
      },
      {
        id: 'cara-meminta-penghapusan-data',
        title: 'Cara meminta penghapusan data',
        paragraphs: [
          'Kirim permintaan penghapusan data ke privacy@oneflow.id dari alamat email akun yang terdaftar, dengan subjek "Permintaan Penghapusan Data - Oneflow.id".',
          'Instruksi lengkap mencakup data identifikasi yang diperlukan, proses verifikasi, waktu pemrosesan, dan ketentuan khusus untuk koneksi Meta atau WhatsApp.',
        ],
        action: {
          href: '/data-deletion',
          label: 'Lihat Instruksi Penghapusan Data',
        },
      },
      {
        id: 'transfer-dan-pemrosesan-lintas-wilayah',
        title: 'Transfer dan pemrosesan lintas wilayah',
        paragraphs: [
          'Sebagian penyedia infrastruktur atau integrasi dapat memproses data di luar Indonesia. Dalam hal tersebut, Oneflow.id akan menggunakan langkah yang wajar untuk memastikan data tetap dilindungi sesuai ketentuan yang berlaku.',
        ],
      },
      {
        id: 'anak-anak',
        title: 'Anak-anak',
        paragraphs: [
          'Layanan Oneflow.id ditujukan untuk bisnis dan organisasi, bukan untuk anak di bawah 18 tahun. Oneflow.id tidak secara sengaja mengumpulkan data anak sebagai pemilik akun.',
        ],
      },
      {
        id: 'perubahan-kebijakan',
        title: 'Perubahan kebijakan',
        paragraphs: [
          'Kebijakan Privasi dapat diperbarui dari waktu ke waktu. Perubahan material akan diinformasikan melalui situs, dashboard, email, atau kanal resmi lain.',
        ],
      },
    ],
  },
  terms: {
    badge: 'Legal',
    title: 'Syarat & Ketentuan Oneflow.id',
    description:
      'Ketentuan penggunaan layanan digital Oneflow.id untuk CRM WhatsApp, AI response, pipeline sales, subscription, billing, dan operasional akun.',
    updatedAt: '11 Mei 2026',
    sections: [
      {
        id: 'penerimaan-ketentuan',
        title: 'Penerimaan ketentuan',
        paragraphs: [
          'Dengan mengakses situs, membuat akun, mencoba dashboard, membeli paket, atau menggunakan layanan Oneflow.id, kamu menyetujui Syarat & Ketentuan ini beserta kebijakan lain yang dirujuk di dalamnya.',
          'Jika kamu menggunakan Oneflow.id atas nama perusahaan, organisasi, atau tim, kamu menyatakan bahwa kamu berwenang untuk mewakili pihak tersebut.',
        ],
      },
      {
        id: 'layanan-oneflow',
        title: 'Layanan Oneflow.id',
        paragraphs: [
          'Oneflow.id adalah layanan digital berbasis software as a service untuk membantu tim mengelola chat WhatsApp, AI response, inbox tim, Contact 360, pipeline deal, reminder follow-up, broadcast segment, dashboard CRM, dan workflow operasional terkait.',
          'Fitur dapat berubah, ditambah, dibatasi, atau dihentikan dari waktu ke waktu untuk alasan keamanan, kepatuhan, peningkatan produk, atau kebutuhan operasional.',
        ],
      },
      {
        id: 'akun-dan-keamanan',
        title: 'Akun dan keamanan',
        paragraphs: [
          'Kamu wajib memberikan data pendaftaran yang benar, akurat, dan terbaru. Kamu bertanggung jawab atas keamanan username, password, akses dashboard, konfigurasi WhatsApp, agent AI, knowledge base, dan aktivitas yang terjadi di akunmu.',
          'Segera hubungi Oneflow.id jika kamu mencurigai akses tidak sah, kebocoran kredensial, atau aktivitas yang tidak kamu kenali.',
        ],
      },
      {
        id: 'paket-credit-dan-harga',
        title: 'Paket, credit, dan harga',
        paragraphs: [
          'Paket publik Oneflow.id saat ini terdiri dari Starter, Growth, dan Business dengan limit credit, sesi WhatsApp, AI agent, dan human user sebagaimana ditampilkan pada halaman Pricing.',
          'Harga ditampilkan dalam Rupiah. Mode tahunan menampilkan estimasi harga per bulan setelah diskon dan tagihan dilakukan berdasarkan periode tahunan apabila pelanggan memilih pembayaran tahunan.',
          'Credit bulanan digunakan untuk pemakaian layanan selama periode berjalan dan tidak dapat diuangkan, dialihkan, atau diklaim sebagai saldo kas kecuali dinyatakan tertulis oleh Oneflow.id.',
          'Oneflow.id dapat menyesuaikan harga, limit, atau fitur paket. Perubahan material akan diinformasikan melalui situs, dashboard, email, invoice, atau kanal komunikasi resmi lainnya.',
        ],
      },
      {
        id: 'pembayaran-dan-invoice',
        title: 'Pembayaran dan invoice',
        paragraphs: [
          'Pembayaran dapat diproses melalui kanal pembayaran yang tersedia, termasuk payment gateway, transfer, atau metode lain yang diaktifkan oleh Oneflow.id.',
          'Akses berbayar dapat diaktifkan setelah pembayaran terverifikasi. Apabila pembayaran gagal, kedaluwarsa, dibatalkan, atau terindikasi bermasalah, Oneflow.id dapat menunda aktivasi atau membatasi akses.',
          'Pajak, biaya administrasi, biaya payment gateway, atau biaya bank dapat dikenakan sesuai ketentuan yang berlaku dan akan mengikuti informasi pada invoice atau halaman pembayaran.',
        ],
      },
      {
        id: 'pembatalan-suspend-dan-terminasi',
        title: 'Pembatalan, suspend, dan terminasi',
        paragraphs: [
          'Pelanggan dapat mengajukan pembatalan paket kapan saja. Pembatalan berlaku untuk periode berikutnya; akses yang sudah dibayar tetap dapat digunakan sampai akhir periode berjalan kecuali akun dihentikan karena pelanggaran.',
          'Oneflow.id dapat menangguhkan atau menghentikan akun jika terjadi penyalahgunaan layanan, pelanggaran hukum, spam, aktivitas yang mengganggu sistem, pelanggaran hak pihak lain, gagal bayar, atau penggunaan yang melanggar kebijakan WhatsApp/Meta dan penyedia layanan terkait.',
        ],
      },
      {
        id: 'refund',
        title: 'Refund',
        paragraphs: [
          'Karena Oneflow.id adalah layanan digital/SaaS, pembayaran yang sudah mengaktifkan akses layanan atau sudah digunakan pada periode berjalan pada prinsipnya tidak dapat dikembalikan.',
          'Refund dapat dipertimbangkan untuk kondisi tertentu seperti pembayaran ganda, kesalahan nominal tagihan, akun berbayar yang gagal diaktifkan karena masalah dari sisi Oneflow.id, atau transaksi tidak sah yang dapat diverifikasi.',
          'Rincian proses refund, batas waktu pengajuan, dan metode pengembalian dana dijelaskan pada halaman Refund & Pembatalan.',
        ],
        action: {
          href: '/refund-policy',
          label: 'Buka Refund & Pembatalan',
        },
      },
      {
        id: 'kewajiban-pengguna',
        title: 'Kewajiban pengguna',
        paragraphs: [
          'Kamu wajib menggunakan Oneflow.id secara sah, etis, dan bertanggung jawab. Kamu dilarang menggunakan layanan untuk spam, penipuan, phishing, konten ilegal, pelanggaran privasi, pelanggaran hak kekayaan intelektual, atau aktivitas yang merugikan pihak lain.',
          'Kamu bertanggung jawab memastikan bahwa pesan WhatsApp, data kontak, data pelanggan, broadcast, template, dan knowledge base yang kamu masukkan ke Oneflow.id diperoleh dan digunakan dengan dasar hukum atau persetujuan yang sesuai.',
        ],
      },
      {
        id: 'ai-response-dan-batasan-hasil',
        title: 'AI response dan batasan hasil',
        paragraphs: [
          'Fitur AI membantu membuat respons, ringkasan, klasifikasi, reminder, atau rekomendasi operasional. Output AI dapat tidak akurat, tidak lengkap, atau tidak sesuai konteks jika data masukan tidak memadai.',
          'Kamu bertanggung jawab meninjau, mengoreksi, dan mengawasi penggunaan AI response sebelum dikirim ke pelanggan. Oneflow.id tidak menjamin peningkatan penjualan, closing deal, atau hasil bisnis tertentu.',
        ],
      },
      {
        id: 'data-dan-privasi',
        title: 'Data dan privasi',
        paragraphs: [
          'Pengelolaan data pribadi, data kontak, riwayat percakapan, file knowledge base, metadata WhatsApp, log teknis, dan data pembayaran dijelaskan lebih lanjut pada Kebijakan Privasi.',
          'Dengan menggunakan layanan, kamu menyetujui pemrosesan data yang diperlukan untuk menyediakan, mengamankan, menagihkan, dan meningkatkan layanan Oneflow.id.',
        ],
        action: {
          href: '/privacy',
          label: 'Buka Kebijakan Privasi',
        },
      },
      {
        id: 'hak-kekayaan-intelektual',
        title: 'Hak kekayaan intelektual',
        paragraphs: [
          'Situs, logo, desain, kode, dokumentasi, fitur, workflow, dan materi Oneflow.id dilindungi oleh hak kekayaan intelektual. Kamu tidak diperbolehkan menyalin, memodifikasi, membongkar, menjual kembali, atau membuat layanan turunan dari Oneflow.id tanpa izin tertulis.',
          'Data bisnis, konten knowledge base, dan data pelanggan yang kamu masukkan tetap menjadi tanggung jawab dan milik kamu atau organisasi kamu, sepanjang kamu memiliki hak untuk menggunakannya.',
        ],
      },
      {
        id: 'batasan-tanggung-jawab',
        title: 'Batasan tanggung jawab',
        paragraphs: [
          'Oneflow.id berupaya menjaga layanan tetap aman dan tersedia, tetapi tidak menjamin layanan selalu bebas gangguan, bebas error, atau kompatibel dengan semua integrasi pihak ketiga.',
          'Sepanjang diperbolehkan hukum yang berlaku, tanggung jawab Oneflow.id dibatasi pada nilai biaya layanan yang dibayarkan untuk periode terkait dan tidak mencakup kerugian tidak langsung, kehilangan keuntungan, kehilangan data akibat kelalaian pengguna, atau gangguan dari pihak ketiga.',
        ],
      },
      {
        id: 'hukum-yang-berlaku-dan-kontak',
        title: 'Hukum yang berlaku dan kontak',
        paragraphs: [
          'Syarat & Ketentuan ini tunduk pada hukum Republik Indonesia.',
          'Pertanyaan terkait ketentuan layanan dapat dikirim ke legal@oneflow.id. Pertanyaan operasional dapat dikirim ke support@oneflow.id.',
        ],
      },
    ],
  },
  dataDeletion: {
    badge: 'Privasi & data',
    title: 'Instruksi Penghapusan Data Pengguna',
    description:
      'Cara meminta penghapusan data pribadi atau data koneksi Meta dan WhatsApp yang tersimpan dalam layanan Oneflow.id.',
    updatedAt: '25 Mei 2026',
    sections: [
      {
        id: 'cara-mengajukan-permintaan',
        title: 'Cara mengajukan permintaan',
        paragraphs: [
          'Kirim email ke privacy@oneflow.id dengan subjek "Permintaan Penghapusan Data - Oneflow.id". Gunakan alamat email yang terdaftar pada akun Oneflow.id jika kamu memilikinya.',
          'Jangan mengirim password, access token, kode verifikasi, isi percakapan pelanggan, atau kredensial rahasia lain melalui email.',
        ],
        bullets: [
          'Nama pemohon dan email akun Oneflow.id yang terkait.',
          'Nama organisasi atau workspace yang terkait dengan permintaan.',
          'Nomor telepon bisnis atau identitas koneksi WhatsApp/Meta yang relevan, jika permintaan terkait integrasi WhatsApp.',
          'Ruang lingkup permintaan: penghapusan akun pribadi, pencabutan koneksi integrasi, atau data tertentu yang ingin dihapus.',
        ],
      },
      {
        id: 'permintaan-terkait-meta-atau-whatsapp',
        title: 'Permintaan terkait Meta atau WhatsApp',
        paragraphs: [
          'Jika kamu menghubungkan WhatsApp melalui Meta atau Facebook Login for Business, kamu dapat meminta penghapusan data yang diterima Oneflow.id melalui integrasi tersebut dengan prosedur email yang sama.',
          'Apabila permintaan berkaitan dengan akun bisnis atau data organisasi, Oneflow.id akan memverifikasi bahwa pemohon berwenang mewakili organisasi sebelum menghapus koneksi, kredensial tersimpan, atau data layanan terkait.',
        ],
      },
      {
        id: 'verifikasi-dan-konfirmasi',
        title: 'Verifikasi dan konfirmasi',
        paragraphs: [
          'Oneflow.id akan memeriksa identitas pemohon dan kewenangan atas akun atau organisasi terkait. Kami dapat meminta informasi tambahan yang wajar untuk mencegah penghapusan oleh pihak yang tidak berwenang.',
          'Setelah permintaan dapat diverifikasi, kami akan mengirim konfirmasi penerimaan dan informasi status melalui email yang digunakan untuk permintaan.',
        ],
      },
      {
        id: 'data-yang-dapat-dihapus',
        title: 'Data yang dapat dihapus',
        bullets: [
          'Data profil akun pribadi dan preferensi yang terkait dengan pengguna pemohon.',
          'Koneksi atau kredensial integrasi Meta/WhatsApp yang dikelola Oneflow.id, jika pemohon berwenang atas organisasi terkait.',
          'Data layanan tertentu yang terkait akun atau organisasi, sejauh penghapusan diminta secara sah dan tidak bertentangan dengan kewajiban retensi.',
        ],
      },
      {
        id: 'waktu-pemrosesan-dan-retensi-terbatas',
        title: 'Waktu pemrosesan dan retensi terbatas',
        paragraphs: [
          'Oneflow.id menargetkan penyelesaian permintaan penghapusan paling lambat 30 hari kalender setelah identitas dan ruang lingkup permintaan terverifikasi, kecuali diperlukan waktu tambahan yang akan diinformasikan kepada pemohon.',
          'Sebagian data dapat tetap disimpan secara terbatas jika diperlukan untuk kewajiban hukum, pencatatan transaksi dan invoice, pencegahan fraud, keamanan, penyelesaian sengketa, atau pembuktian kepatuhan. Data tersebut tidak digunakan kembali untuk tujuan pemasaran yang tidak terkait.',
        ],
      },
      {
        id: 'kontak',
        title: 'Kontak',
        paragraphs: [
          'Pertanyaan dan permintaan penghapusan data dapat dikirim ke privacy@oneflow.id. Pertanyaan umum layanan dapat dikirim ke support@oneflow.id.',
        ],
      },
    ],
  },
};
