# breakeven_point — расчёт точки безубыточности нового продукта

Автор: Кельцин Никита, ИУ5-51Б

Услуга = **вид затрат** (постоянные / переменные), из которых складывается точка безубыточности.
Стек: Go (Gin, GORM), PostgreSQL, MinIO, Adminer, Docker Compose.

```
docker compose up -d          # postgres:5433, adminer:8081, minio:9000/9001
psql ... -f migrations/001_schema.sql -f migrations/002_seed.sql   # или через Adminer → SQL-запрос
go run ./cmd/Break-Even_Point_Calculation_for_a_New_Product        # http://localhost:8080
```

## ЛР-2 — SSR-страницы (6 HTTP-методов)

| Метод | URL | Что делает | Доступ к БД |
|---|---|---|---|
| GET | /breakeven-point/cost-types | плитка услуг + поиск (сумма от/до, постоянные/переменные) | ORM |
| GET | /breakeven-point/feed, /breakeven-point/feed/:id | лента, из БД читается одна строка | ORM |
| GET | /breakeven-point/addition | добавление: шаг 1 (название, фото, видео, «Далее») или шаг 2 (черновик, «Опубликовать») | ORM |
| POST | /breakeven-point/draft | «Далее» — создать черновик | ORM |
| POST | /breakeven-point/publish | «Опубликовать» — краткое описание + 2 поля по теме, статус → published | ORM |
| POST | /breakeven-point/delete | логическое удаление, статус → deleted | SQL UPDATE (без ORM) |

## ЛР-3 — веб-сервис REST API (10 методов)

### Домен breakeven_point `/api/breakeven_point`
1. `GET /api/breakeven_point?min_amount=&max_amount=&is_fixed=0|1` — список опубликованных, фильтрация на бэкенде.
2. `GET /api/breakeven_point/feed` — лента без id (первая опубликованная); `GET /api/breakeven_point/feed/:id` — по id; `GET /api/breakeven_point/feed?id=1&next=true` (или `/feed/1?next=true`) — следующая после id (в конце — снова первая). Из БД читается одна строка.
3. `GET /api/breakeven_point/draft` — черновик текущего пользователя (id не передаётся), 404 если нет.
4. `POST /api/breakeven_point` — multipart/form-data: `name`, файлы `image`, `video`. Файлы → MinIO под латинскими именами `cost_<uid>_<image|video>_<время>.<ext>`, имена → `image_url`/`video_url`. 201; 409 если черновик уже есть.
5. `PUT /api/breakeven_point/:id/publish` — JSON `{short_description, amount_monthly, is_fixed}`; только свой черновик, draft → published. Вернуть в черновик нельзя.
6. `DELETE /api/breakeven_point/:id` — soft delete своей услуги (status = deleted).
7. `POST /api/breakeven_point/:id/like` — `{"like":1}` ставит, `{"like":0}` отменяет.

### Домен users `/api/users`
8. `POST /api/users/register` — `{username, password}` → 201 `{id, username}`; 409 если логин занят.
9. `POST /api/users/login` — заглушка (ЛР-4).
10. `POST /api/users/logout` — заглушка (ЛР-4).

Ответ по услуге: `id, cost_name, short_description, image_url, video_url, amount_monthly, is_fixed, creator_id, created_at, formed_at, likes_count, like (0/1 — лайк текущего пользователя), is_mine (0/1)`.
Статус на клиент не передаётся, удалённые записи не отдаются. Системные поля (id, status, creator_id, даты) с клиента не принимаются.
Пользователь-создатель до ЛР-4 зафиксирован константой через функцию-singleton `CurrentUserID()`.

## Таблицы БД

**users**: id SERIAL PK · username VARCHAR(50) NOT NULL UNIQUE · password_hash VARCHAR(100) NOT NULL · email VARCHAR(100) · role VARCHAR(20) NOT NULL

**breakeven_types**: id SERIAL PK · cost_name VARCHAR(100) NOT NULL · short_description VARCHAR(255) · status VARCHAR(20) NOT NULL (draft/published/deleted) · image_url VARCHAR(255) NOT NULL · video_url VARCHAR(255) NOT NULL · amount_monthly NUMERIC(12,2) (поле по теме 1) · is_fixed BOOLEAN NOT NULL (поле по теме 2) · created_at TIMESTAMP NOT NULL · creator_id INT NOT NULL FK → users.id · formed_at TIMESTAMP

**likes** (м-м): id SERIAL PK · user_id INT FK → users.id · cost_id INT FK → breakeven_types.id · UNIQUE(user_id, cost_id)

Все внешние ключи — ON DELETE RESTRICT (каскадное удаление запрещено). У пользователя не более одного черновика (частичный уникальный индекс).
