-- Убираем таблицы прошлой итерации лабы (иначе их останется 5)
DROP TABLE IF EXISTS request_cost_links;
DROP TABLE IF EXISTS bep_requests;

DROP TABLE IF EXISTS likes, cost_types, users;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'creator'
);

CREATE TABLE cost_types (
    id SERIAL PRIMARY KEY,
    cost_name VARCHAR(100) NOT NULL,
    short_description VARCHAR(255),          
    full_description TEXT,                   
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    image_url VARCHAR(255),                  
    video_url VARCHAR(255),                  
    amount_monthly NUMERIC(12,2),            
    cost_kind VARCHAR(20),                   
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    creator_id INT NOT NULL REFERENCES users(id),
    formed_at TIMESTAMP,                    
    CONSTRAINT chk_status CHECK (status IN ('draft','published','deleted'))
);


CREATE UNIQUE INDEX idx_one_draft_per_user ON cost_types(creator_id) WHERE status = 'draft';

CREATE TABLE likes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    cost_id INT NOT NULL REFERENCES cost_types(id),
    CONSTRAINT unique_user_cost UNIQUE (user_id, cost_id)
);