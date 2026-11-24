-- Run only against the lab database. Repeated runs insert no extra demo rows.
BEGIN;
SELECT pg_advisory_xact_lock(810020);
INSERT INTO breakeven_point (cost_name, short_description, status, amount_monthly, is_fixed, creator_id, formed_at)
SELECT (ARRAY['Аренда площадки', 'Поставка сырья', 'Амортизация', 'Упаковка', 'Обслуживание склада', 'Доставка'])[(n - 1) % 6 + 1]
       || ' · ' || lpad(n::text, 6, '0'),
       'Демонстрационная запись для проверки фильтрации и пагинации.',
       'published', (n * 7919::bigint % 7995000) + 5000, n % 2 = 1, analyst.id, now()
FROM generate_series(1, 120000) AS n
CROSS JOIN (SELECT id FROM users WHERE username = 'ivanov') AS analyst
WHERE NOT EXISTS (SELECT 1 FROM breakeven_point WHERE short_description = 'Демонстрационная запись для проверки фильтрации и пагинации.');
COMMIT;
ANALYZE breakeven_point;
SELECT count(*) AS published_costs FROM breakeven_point WHERE status = 'published';
