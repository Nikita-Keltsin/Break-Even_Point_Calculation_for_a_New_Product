# Лабораторная работа №3 — веб-сервис REST API (breakeven_point)
**Автор:** Кельцин Никита Максимович, ИУ5-51Б

**Тема:** расчёт точки безубыточности нового продукта. Услуга — **вид затрат** (постоянные / переменные).

**Цель работы:** создание веб-сервиса в бэкенде нашей системы для использования его в SPA.

**Порядок показа:** **Скриншоты 1–10** — через Postman показать коллекцию 10 запросов, выполнить GET списка услуг (с фильтром), добавить новую услугу с картинкой и видео, получить поля черновика, опубликовать услугу, получить ленту без id, получить ленту по id ?next=true, поставить лайк, удалить услугу, зарегистрировать нового пользователя. **Скриншоты 11–13** — показать изменённые данные через select. **Скриншоты 14–16** — объяснить модели, сериализаторы; показать функцию-singleton, где пользователь указан константой, и использование этой функции в методах.

**Контрольные вопросы:** веб-сервис, REST, RPC, заголовки и методы HTTP, версии HTTP, HTTPS, OSI ISO.

**Диаграмма классов:** в StarUML (проект `breakeven_point.mdj`): 4 страницы фронтенда, интерфейсы API users и API breakeven_point, домены со всеми методами по url, модели User, Like, BreakEvenPoint и таблицы БД.

**Задание:** создание веб-сервиса со всей итоговой бизнес-логикой (кроме авторизации), подключение его к БД и тестирование в Postman.

## Запуск

```bash
docker compose up -d   # postgres :5433, adminer :8081, minio :9000 (консоль :9001)
# Adminer → SQL-запрос → migrations/001_schema.sql, затем migrations/002_seed.sql
go run ./cmd/Break-Even_Point_Calculation_for_a_New_Product   # http://localhost:8080
# Postman → Import → breakeven_point.postman_collection.json
```

## Методы веб-сервиса (10)

### Домен breakeven_point — `/api/breakeven_point`
1. `GET /api/breakeven_point?min_amount=&max_amount=&is_fixed=0|1` — список опубликованных видов затрат, фильтрация на бэкенде (SQL WHERE).
2. `GET /api/breakeven_point/feed` — лента без id (первая опубликованная); `GET /api/breakeven_point/feed/:id` — по id; `GET /api/breakeven_point/feed?id=1&next=true` — следующая после id (в конце — снова первая). Из БД читается одна строка.
3. `GET /api/breakeven_point/draft` — черновик текущего пользователя (id не передаётся); 404, если черновика нет.
4. `POST /api/breakeven_point` — multipart/form-data: `name`, файлы `image`, `video`. Файлы → MinIO (бакет `break-even`) под латинскими именами `cost_<uid>_<image|video>_<время>.<ext>`, имена → `image_url` / `video_url`. 201; 409, если черновик уже есть.
5. `PUT /api/breakeven_point/:id/publish` — JSON `{short_description, amount_monthly, is_fixed}`; только свой черновик, draft → published, заполняется `formed_at`. Вернуть в черновик нельзя.
6. `DELETE /api/breakeven_point/:id` — только soft delete своей услуги (status = deleted).
7. `POST /api/breakeven_point/:id/like` — `{"like":1}` ставит лайк текущего пользователя, `{"like":0}` отменяет.

### Домен users — `/api/users`
8. `POST /api/users/register` — `{username, password}` → 201 `{id, username}`; 409, если логин занят.
9. `POST /api/users/login` — аутентификация, заглушка для ЛР-4.
10. `POST /api/users/logout` — деавторизация, заглушка для ЛР-4.

Ответ по услуге: `id, cost_name, short_description, image_url, video_url, amount_monthly, is_fixed, creator_id, created_at, formed_at, likes_count`. Список дополнительно отдаёт `is_mine` (0/1 — своя карточка, для кнопки «Удалить»), лента — `is_liked` (0/1 — поставил ли текущий пользователь лайк). Ответ на лайк: `{is_liked, likes_count}`.

- Статус на клиент не передаётся, записи в статусе deleted не отдаются.
- Системные поля (id, status, creator_id, даты) с клиента не принимаются — вычисляются на бэкенде.
- Пользователь-создатель до ЛР-4 зафиксирован константой через функцию-singleton `CurrentUserID()` (`internal/api/handlers.go`).
- SSR-страницы из ЛР-2 (`/breakeven-point/...`) сохранены.

## Таблицы БД

**users**: `id` SERIAL PK · `username` VARCHAR(50) NOT NULL UNIQUE · `password_hash` VARCHAR(100) NOT NULL · `email` VARCHAR(100) NOT NULL · `role` VARCHAR(20) NOT NULL

**breakeven_point** (услуги — виды затрат): `id` SERIAL PK · `cost_name` VARCHAR(100) NOT NULL · `short_description` VARCHAR(255) NULL · `status` VARCHAR(20) NOT NULL (draft/published/deleted) · `image_url` VARCHAR(255) NOT NULL · `video_url` VARCHAR(255) NOT NULL · `amount_monthly` NUMERIC(12,2) NULL (поле по теме 1) · `is_fixed` BOOLEAN NULL (поле по теме 2) · `created_at` TIMESTAMP NOT NULL · `creator_id` INT NOT NULL FK → users.id · `formed_at` TIMESTAMP NULL

**likes** (м-м): `id` SERIAL PK · `user_id` INT FK → users.id · `cost_id` INT FK → breakeven_point.id · UNIQUE(user_id, cost_id)

Необязательных полей 4 — поля публикации `short_description`, `amount_monthly`, `is_fixed`, `formed_at`: у черновика они NULL и заполняются при публикации. Остальные столбцы NOT NULL. Все внешние ключи — ON DELETE RESTRICT (каскадное удаление запрещено). У пользователя не более одного черновика (частичный уникальный индекс).
