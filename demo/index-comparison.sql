\timing on
\echo 'With the index'
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM breakeven_point
 WHERE status = 'published' AND is_fixed = true AND amount_monthly BETWEEN 900000 AND 901000
 ORDER BY amount_monthly, id LIMIT 24 OFFSET 0;
BEGIN;
DROP INDEX idx_costs_published_kind_amount, idx_costs_published_amount;
\echo 'Without the indexes (transaction is rolled back)'
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM breakeven_point
 WHERE status = 'published' AND is_fixed = true AND amount_monthly BETWEEN 900000 AND 901000
 ORDER BY amount_monthly, id LIMIT 24 OFFSET 0;
ROLLBACK;
