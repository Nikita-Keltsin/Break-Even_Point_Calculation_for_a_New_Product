# Лабораторная работа №4 — авторизация, Redis и Swagger (breakeven_point)
**Автор:** Кельцин Никита Максимович, ИУ5-51Б

**Тема:** расчёт точки безубыточности нового продукта. Услуга — **вид затрат** (постоянные / переменные).

**Цель работы:** завершение бэкенда для SPA.

**Порядок показа:** **Скриншоты 1–3** — выполнить метод аутентификации через Swagger в режиме инкогнито, получить список услуг в Swagger; JWT из ответа аутентификации вручную указать в заголовке Authorization остальных запросов в Postman (белая галочка). **Скриншоты 4–5** — в Postman выполнить получение всех услуг: работает независимо от токена. **Скриншоты 6–8** — выполнить PUT публикации услуги: для гостя 401/403, для авторизованного — успех и обновление полей; показать содержимое Redis через cmd. **Скриншот 9** — в StarUML свойство signature у сообщений переиспользовано из диаграммы классов.

**Контрольные вопросы:** куки, сессия, Redis, JWT, авторизация и аутентификация, SSO, двухфакторная аутентификация, RSA.

**Sequence-диаграмма:** в StarUML (проект `breakeven_point.mdj`, «Диаграмма последовательности»): аутентификация, GET списка услуг с фильтром, добавление услуги с картинкой и видео, поля черновика, публикация, лента без id, лента по id с next=true, лайк. Lifeline — домены users и breakeven_point, у сообщений signature выбран из методов доменов на диаграмме классов, ключевые входные и выходные данные — в arguments.

**Задание:** добавление авторизации и Swagger в веб-сервис, подготовка ТЗ.

## Запуск

```bash
docker compose up -d            # postgres :5433, adminer :8081, minio :9000/9001, redis :6379
# новая БД: Adminer → SQL-запрос → migrations/001_schema.sql, затем 002_seed.sql
# БД уже есть с ЛР-3: выполнить только migrations/004_bcrypt.sql (пароли → bcrypt)
go get github.com/golang-jwt/jwt/v5 github.com/redis/go-redis/v9 golang.org/x/crypto/bcrypt \
       github.com/swaggo/gin-swagger github.com/swaggo/files github.com/swaggo/swag
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/Break-Even_Point_Calculation_for_a_New_Product/main.go --parseInternal -o docs
go mod tidy
go run ./cmd/Break-Even_Point_Calculation_for_a_New_Product   # Swagger: http://localhost:8080/swagger/index.html
```

Тестовые пользователи: `ivanov`, `petrov`, `sidorova`, пароль у всех `123456` (bcrypt). Черновик (id = 6) есть у `sidorova`.

## Авторизация

- `POST /api/users/login` проверяет пароль (bcrypt) и выдаёт JWT (HS256, срок 1 час): `{access_token, token_type: "Bearer", expires_in}`.
- Токен передаётся в заголовке `Authorization: Bearer <access_token>`.
- `POST /api/users/logout` кладёт в Redis ключ `blacklist:<jti>` (jti — короткий ID токена из его payload, значение — id пользователя) до конца срока действия токена; такой токен больше не принимается.
- Функция-singleton из ЛР-3 удалена: пользователь берётся из JWT (`currentUserID(ctx)`), при создании вида затрат он автоматически записывается в `creator_id`.
- Пароль при регистрации хранится bcrypt-хэшем.

## Методы и доступ

| № | Метод | URL | Гость | Создатель |
|---|---|---|---|---|
| 1 | GET | `/api/breakeven_point?min_amount=&max_amount=&is_fixed=0\|1` | да | да (+ is_mine) |
| 2 | GET | `/api/breakeven_point/feed`, `?id=1&next=true`, `/feed/:id` | да | да (+ is_liked) |
| 3 | GET | `/api/breakeven_point/draft` | 401 | да |
| 4 | POST | `/api/breakeven_point` (multipart: name, image, video) | 401 | да |
| 5 | PUT | `/api/breakeven_point/:id/publish` | 401 | да (только свой черновик, иначе 403) |
| 6 | DELETE | `/api/breakeven_point/:id` | 401 | да (только своя услуга) |
| 7 | POST | `/api/breakeven_point/:id/like` | 401 | да |
| 8 | POST | `/api/users/register` | да | да |
| 9 | POST | `/api/users/login` | да | да |
| 10 | POST | `/api/users/logout` | 401 | да |

