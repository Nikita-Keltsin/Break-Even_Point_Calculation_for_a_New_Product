# Асинхронный поворот модели вида затрат

NestJS принимает `POST /rotations` с JSON `{"id":123}` и заголовком `X-Rotation-Key`. Сервис читает текущие углы у Go-бэкенда, ждёт 5–10 секунд и отправляет результат обратно через HTTP. Доступ к PostgreSQL выполняет основной сервис.

Установите зависимости и запустите сервис из каталога репозитория.

```bash
npm ci
npm run build
ROTATION_KEY='<тот же ключ, что у бэкенда>' npm start
```

Переменная `MAIN_API_URL` указывает на Go-бэкенд, по умолчанию `http://127.0.0.1:8080`. NestJS слушает `127.0.0.1:3100`, адрес можно задать через `HOST`, порт через `PORT`. Для совместного запуска используйте ветку `extra-async-rotation` в [бэкенде](https://github.com/Nikita-Keltsin/Break-Even_Point_Calculation_for_a_New_Product) и во [фронтенде](https://github.com/Nikita-Keltsin/Break-Even_Point_Calculation_for_a_New_Product_Frontend).
