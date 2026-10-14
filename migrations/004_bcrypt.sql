-- ЛР-4: если база уже создана в ЛР-2/ЛР-3 (пароли в sha256) — перевести всех на bcrypt.
-- Пароль у тестовых пользователей остаётся 123456.
UPDATE users SET password_hash = '$2a$10$Bc8jFFkFDV39Ehb2qwmfH.GFFtiuUF8t78gnYFKs2E83BnePgiEh6'
WHERE password_hash NOT LIKE '$2a$%';
