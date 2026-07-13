# Change log

## 2026-07-13

- Создан API сокращения ссылок с истечением срока действия.
- Добавлены PostgreSQL, Redis, миграция, Docker Compose, CI и GitHub Actions deployment.
- Добавлена репозиторная память и инструкции для AI-агентов.
- GitHub Actions сборка расширена платформой `linux/arm/v7`.
- В production deploy добавлено Linux-сопоставление `host.docker.internal:host-gateway` для подключения к отдельным контейнерам PostgreSQL и Redis.
- Поддержка `original_url` расширена с 4096 до 65535 символов, включая HTTP-лимит и миграцию PostgreSQL.
- Исправлены переходы по длинным ссылкам через nginx: назначения длиннее 2048 байт открываются клиентским HTML-переходом без большого заголовка `Location`.
- Исправлено сохранение URL fragments: разделитель `#` и percent-encoded данные больше не искажаются при создании ссылки.
