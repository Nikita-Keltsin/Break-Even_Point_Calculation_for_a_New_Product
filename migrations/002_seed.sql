-- breakeven_point: тестовые данные
-- пароль у всех пользователей: 123456 (sha256)
INSERT INTO users (username, password_hash, email, role) VALUES
('ivanov',   '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'ivanov@company.ru',   'creator'),
('petrov',   '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'petrov@company.ru',   'moderator'),
('sidorova', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'sidorova@company.ru', 'creator');

-- обязательно есть все три статуса: опубликован, удалён, черновик
INSERT INTO breakeven_types (cost_name, short_description, status, image_url, video_url, amount_monthly, is_fixed, creator_id, formed_at) VALUES
('Операционные расходы (OPEX)', 'Аренда офиса, коммунальные платежи, клининг и охрана. Не зависят от объёма выпуска.',
 'published', 'http://localhost:9000/media/screen.png',   'http://localhost:9000/media/7426703-hd_1080_1920_25fps.mp4', 2850000, TRUE,  1, NOW()),
('Фонд оплаты труда (ФОТ)',     'Оклады, страховые взносы и ДМС управленческой команды; индексируются раз в год.',
 'published', 'http://localhost:9000/media/screen_2.png', 'http://localhost:9000/media/7652821-hd_1080_1920_25fps.mp4', 4620000, TRUE,  1, NOW()),
('Себестоимость продаж (COGS)', 'Сырьё, упаковка, склад и доставка до РЦ; растут пропорционально объёму продаж.',
 'published', 'http://localhost:9000/media/screen_3.png', 'http://localhost:9000/media/7818630-hd_1080_1920_30fps.mp4', 7410000, FALSE, 1, NOW()),
('Маркетинг и аналитика',       'Медиабаинг и сквозная аналитика; планируются как процент от выручки.',
 'deleted',   'http://localhost:9000/media/screen_4.png', 'http://localhost:9000/media/7643847-uhd_2160_4096_25fps.mp4', 1150000, FALSE, 1, NOW()),
('Логистика последней мили',    'Курьерские сервисы и компенсации возвратов; зависят от числа заказов.',
 'published', '', '', 75000, FALSE, 1, NOW()),
-- черновик другого пользователя (sidorova): у ivanov черновика нет, его создаём в демонстрации
('Комиссия эквайринга',         NULL,
 'draft',     '', '', NULL, FALSE, 3, NULL);

INSERT INTO likes (user_id, cost_id) VALUES (1,1),(2,1),(3,1),(1,2),(2,3);
