# short

API сокращения ссылок на Go с PostgreSQL как основным хранилищем и Redis как cache-aside слоем.

## Быстрый старт

Требуется Docker с Compose:

```bash
docker compose up --build
```

API будет доступен на `http://localhost:8080`.

Создать короткую ссылку:

```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com/page"}'
```

Можно передать срок действия в UTC:

```json
{
  "original_url": "https://example.com/page",
  "expires_at": "2027-01-01T00:00:00Z"
}
```

Ответ `201 Created`:

```json
{
  "code": "Ab12Cd34",
  "original_url": "https://example.com/page",
  "created_at": "2026-07-13T12:00:00Z",
  "short_url": "http://localhost:8080/Ab12Cd34"
}
```

`GET /{code}` отвечает `302 Found`. Проверки процесса: `GET /health/live` и `GET /health/ready`.

## Конфигурация

Все настройки передаются переменными окружения; безопасный пример находится в `.env.example`. Обязательны `DATABASE_URL` и корректный `PUBLIC_BASE_URL`. `DEFAULT_LINK_TTL=0` означает, что ссылки по умолчанию не истекают.

## Проверки

```bash
go test ./...
go vet ./...
```

## Деплой

Workflow `.github/workflows/deploy.yml` запускается вручную или тегом `v*`, публикует в GHCR образ для `linux/amd64`, `linux/arm64` и `linux/arm/v7`, затем обновляет контейнер `short` на сервере по SSH.

Нужны GitHub Secrets: `SSH_HOST`, `SSH_USER`, `SSH_PRIVATE_KEY`, `SSH_PORT`, `GHCR_USER`, `GHCR_TOKEN`, `APP_PORT`, `DATABASE_URL`, `REDIS_ADDR`, `REDIS_PASSWORD`, `PUBLIC_BASE_URL`. PostgreSQL и Redis должны быть доступны серверу деплоя заранее.

Если PostgreSQL и Redis запущены на том же Linux-сервере в отдельных контейнерах с опубликованными портами, используй:

```text
DATABASE_URL=postgres://short:password@host.docker.internal:5432/short?sslmode=disable
REDIS_ADDR=host.docker.internal:6379
```

Deploy workflow добавляет `host.docker.internal` через Docker `host-gateway`. Порты PostgreSQL и Redis должны быть опубликованы на хосте и доступны с Docker bridge; ограничь внешний доступ к ним firewall сервера.
