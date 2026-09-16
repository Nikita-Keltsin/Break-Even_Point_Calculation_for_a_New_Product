INSERT INTO users (username, email, role) VALUES
('ivanov','ivanov@company.ru','creator'),
('petrov','petrov@company.ru','moderator');

INSERT INTO cost_types (cost_name, short_description, full_description, image_url, video_url, cost_kind, amount_monthly, cost_share) VALUES
('Операционные расходы (OPEX)','Постоянные расходы недвижимости и офиса','Аренда переговорных и представительского офиса, коммунальные платежи, клининг и охрана; регламент пересмотра ставок ежеквартальный, ответственный финансовый директор.','http://localhost:9000/media/screen.png','http://localhost:9000/media/7426703-hd_1080_1920_25fps.mp4','fixed',2850000,24),
('Фонд оплаты труда (ФОТ)','Оклады управленческой команды','Оклады, страховые взносы и ДМС управленческой команды головного офиса; не зависит от выручки, индексируется раз в год.','http://localhost:9000/media/screen_2.png','http://localhost:9000/media/7652821-hd_1080_1920_25fps.mp4','fixed',4620000,38),
('Себестоимость продаж (COGS)','Переменные затраты производства','Сырьё, упаковка, складская обработка и магистральная доставка до РЦ; норматив пересчитывается от фактической партии еженедельно.','http://localhost:9000/media/screen_3.png','http://localhost:9000/media/7818630-hd_1080_1920_30fps.mp4','variable',7410000,28),
('Маркетинг и аналитика','Переменные расходы на продвижение','Медиабаинг, сквозная аналитика и A/B-платформы; планируются от процента выручки, поэтому относятся к переменной части.','http://localhost:9000/media/screen_4.png','http://localhost:9000/media/7643847-uhd_2160_4096_25fps.mp4','variable',1150000,10);

INSERT INTO bep_requests (status, creator_id, product_name, selling_price) VALUES
('draft', 1, 'Новая продуктовая линейка', 1250);

INSERT INTO request_cost_links (request_id, cost_id, volume, is_critical, sort_order, comment) VALUES
(1, 1, 1, TRUE, 1, 'Офис, контракт на 11 мес.');