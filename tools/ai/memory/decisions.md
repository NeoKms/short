# Decisions

## 2026-07-13: базовая архитектура

- Выбраны Go, PostgreSQL и Redis.
- HTTP реализован стандартной библиотекой, чтобы сохранить малое число зависимостей.
- PostgreSQL хранит ссылки постоянно, Redis используется как ускоряющий cache-aside слой.
- Деплой повторяет `team-planning`: tag `v*` публикует образ для `linux/amd64`, `linux/arm64` и `linux/arm/v7` в GHCR и обновляет контейнер по SSH.
- PostgreSQL и Redis развёрнуты отдельно; API подключается к их опубликованным портам через `host.docker.internal`, сопоставленный с `host-gateway` при `docker run`.
