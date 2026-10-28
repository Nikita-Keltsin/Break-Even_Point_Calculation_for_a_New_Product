-- breakeven_point: схема БД (ЛР-2/ЛР-3)
-- Каскадное удаление запрещено: все внешние ключи ON DELETE RESTRICT.
-- 4 необязательных поля услуги (NULL у черновика, заполняются при публикации):
-- short_description, amount_monthly, is_fixed, formed_at.

-- таблицы прошлых итераций лабы
DROP TABLE IF EXISTS request_cost_links;
DROP TABLE IF EXISTS bep_requests;

DROP TABLE IF EXISTS likes, cost_types, breakeven_types, breakeven_point, users;  -- cost_types, breakeven_types — старые имена таблицы

-- Пользователи
CREATE TABLE users (
    id            SERIAL       PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(100) NOT NULL,                 -- хэш пароля (sha256, в ЛР-4 — bcrypt)
    email         VARCHAR(100) NOT NULL DEFAULT '',
    role          VARCHAR(20)  NOT NULL DEFAULT 'creator'
);

-- Услуги = виды затрат для расчёта точки безубыточности
CREATE TABLE breakeven_point (
    id                SERIAL        PRIMARY KEY,
    cost_name         VARCHAR(100)  NOT NULL,             -- наименование
    short_description VARCHAR(255),                       -- N: краткое описание (вводится при публикации)
    status            VARCHAR(20)   NOT NULL DEFAULT 'draft',
    image_url         VARCHAR(255)  NOT NULL DEFAULT '',  -- '' = фото по умолчанию с SSR
    video_url         VARCHAR(255)  NOT NULL DEFAULT '',  -- '' = видео по умолчанию с SSR
    amount_monthly    NUMERIC(12,2),                      -- N: поле по теме 1: сумма затрат, руб/мес
    is_fixed          BOOLEAN,                            -- N: поле по теме 2: постоянные затраты (галочка)
    created_at        TIMESTAMP     NOT NULL DEFAULT NOW(), -- дата создания
    creator_id        INT           NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    formed_at         TIMESTAMP,                          -- N: дата формирования (публикации)
    CONSTRAINT chk_status CHECK (status IN ('draft', 'published', 'deleted'))
);

-- у каждого пользователя не более одного черновика
CREATE UNIQUE INDEX idx_one_draft_per_user ON breakeven_point(creator_id) WHERE status = 'draft';

-- Лайки: м-м пользователь — услуга (первичный и два внешних ключа)
CREATE TABLE likes (
    id      SERIAL PRIMARY KEY,
    user_id INT    NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    cost_id INT    NOT NULL REFERENCES breakeven_point(id) ON DELETE RESTRICT,
    CONSTRAINT unique_user_cost UNIQUE (user_id, cost_id)
);
