UPDATE knowledge_documents
SET file_url = CASE
      WHEN file_url LIKE 'urus-in/%' THEN 'oneflow/' || SUBSTRING(file_url FROM LENGTH('urus-in/') + 1)
      ELSE REPLACE(file_url, '/urus-in/', '/oneflow/')
    END,
    updated_at = NOW()
WHERE file_url LIKE 'urus-in/%'
   OR file_url LIKE '%/urus-in/%';
