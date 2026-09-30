INSERT INTO users (username, email, role) VALUES
('ivanov','ivanov@company.ru','creator'),
('petrov','petrov@company.ru','moderator'),
('sidorova','sidorova@company.ru','creator');

INSERT INTO cost_types (cost_name, short_description, full_description, status, image_url, video_url, amount_monthly, cost_kind, creator_id, formed_at) VALUES
('Операционные расходы (OPEX)','Постоянные расходы недвижимости и офиса','Аренда переговорных и представительского офиса, коммунальные платежи, клининг и охрана; регламент пересмотра ставок ежеквартальный.','published','http://localhost:9000/media/screen.png','http://localhost:9000/media/7426703-hd_1080_1920_25fps.mp4',2850000,'fixed',1,NOW()),
('Фонд оплаты труда (ФОТ)','Оклады управленческой команды','Оклады, страховые взносы и ДМС управленческой команды; не зависит от выручки, индексируется раз в год.','published','http://localhost:9000/media/screen_2.png','http://localhost:9000/media/7652821-hd_1080_1920_25fps.mp4',4620000,'fixed',1,NOW()),
('Себестоимость продаж (COGS)','Переменные затраты производства','Сырьё, упаковка, склад и магистральная доставка до РЦ; норматив пересчитывается еженедельно.','published','http://localhost:9000/media/screen_3.png','http://localhost:9000/media/7818630-hd_1080_1920_30fps.mp4',7410000,'variable',1,NOW()),
('Маркетинг и аналитика','Переменные расходы на продвижение','Медиабаинг, сквозная аналитика и A/B-платформы; планируются от процента выручки.','deleted','http://localhost:9000/media/screen_4.png','http://localhost:9000/media/7643847-uhd_2160_4096_25fps.mp4',1150000,'variable',1,NOW()),
('Логистика последней мили','Переменные расходы доставки','Курьерские сервисы, эквайринг доставки, компенсации возвратов; планируются от числа заказов.','published',NULL,NULL,75000,'variable',1,NOW());

INSERT INTO likes (user_id, cost_id) VALUES (1,1),(2,1),(3,1),(1,2),(2,3);