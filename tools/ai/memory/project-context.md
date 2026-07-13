# Project context

`short` — API сокращения ссылок на Go 1.23.

- Точка входа: `cmd/api/main.go`.
- HTTP API: стандартный `net/http`, маршруты находятся в `internal/httpapi`.
- Бизнес-логика и интерфейсы хранилищ: `internal/link`.
- PostgreSQL через `pgx`; SQL-миграции встроены в бинарник и применяются при старте.
- Redis через `go-redis`; кеш хранит соответствие короткого кода и ссылки.
- Исходные URL поддерживаются длиной до 65535 символов; лимит JSON-запроса на создание ссылки — 128 KiB.
- Локальная инфраструктура: `docker-compose.yml`.
- CI и tag-based deployment в GHCR: `.github/workflows`; Docker-образ публикуется для `linux/amd64`, `linux/arm64` и `linux/arm/v7`.
- Production API обращается к отдельным контейнерам PostgreSQL и Redis через `host.docker.internal`; deploy добавляет это имя через `host-gateway` для Linux.
