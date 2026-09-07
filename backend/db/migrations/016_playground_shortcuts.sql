CREATE TABLE IF NOT EXISTS playground_shortcuts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  label TEXT NOT NULL,
  question TEXT NOT NULL,
  expected_behavior TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 100,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_playground_shortcuts_active_order
  ON playground_shortcuts (is_active, sort_order, created_at);

INSERT INTO playground_shortcuts (label, question, expected_behavior, sort_order)
SELECT *
FROM (
  VALUES
    ('Alur layanan', 'Apa saja tahapan layanan?', 'Jawab alur layanan dari knowledge resmi dan jangan mengarang jadwal.', 10),
    ('Update permintaan', 'Kalau sudah menghubungi admin, kapan dapat update?', 'Arahkan ke proses update layanan dan kontak tim operasional bila perlu.', 20),
    ('Layanan aktif', 'Layanan apa yang sedang tersedia?', 'Jawab berdasarkan knowledge layanan yang tersedia dan link resmi.', 30)
) AS seed(label, question, expected_behavior, sort_order)
WHERE NOT EXISTS (SELECT 1 FROM playground_shortcuts);
