CREATE INDEX IF NOT EXISTS idx_costs_published_amount
    ON breakeven_point (amount_monthly, id) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_costs_published_kind_amount
    ON breakeven_point (is_fixed, amount_monthly, id) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_likes_cost ON likes (cost_id);
