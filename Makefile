.PHONY: run test vet compose-up compose-down

run:
	go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down

