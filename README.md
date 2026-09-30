# Лабораторная 3 — веб-сервис REST API (тема: расчёт точки безубыточности)

## Домен «Услуга» (вид затрат)
- GET /api/costs — список опубликованных услуг; необязательные query-фильтры min_amount, max_amount (диапазон суммы), kind=fixed|variable. Ответ 200: массив объектов с is_mine и is_liked.
- GET /api/costs/feed — лента: первая опубликованная услуга, id не указывается.
- GET /api/costs/feed/:id — опубликованная услуга по id; 404, если не существует или удалена.
- GET /api/costs/feed/:id?next=true — следующая опубликованная после id, в конце ленты заворот на первую.
- GET /api/costs/draft — единственный черновик текущего пользователя, id с клиента не передаётся; 404, если черновика нет.
- POST /api/costs — создание черновика; принимает name (текст) и файлы image, video (multipart/form-data, именно файлы, а не URL). Файлы сохраняются в MinIO под сгенерированными латинскими именами, имена идут в поля БД. Ответ 201 + объект черновика; 409, если черновик уже есть.
- PUT /api/costs/:id/publish — публикация своего черновика (смена статуса draft→published) + доп. поля; JSON-тело: short_description, full_description, amount_monthly, cost_kind. Ответ 200 + обновлённый объект; 404 чужой/несуществующий, 403 если не черновик. Вернуть в черновик нельзя.
- DELETE /api/costs/:id — логическое удаление своей услуги (status='deleted', SQL UPDATE через курсор без ORM). Ответ 200; чужая услуга — 403.
- POST /api/costs/:id/like — лайк текущего пользователя; JSON {"like":1} ставит, {"like":0} отменяет. Ответ 200.

## Домен «Пользователь»
- POST /api/users/register — регистрация; JSON {username, password}. Ответ 201 + {id, username} (пароль не сериализуется); 409, если логин занят.
- POST /api/users/login — заглушка аутентификации для ЛР4. Ответ 200.
- POST /api/users/logout — заглушка деавторизации для ЛР4. Ответ 200.

## Соглашения ответов
- Набор полей JSON всегда одинаковый; незаполненные значения приходят как null/"".
- Статус услуги в ответах не передаётся (хранится только в БД); коды состояния в тело не дублируются.
- Успех: 200/201; ошибки: 400, 403, 404, 409 без текстовых сообщений.
- Пользователь-создатель зафиксирован константой через функцию-singleton CurrentUserID().

## Таблицы базы данных
users — зарегистрированные пользователи
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| username | VARCHAR(50), UNIQUE, NOT NULL | имя |
| password_hash | VARCHAR(100), NOT NULL | хэш пароля (ЛР4) |
| email | VARCHAR(100) | почта |
| role | VARCHAR(20), NOT NULL | роль |

cost_types — черновики, опубликованные и удалённые услуги
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| status | VARCHAR(20), NOT NULL, DEFAULT 'draft' | draft/published/deleted (только в БД) |
| cost_name | VARCHAR(100), NOT NULL | название |
| short_description | VARCHAR(255), NULL | краткое описание |
| full_description | TEXT, NULL | полное описание |
| image_url | VARCHAR(255), NULL | имя файла изображения в MinIO |
| video_url | VARCHAR(255), NULL | имя файла видео в MinIO |
| amount_monthly | NUMERIC(12,2), NULL | поле по теме 1 |
| cost_kind | VARCHAR(20), NULL | поле по теме 2 |
| creator_id | INTEGER, NOT NULL, FK → users.id | создатель (ON DELETE RESTRICT) |
| created_at | TIMESTAMP | дата создания |
| formed_at | TIMESTAMP, NULL | дата формирования/публикации |

likes — связь многие-ко-многим пользователи→услуги
| поле | тип | описание |
|---|---|---|
| id | SERIAL, PK | первичный ключ |
| user_id | INTEGER, FK → users.id | кто лайкнул (ON DELETE RESTRICT) |
| cost_id | INTEGER, FK → cost_types.id | что лайкнули (ON DELETE RESTRICT) |
| — | UNIQUE(user_id, cost_id) | один лайк на пару |