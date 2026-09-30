# Лабораторная 2 — База данных PostgreSQL и подключение к бэкенду
## Тема: «Расчёт точки безубыточности для нового продукта» (виды затрат)

Стек: Go + Gin + GORM, PostgreSQL (docker, host-порт 5433), Adminer (:8081), MinIO (:9000, медиа лабы 1), дефолтные медиа — на SSR-сервере в `resources/media`.

## Страницы (3 шаблона, SSR)
- **Лента** (`feed.html`) — одна опубликованная услуга: БД возвращает сразу одну строку (First = LIMIT 1), без фильтрации в коде. «Еще» (?full=1) раскрывает полное описание, «Свернуть» сворачивает; лайк-сердце (красное, если текущий пользователь лайкнул, счёт из таблицы likes); «ДАЛЕЕ» — следующая опубликованная, в конце заворот на первую. Удалённые услуги не просматриваются (404).
- **Добавление** (`addition.html`) — фото и видео сверху, друг под другом, превью из директории media SSR-сервера, выбор из проводника (файлы НЕ отправляются на сервер и НЕ сохраняются в БД). Если у пользователя нет черновика — форма и кнопка «Далее»; если черновик есть — страница открывается с заполненными полями и кнопкой «Опубликовать».
- **Виды затрат / плитка** (`cost_types.html`) — только опубликованные; фильтрация на бэкенде: двойной ползунок диапазона суммы (amount_from/amount_to) и чекбоксы «Постоянные/Переменные»; карточки с фото (при пустом url или недоступном файле — заглушка с SSR через onerror), счётчиком лайков из БД и кнопкой «Удалить».

## HTTP-методы (все /breakeven-point/...)
- GET /breakeven-point/cost-types — плитка со списком опубликованных услуг и фильтрацией на бэкенде.
- GET /breakeven-point/feed и /breakeven-point/feed?id=N — лента (одна строка из БД), ?full=1 — полное описание.
- GET /breakeven-point/addition — страница добавления (черновик или пустая форма).
- POST /breakeven-point/draft — создание черновика через ORM (DB.Create); статус, создатель и даты ставит бэкенд; не более одного черновика у пользователя (частичный уникальный индекс).
- POST /breakeven-point/publish — публикация через ORM (DB.Updates): краткая/полная информация, amount_monthly, cost_kind, status='published', formed_at=NOW(); вернуть в черновик нельзя.
- POST /breakeven-point/delete — логическое удаление услуги: SQL UPDATE status='deleted' через database/sql, БЕЗ ORM.
- POST /breakeven-point/like — лайк текущего пользователя: INSERT/DELETE в таблицу likes (м-м); UNIQUE(user_id, cost_id) — один лайк на пару.

## Соглашения
- Дефолтные фото и видео лежат на SSR-сервере (`/static/media/default-image.jpg`, `/static/media/default-video.mp4`), НЕ в MinIO; подставляются сервером при пустых url и через onerror в HTML при недоступном файле.
- Статус услуги (draft/published/deleted) хранится только в БД; удалённые услуги не передаются на клиент и не просматриваются.
- Системные поля (id, статус, создатель, даты создания/формирования) вычисляются на бэкенде, с клиента не передаются.
- Пользователь-создатель зафиксирован константой через функцию-singleton CurrentUserID() = 1.
- Каскадное удаление запрещено: все FK без ON DELETE CASCADE.

## Таблицы базы данных

### users — пользователи
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| username | VARCHAR(50), UNIQUE, NOT NULL | имя пользователя |
| email | VARCHAR(100) | почта |
| role | VARCHAR(20), NOT NULL, DEFAULT 'creator' | роль |

### cost_types — черновики, опубликованные и удалённые услуги
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| cost_name | VARCHAR(100), NOT NULL | наименование услуги |
| short_description | VARCHAR(255), NULL | краткое описание |
| full_description | TEXT, NULL | полное описание |
| status | VARCHAR(20), NOT NULL, DEFAULT 'draft' | draft/published/deleted, CHECK-ограничение |
| image_url | VARCHAR(255), NULL | url/имя изображения (пусто → дефолт с SSR) |
| video_url | VARCHAR(255), NULL | url/имя видео (пусто → дефолт с SSR) |
| amount_monthly | NUMERIC(12,2), NULL | поле по теме 1: норматив, ₽/мес |
| cost_kind | VARCHAR(20), NULL | поле по теме 2: fixed/variable |
| created_at | TIMESTAMP, NOT NULL, DEFAULT NOW() | дата создания |
| creator_id | INTEGER, NOT NULL, FK → users.id | создатель (без каскада) |
| formed_at | TIMESTAMP, NULL | дата формирования/публикации |

Индекс: CREATE UNIQUE INDEX idx_one_draft_per_user ON cost_types(creator_id) WHERE status='draft' — не более одного черновика у пользователя.

### likes — связь многие-ко-многим пользователь→услуга
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| user_id | INTEGER, NOT NULL, FK → users.id | кто лайкнул (без каскада) |
| cost_id | INTEGER, NOT NULL, FK → cost_types.id | что лайкнули (без каскада) |
| — | UNIQUE(user_id, cost_id) | один лайк на пару |

## Миграции и наполнение
- `migrations/001_schema.sql` — CREATE TABLE users / cost_types / likes, CHECK статусов, уникальный индекс черновика.
- `migrations/002_seed.sql` — INSERT: 3 опубликованные услуги, 1 удалённая, 1 черновик; лайки.
- `migrations/003_api.sql` — ALTER TABLE users ADD password_hash (для ЛР3).
- Наполнение и правки также через Adminer (:8081, сервер `postgres`).

## Порядок показа (Adminer + приложение)
1. UPDATE cost_types SET status='deleted' WHERE id=3; + SELECT * FROM cost_types; — логическое удаление через статус.
2. Плитка: поиск/фильтр двойным ползунком; «Удалить» на карточке; переход по url удалённой → «не найдена».
3. Добавление: «Далее» → SELECT (draft) → заполнить → «Опубликовать» → SELECT (published).
4. UPDATE cost_types SET amount_monthly=3000000, cost_kind='fixed' WHERE id=2; и INSERT INTO likes (user_id, cost_id) VALUES (2,2),(3,2); → показать изменённые поля и счёт лайков в ленте/плитке.
5. В коде: модели (GORM-теги), 5 контроллеров через ORM, удаление услуги через SQL UPDATE (sqlDB.Exec).