В Swagger без кнопки **Authorize** работает только чтение; методы создателя помечены замком (`@Security BearerAuth`).

Коллекция Postman: `breakeven_point.postman_collection.json`. Токен при тестировании указывается **вручную в заголовке каждого запроса**: вкладка Headers → `Authorization: Bearer <JWT из ответа POST /api/users/login>` (белая галочка — заголовок отправляется, снятая — запрос гостя). Вкладка Authorization (No Auth) и меню Cookies не используются.

Redis: `bash redis_blacklist.sh` (или `docker exec -it redis_store redis-cli -a redispassword` → `KEYS blacklist:*`, `GET blacklist:<jti>`, `TTL blacklist:<jti>`).

## Таблицы БД

**users**: `id` SERIAL PK · `username` VARCHAR(50) NOT NULL UNIQUE · `password_hash` VARCHAR(100) NOT NULL (bcrypt) · `email` VARCHAR(100) NOT NULL · `role` VARCHAR(20) NOT NULL

**breakeven_point**: `id` SERIAL PK · `cost_name` VARCHAR(100) NOT NULL · `short_description` VARCHAR(255) NULL · `status` VARCHAR(20) NOT NULL (draft/published/deleted) · `image_url` VARCHAR(255) NOT NULL · `video_url` VARCHAR(255) NOT NULL · `amount_monthly` NUMERIC(12,2) NULL · `is_fixed` BOOLEAN NULL · `created_at` TIMESTAMP NOT NULL · `creator_id` INT NOT NULL FK → users.id · `formed_at` TIMESTAMP NULL

**likes** (м-м): `id` SERIAL PK · `user_id` INT FK → users.id · `cost_id` INT FK → breakeven_point.id · UNIQUE(user_id, cost_id)

Необязательных полей 4 — поля публикации `short_description`, `amount_monthly`, `is_fixed`, `formed_at`: у черновика они NULL и заполняются при публикации. Остальные столбцы NOT NULL. Все внешние ключи — ON DELETE RESTRICT.

## Дополнительное задание по индексам

После запуска инфраструктуры и применения миграций 001–004 выполните в учебной базе:

```bash
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d breakeven_db < migrations/005_catalog_indexes.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d breakeven_db < demo/seed.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d breakeven_db < demo/index-comparison.sql
```

`GET /api/breakeven_point/catalog` принимает `page`, `page_size`, `min_amount`, `max_amount` и `is_fixed`. Размер страницы по умолчанию 24, максимум 100. Ответ содержит `items`, `total`, `page` и `page_size`. Планы сравниваются в транзакции с откатом удаления индексов.

`GET /api/breakeven_point/feed/session` создаёт порцию из 100 уникальных ID при достаточном количестве карточек. При исчерпании порции Redis сохраняет новую. Для проверки ключей используйте `redis-cli --scan --pattern 'feed:*'`. Для демонстрации в React используйте ветку `extra-db-indexes` фронтенда.

## Дополнительное задание по асинхронному повороту

Примените `migrations/006_model_rotation.sql` после миграций ветки `extra-db-indexes`. Запустите сервис поворота NestJS на порту 3100. Для совместного запуска укажите одинаковый `ROTATION_KEY` у Go и NestJS и при необходимости задайте `ROTATION_URL` для Go.

`GET /api/breakeven_point/feed?next=true` ставит поворот в очередь. Через 5–10 с NestJS отправляет новые углы и прочитанную версию в `PUT /internal/costs/{id}/rotation`. Этот адрес предназначен для внутреннего доступа. Клиент читает результат через `GET /api/breakeven_point/feed/{id}`.
