UPDATE knowledge_faqs
SET answer = regexp_replace(
    answer,
    '^\s*\([^)]*jangan ubah jawaban/format[^)]*\)\s*',
    '',
    'i'
)
WHERE answer ~* '^\s*\([^)]*jangan ubah jawaban/format[^)]*\)';
