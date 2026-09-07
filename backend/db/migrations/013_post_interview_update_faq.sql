DO $$
DECLARE
  faq_question text := 'Kalau sudah menghubungi admin, kapan dapat update layanan?';
  faq_answer text := 'Update layanan disampaikan melalui WhatsApp oleh tim operasional. Belum ada estimasi waktu khusus yang dicantumkan untuk setiap permintaan. Pelanggan dapat menunggu informasi melalui WhatsApp; jika perlu menanyakan status layanan, konfirmasi tahapan, atau informasi lanjutan, hubungi nomor resmi operasional di +62 823-1319-1024.';
BEGIN
  IF EXISTS (
    SELECT 1
    FROM knowledge_faqs
    WHERE lower(question) = lower(faq_question)
  ) THEN
    UPDATE knowledge_faqs
    SET answer = faq_answer,
        status = 'published',
        source_type = 'faq',
        priority = 99,
        locked = true,
        topic = 'service_update',
        intent = 'post_contact_update',
        metadata_json = jsonb_build_object(
          'aliases',
          jsonb_build_array(
            'kapan update setelah menghubungi admin',
            'kapan dapat kabar layanan',
            'kapan dapat update dari operasional',
            'status layanan saya'
          )
        ),
        updated_at = now(),
        published_at = COALESCE(published_at, now())
    WHERE lower(question) = lower(faq_question);
  ELSE
    INSERT INTO knowledge_faqs (
      question,
      answer,
      status,
      source_type,
      priority,
      locked,
      topic,
      intent,
      metadata_json,
      published_at
    )
    VALUES (
      faq_question,
      faq_answer,
      'published',
      'faq',
      99,
      true,
      'service_update',
      'post_contact_update',
      jsonb_build_object(
        'aliases',
        jsonb_build_array(
          'kapan update setelah menghubungi admin',
          'kapan dapat kabar layanan',
          'kapan dapat update dari operasional',
          'status layanan saya'
        )
      ),
      now()
    );
  END IF;
END $$;
