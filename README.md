# Лабораторная работа №2 — база данных PostgreSQL (breakeven_point)
**Автор:** Кельцин Никита Максимович, ИУ5-51Б

**Тема:** расчёт точки безубыточности нового продукта. Услуга — **вид затрат** (постоянные: аренда, оклады; переменные: сырьё, сдельная зарплата).

**Цель работы:** разработка структуры базы данных и её подключение к бэкенду.

**Порядок показа:** **Скриншоты 1–2** — показать панель администратора Adminer, в которой логически удалить через статус услугу, посмотреть данные через select в БД. **Скриншоты 3–10** — показать 3 страницы приложения: выполнить поиск, удалить услугу, перейти по url удалённой услуги, добавить услугу, показать select в БД, опубликовать карточку, снова показать select в БД. **Скриншоты 11–13** — в БД изменить поля по теме в услуге и количество строк в м-м и показать их в приложении. **Скриншоты 14–21** — в коде показать модели, 5 контроллеров через ORM, а удаление услуги через SQL UPDATE. **Скриншот 22** — фото и видео по умолчанию в HTML.

**Контрольные вопросы:** виды БД, SQL-запросы, курсоры, ORM, модель и миграции, чистая архитектура.

**ER-диаграмма:** сделана в StarUML (проект `breakeven_point.mdj`): таблицы, связи, столбцы, типы столбцов и их длина, первичные и внешние ключи.

**Задание:** создание базы данных PostgreSQL по теме работы, подключение БД к созданным трём шаблонам.

## Запуск

```bash
docker compose up -d   # postgres :5433, adminer :8081, minio :9000/9001
# Adminer → http://localhost:8081 (PostgreSQL, сервер postgres, логин/пароль postgres, БД breakeven_db)
#   → «SQL-запрос» → выполнить migrations/001_schema.sql, затем migrations/002_seed.sql
go run ./cmd/Break-Even_Point_Calculation_for_a_New_Product   # http://localhost:8080/breakeven-point/cost-types
```

## HTTP-методы (ровно 6)

| № | Метод | URL | Что делает | Доступ к БД |
|---|---|---|---|---|
| 1 | GET | `/breakeven-point/cost-types` | плитка видов затрат + поиск: сумма от/до, галочки «Постоянные / Переменные» | ORM, фильтр в SQL WHERE |
| 2 | GET | `/breakeven-point/feed`, `/breakeven-point/feed/:id` | лента: из БД возвращается ровно одна строка | ORM (`First`) |
| 3 | GET | `/breakeven-point/addition` | добавление: шаг 1 (название, фото, видео, «Далее») или шаг 2 (черновик, «Опубликовать») | ORM |
| 4 | POST | `/breakeven-point/draft` | «Далее» — создать черновик | ORM |
| 5 | POST | `/breakeven-point/publish` | «Опубликовать» — одно краткое описание + 2 поля по теме, статус → published | ORM |
| 6 | POST | `/breakeven-point/delete` | логическое удаление с плитки, статус → deleted | SQL `UPDATE` (без ORM) |

- Черновик создаётся только кнопкой «Далее»; если черновик уже есть, страница добавления открывается с заполненными полями и кнопкой «Опубликовать».
- Описание вводится только при публикации, и оно одно (`short_description`). Поля по теме: `amount_monthly` (сумма затрат в месяц) и `is_fixed` (галочка «Постоянные затраты»).
- Новые фото и видео в ЛР-2 на сервер не передаются. Если url пустой или файл недоступен, показываются фото и видео по умолчанию из `resources/media` (SSR-сервер).
- Удалённые услуги просматривать нельзя. Лайки только отображаются из БД.

## Таблицы БД

**users** — пользователи: `id` SERIAL PK · `username` VARCHAR(50) NOT NULL UNIQUE · `password_hash` VARCHAR(100) NOT NULL · `email` VARCHAR(100) NOT NULL · `role` VARCHAR(20) NOT NULL

**breakeven_point** — услуги (виды затрат): `id` SERIAL PK · `cost_name` VARCHAR(100) NOT NULL · `short_description` VARCHAR(255) NULL · `status` VARCHAR(20) NOT NULL (draft / published / deleted) · `image_url` VARCHAR(255) NOT NULL · `video_url` VARCHAR(255) NOT NULL · `amount_monthly` NUMERIC(12,2) NULL · `is_fixed` BOOLEAN NULL · `created_at` TIMESTAMP NOT NULL · `creator_id` INT NOT NULL FK → users.id · `formed_at` TIMESTAMP NULL

**likes** — лайки, м-м пользователь — услуга: `id` SERIAL PK · `user_id` INT FK → users.id · `cost_id` INT FK → breakeven_point.id · UNIQUE(user_id, cost_id)

Необязательных полей 4 — поля публикации `short_description`, `amount_monthly`, `is_fixed`, `formed_at`: у черновика они NULL и заполняются при публикации. Остальные столбцы NOT NULL. Каскадное удаление запрещено (все FK — ON DELETE RESTRICT). В тестовых данных есть услуги во всех трёх статусах. У пользователя не более одного черновика (частичный уникальный индекс).
