CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'creator',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE cost_types (
    id SERIAL PRIMARY KEY,
    cost_name VARCHAR(100) NOT NULL,
    short_description TEXT,
    full_description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    image_url VARCHAR(255),
    video_url VARCHAR(255),
    cost_kind VARCHAR(20) NOT NULL DEFAULT 'fixed',
    amount_monthly NUMERIC(12,2) NOT NULL,
    cost_share INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE bep_requests (
    id SERIAL PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    creator_id INT NOT NULL REFERENCES users(id),
    formed_at TIMESTAMP,
    completed_at TIMESTAMP,
    moderator_id INT REFERENCES users(id),
    product_name VARCHAR(100),
    selling_price NUMERIC(10,2),
    bep_units INT,
    bep_revenue NUMERIC(12,2),
    CONSTRAINT chk_status CHECK (status IN ('draft','deleted','formed','completed','rejected'))
);

-- не более одного черновика у пользователя
CREATE UNIQUE INDEX idx_one_draft_per_user ON bep_requests(creator_id) WHERE status = 'draft';

-- м-м: составной уникальный ключ, БЕЗ каскадного удаления
CREATE TABLE request_cost_links (
    request_id INT NOT NULL REFERENCES bep_requests(id),
    cost_id INT NOT NULL REFERENCES cost_types(id),
    volume INT NOT NULL DEFAULT 1,
    is_critical BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    comment TEXT,
    PRIMARY KEY (request_id, cost_id)
);