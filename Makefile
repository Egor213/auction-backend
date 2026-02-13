.PHONY: build run up down logs

build:
	go build -o bin/app ./cmd/app

run:
	go run ./cmd/app

infra-up:
	docker-compose up -d postgres redis zookeeper kafka kafka-ui prometheus grafana

infra-down:
	docker-compose down

up:
	docker-compose up -d --build

down:
	docker-compose down -v

logs:
	docker-compose logs -f app