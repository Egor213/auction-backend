# Auction Platform

> Платформа для проведения онлайн-аукционов на Go: создание лотов, приём ставок через HTTP, асинхронная обработка ставок через Kafka, распределённые блокировки в Redis, отказоустойчивость через Circuit Breaker и Retry, полный стек мониторинга (Prometheus + Grafana).

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7+-DC382D?logo=redis)
![Kafka](https://img.shields.io/badge/Kafka-3+-231F20?logo=apachekafka)
![Prometheus](https://img.shields.io/badge/Prometheus-2+-E6522C?logo=prometheus)
![Grafana](https://img.shields.io/badge/Grafana-10+-F46800?logo=grafana)

---

## Содержание

- [О проекте](#о-проекте)
- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Технологии](#технологии)
- [Быстрый старт](#быстрый-старт)
- [Конфигурация](#конфигурация)
- [API](#api)
- [Как это работает](#как-это-работает)
- [Мониторинг](#мониторинг)
- [Структура проекта](#структура-проекта)
- [Разработка](#разработка)
- [Безопасность](#безопасность)
- [Roadmap](#roadmap)
- [Лицензия](#лицензия)

---

## О проекте

**Auction Platform** — это backend-сервис на Go для проведения онлайн-аукционов. Сервис позволяет создавать лоты, принимать ставки, автоматически завершать аукционы по истечении срока и определять победителя.

Ключевая особенность проекта — **асинхронная обработка ставок через Kafka**. HTTP-запрос на размещение ставки лишь сохраняет её со статусом `PENDING` и публикует событие в Kafka. Отдельный consumer читает событие, валидирует ставку с учётом текущего состояния аукциона и обновляет её статус. Такой подход снижает latency HTTP-ответа и развязывает нагрузку между приёмом и обработкой ставок.

Проект построен по принципам слоистой архитектуры:

- `controller/http/v1` — HTTP-слой на Echo;
- `service` — бизнес-логика;
- `repo` — доступ к PostgreSQL;
- `infrastruct` — внешние интеграции: Kafka, Circuit Breaker, Retry;
- `worker` — фоновые задачи (завершение аукционов);
- `entity` — доменные модели и константы.

Сервис поддерживает:
- создание и управление аукционами;
- асинхронную обработку ставок через Kafka;
- распределённые блокировки через Redis для предотвращения гонок;
- автоматическое завершение аукционов по истечении срока;
- Circuit Breaker для PostgreSQL и Kafka Producer;
- Retry с экспоненциальной задержкой;
- Rate Limiting (глобальный и per-IP);
- метрики Prometheus и дашборды Grafana;
- graceful shutdown.

---

## Возможности

- **Управление аукционами**  
  Создание лотов с указанием стартовой цены, минимального шага и длительности. Список активных аукционов с пагинацией.

- **Асинхронная обработка ставок**  
  HTTP-запрос `POST /bid/place` сохраняет ставку в PostgreSQL и публикует событие `BidPlacedEvent` в Kafka. Consumer читает событие и выполняет валидацию.

- **Валидация ставок**  
  Проверка: аукцион активен, ставка больше текущей цены + минимальный шаг, продавец не может делать ставку на собственный лот.

- **Распределённые блокировки**  
  Redis-лок `lock:auction:<auction_id>` защищает от одновременной обработки ставок по одному аукциону. Реализован через `SETNX` с TTL и Lua-скрипт для безопасного снятия.

- **Автоматическое завершение аукционов**  
  Воркер `BidProcessor` каждые 10 секунд проверяет истёкшие аукционы, определяет победителя (по наибольшей ставке) и публикует событие `AuctionEndedEvent`.

- **Circuit Breaker**  
  Для PostgreSQL и Kafka Producer через `sony/gobreaker`. Настраивается через `failure_ratio`, `min_requests`, `interval`, `timeout`.

- **Retry с экспоненциальной задержкой**  
  Кастомный `Retryer` с настраиваемыми `MaxAttempts`, `InitialWait`, `MaxWait`, `Multiplier`. Учитывает `context.Context` для отмены.

- **Rate Limiting**  
  Глобальный и per-IP лимиты на базе `golang.org/x/time/rate`.

- **Метрики Prometheus**  
  HTTP-метрики, бизнес-метрики (создано аукционов, ставок, принято/отклонено), метрики Kafka, Circuit Breaker, Retry, DB.

- **Health Checks**  
  `GET /health` — liveness, `GET /ready` — readiness с проверкой PostgreSQL и Redis.

- **Graceful shutdown**  
  Корректное завершение HTTP-сервера, Kafka consumer и воркеров.

- **Миграции**  
  Автоматическое применение миграций при старте через `golang-migrate`.

---

## Архитектура

```mermaid
flowchart TB
    Client["HTTP Client"] --> API["Echo API"]
    API --> AuctionSvc["Auction Service"]
    API --> BidSvc["Bid Service"]

    BidSvc --> KafkaProducer["Kafka Producer"]
    KafkaProducer --> Kafka[("Kafka")]
    Kafka --> KafkaConsumer["Kafka Consumer"]
    KafkaConsumer --> BidSvc

    Worker["Bid Processor Worker"] --> AuctionRepo
    Worker --> BidSvc
    Worker --> KafkaProducer

    AuctionSvc --> CB["Circuit Breaker"]
    BidSvc --> CB
    CB --> AuctionRepo[("PostgreSQL")]
    CB --> BidRepo[("PostgreSQL")]

    BidSvc --> Redis[("Redis Lock")]
    CB --> Retryer["Retryer"]
```

### Поток размещения ставки

1. Клиент отправляет `POST /api/v1/bid/place`.
2. `BidService.PlaceBid` сохраняет ставку в PostgreSQL со статусом `PENDING`.
3. Публикуется событие `BidPlacedEvent` в топик `bid_placed`.
4. HTTP-ответ возвращается сразу с кодом `202 Accepted`.
5. Kafka Consumer читает событие из топика `bid_placed`.
6. `BidService.ProcessBidEvent` захватывает Redis-лок `lock:auction:<auction_id>`.
7. Загружается актуальное состояние аукциона.
8. Проверки: статус `ACTIVE`, ставка ≥ `current_bid + min_step`, `bidder_id != seller_id`.
9. При успехе: обновляется `current_bid` аукциона, ставка переводится в `ACCEPTED`, публикуется `BidResultEvent`, инвалидируется кэш.
10. При ошибке: ставка переводится в `REJECTED`, публикуется `BidResultEvent` с причиной.

### Поток завершения аукциона

1. Воркер `BidProcessor.StartExpiryChecker` запускает тикер на 10 секунд.
2. На каждом тике выполняется `GetExpired` — выборка аукционов со статусом `ACTIVE` и `ends_at <= NOW()`.
3. Для каждого аукциона находится победитель (наибольшая ставка).
4. Вызывается `FinishAuction` — обновляется статус, `winner_id`, `current_bid`, `finished_at`.
5. Публикуется событие `AuctionEndedEvent` в топик `auction_ended`.
6. Инкрементируются метрики `AuctionsFinished` и `ActiveAuctions.Dec()`.

---

## Технологии

| Слой | Технологии |
|---|---|
| Язык | Go 1.22 |
| HTTP | Echo v4 |
| Конфигурация | cleanenv, godotenv |
| PostgreSQL | pgx v5, Squirrel |
| Миграции | golang-migrate |
| Redis | go-redis/v9 |
| Kafka | segmentio/kafka-go |
| Circuit Breaker | sony/gobreaker |
| Retry | кастомный `internal/infrastruct/retry` |
| Валидация | go-playground/validator |
| Логирование | logrus (JSON) |
| Метрики | Prometheus client_golang |
| Мониторинг | Prometheus, Grafana |
| Транзакции | avito-tech/go-transaction-manager |

---

## Быстрый старт

### Требования

- Docker и Docker Compose
- (Опционально) Go 1.22 для локальной разработки

### 1. Клонирование

```bash
git clone <repo-url>
cd auction-platform
```

### 2. Запуск через Docker Compose

```bash
docker-compose up -d --build
```

После запуска будут доступны сервисы:

| Сервис | URL |
|---|---|
| API | http://localhost:8080 |
| Kafka UI | http://localhost:8090 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin / admin) |

### 3. Проверка работоспособности

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/ready
# {"checks":{"postgres":"ok","redis":"ok"}}
```

### 4. Остановка

```bash
docker-compose down -v
```

### Локальный запуск без Docker

1. Поднимите инфраструктуру:
   ```bash
   docker-compose up -d postgres redis zookeeper kafka
   ```

2. Создайте `infra/.env` (см. [Конфигурацию](#конфигурация)).

3. Запустите приложение:
   ```bash
   go mod download
   go run ./cmd/app
   ```

Миграции применяются автоматически в `app.Run()`.

---

## Конфигурация

Конфигурация загружается в два этапа:

1. Из YAML-файла (`config/config.yaml`), путь задаётся через `APP_CONFIG_PATH`.
2. Из переменных окружения через `cleanenv.UpdateEnv`, которые **переопределяют** значения из YAML.

Файл `infra/.env` загружается через `godotenv` (опционально).

### Переменные окружения

| Переменная | Обязательна | По умолчанию | Описание |
|---|---:|---:|---|
| `APP_CONFIG_PATH` | нет | `config/config.yaml` | Путь к YAML-конфигу |
| `SERVER_ADDRESS` | да | — | Адрес HTTP-сервера |
| `LOG_LEVEL` | нет | `info` | Уровень логирования |
| `POSTGRES_CONN` | да | — | DSN для PostgreSQL |
| `MAX_POOL_SIZE` | да | — | Размер пула подключений PostgreSQL |
| `REDIS_ADDR` | нет | `localhost:6379` | Адрес Redis |
| `REDIS_PASSWORD` | нет | — | Пароль Redis |
| `REDIS_DB` | нет | `0` | Номер БД Redis |
| `KAFKA_BROKERS` | нет | `localhost:9092` | Список брокеров Kafka |
| `RATE_RPS` | нет | — | Глобальный RPS |
| `RATE_BURST` | нет | — | Глобальный burst |

### Пример `config/config.yaml`

```yaml
app:
  name: auction-platform
  version: 1.0.0

http:
  address: ":8080"

log:
  level: info

postgres:
  max_pool_size: 10

kafka:
  bid_placed_topic: bid_placed
  bid_result_topic: bid_result
  auction_ended_topic: auction_ended
  group_id: auction-group

redis:
  cache_ttl: 5m

rate_limiter:
  rps: 100
  burst: 200

retry:
  max_attempts: 3
  initial_wait: 100ms
  max_wait: 2s
  multiplier: 2.0

circuit_breaker:
  max_requests: 5
  interval: 10s
  timeout: 30s
  min_requests: 10
  failure_ratio: 0.5
```

### Пример `infra/.env`

```env
SERVER_ADDRESS=:8080
POSTGRES_CONN=postgres://auction:auction@localhost:5432/auction?sslmode=disable
MAX_POOL_SIZE=10
KAFKA_BROKERS=localhost:9092
REDIS_ADDR=localhost:6379
LOG_LEVEL=debug
RATE_RPS=100
RATE_BURST=200
```

> Не коммитьте `infra/.env` и реальные секреты в репозиторий.

---

## API

Базовый префикс: `/api/v1`

### Health

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/health` | Liveness-проверка |
| `GET` | `/ready` | Readiness-проверка (PostgreSQL, Redis) |
| `GET` | `/metrics` | Метрики Prometheus |

### Аукционы

#### Создать аукцион

```http
POST /api/v1/auction/create
Content-Type: application/json

{
  "auction_id": "auc-001",
  "title": "Vintage Watch",
  "description": "Rare 1960s watch",
  "seller_id": "seller-1",
  "start_price": 1000,
  "min_step": 50,
  "duration_min": 60
}
```

**Ответ:** `201 Created`

```json
{
  "auction": {
    "auction_id": "auc-001",
    "title": "Vintage Watch",
    "description": "Rare 1960s watch",
    "seller_id": "seller-1",
    "start_price": 1000,
    "current_bid": 1000,
    "min_step": 50,
    "status": "ACTIVE",
    "ends_at": "2025-01-01T12:00:00Z",
    "created_at": "2025-01-01T11:00:00Z"
  }
}
```

Возможные ошибки:

| Код | Описание |
|---|---|
| `400` | Невалидные параметры |
| `409` | Аукцион уже существует |

#### Получить аукцион

```http
GET /api/v1/auction/get?auction_id=auc-001
```

#### Список активных аукционов

```http
GET /api/v1/auction/list?page=1&page_size=20
```

**Ответ:**

```json
{
  "auctions": [ /* ... */ ],
  "total": 42,
  "page": 1,
  "page_size": 20,
  "total_pages": 3
}
```

### Ставки

#### Сделать ставку

```http
POST /api/v1/bid/place
Content-Type: application/json

{
  "bid_id": "bid-001",
  "auction_id": "auc-001",
  "bidder_id": "bidder-1",
  "amount": 1100
}
```

**Ответ:** `202 Accepted`

```json
{
  "bid": {
    "bid_id": "bid-001",
    "auction_id": "auc-001",
    "bidder_id": "bidder-1",
    "amount": 1100,
    "status": "PENDING",
    "created_at": "2025-01-01T11:05:00Z"
  }
}
```

> Ставка создаётся со статусом `PENDING`. Итоговый статус (`ACCEPTED` / `REJECTED`) проставляется асинхронно после обработки Kafka-события. Для получения актуального статуса используйте `GET /bid/list`.

Возможные ошибки:

| Код | Описание |
|---|---|
| `400` | Невалидные параметры / ставка слишком низкая |
| `404` | Аукцион не найден |
| `409` | Аукцион завершён |

#### Список ставок по аукциону

```http
GET /api/v1/bid/list?auction_id=auc-001&limit=50
```

---

## Как это работает

### Размещение ставки

Логика в `internal/service/bid.go`.

1. HTTP-хендлер вызывает `BidService.PlaceBid`.
2. Через `CircuitBreaker.Execute("postgres", ...)` и `Retryer.Do(ctx, "create_bid", ...)` создаётся ставка в PostgreSQL со статусом `PENDING`.
3. Инкрементируются метрики `BidsPlaced` и `BidAmountHistogram`.
4. Публикуется событие `BidPlacedEvent` в топик `bid_placed` через `Producer.Publish` (тоже под Circuit Breaker `kafka_producer` и Retryer).
5. Если публикация не удалась, выполняется fallback — синхронный вызов `ProcessBidEvent`.
6. HTTP-ответ `202 Accepted` с `PENDING`-ставкой.

### Обработка ставки

1. Kafka Consumer читает сообщение из `bid_placed`.
2. `BidService.ProcessBidEvent`:
   - захватывает Redis-лок `lock:auction:<auction_id>` (TTL 5 секунд);
   - загружает аукцион из PostgreSQL;
   - проверяет статус `ACTIVE`;
   - проверяет, что `bidder_id != seller_id`;
   - проверяет `amount >= current_bid + min_step`;
   - обновляет `current_bid` у аукциона;
   - переводит ставку в `ACCEPTED`;
   - инвалидирует кэш `auction:<auction_id>` в Redis;
   - публикует `BidResultEvent` в топик `bid_result`;
   - инкрементирует `BidsAccepted`.
3. При нарушении любой проверки — `rejectBid`: ставка переводится в `REJECTED`, публикуется `BidResultEvent` с причиной, инкрементируется `BidsRejected`.

### Завершение аукциона

Логика в `internal/worker/bidprocessor.go`.

1. Воркер стартует вместе с приложением и запускает тикер с интервалом 10 секунд.
2. `checkExpired` вызывает `AuctionRepo.GetExpired` — выборка аукционов со статусом `ACTIVE` и `ends_at <= NOW()`.
3. Для каждого аукциона `finishAuction`:
   - находит победителя через `BidService.GetHighestBid`;
   - вызывает `AuctionRepo.FinishAuction` — обновляет статус, `winner_id`, `current_bid`, `finished_at`;
   - считает общее число ставок;
   - публикует `AuctionEndedEvent` в топик `auction_ended`;
   - инкрементирует `AuctionsFinished`, декрементирует `ActiveAuctions`.

### Отказоустойчивость

- **Circuit Breaker** — `sony/gobreaker` с двумя экземплярами: `postgres` и `kafka_producer`. Настраивается через `CircuitBreakerConfig`. Метрики состояния пишутся в `CircuitBreakerState` и `CircuitBreakerTrips`.
- **Retryer** — `internal/infrastruct/retry`. Экспоненциальный backoff: `min(InitialWait * Multiplier^(attempt-1), MaxWait)`. Метрики `RetryAttempts` и `RetryExhausted`.
- **Rate Limiter** — глобальный и per-IP, на базе `golang.org/x/time/rate`. Per-IP лимит = 1/10 от глобального. Метрика `RateLimiterRejected`.

---

## Мониторинг

### Метрики Prometheus

Доступны по адресу `http://localhost:8080/metrics`.

Категории метрик:

| Метрика | Тип | Описание |
|---|---|---|
| `auction_http_requests_total` | CounterVec | Всего HTTP-запросов (method, path, status) |
| `auction_http_request_duration_seconds` | HistogramVec | Длительность HTTP-запросов |
| `auction_http_active_requests` | Gauge | Активные HTTP-запросы |
| `auction_auctions_created_total` | Counter | Создано аукционов |
| `auction_auctions_finished_total` | Counter | Завершено аукционов |
| `auction_bids_placed_total` | Counter | Размещено ставок |
| `auction_bids_accepted_total` | Counter | Принято ставок |
| `auction_bids_rejected_total` | Counter | Отклонено ставок |
| `auction_active_auctions` | Gauge | Активных аукционов |
| `auction_bid_amount` | Histogram | Распределение сумм ставок |
| `auction_kafka_produced_total` | CounterVec | Опубликовано Kafka-сообщений |
| `auction_kafka_consumed_total` | CounterVec | Прочитано Kafka-сообщений (status: success/error) |
| `auction_kafka_produce_errors_total` | CounterVec | Ошибки публикации в Kafka |
| `auction_kafka_consume_latency_seconds` | HistogramVec | Задержка обработки Kafka-сообщений |
| `auction_cb_state` | GaugeVec | Состояние Circuit Breaker (0=closed, 1=half-open, 2=open) |
| `auction_cb_trips_total` | CounterVec | Срабатывания Circuit Breaker |
| `auction_rate_limiter_rejected_total` | Counter | Отклонено Rate Limiter |
| `auction_retry_attempts` | HistogramVec | Количество попыток Retry |
| `auction_retry_exhausted_total` | CounterVec | Исчерпаны все попытки Retry |
| `auction_db_query_duration_seconds` | HistogramVec | Длительность DB-запросов |
| `auction_db_errors_total` | CounterVec | Ошибки DB |

### Prometheus

Конфигурация: `monitoring/prometheus/prometheus.yml`.  
Alerting-правила: `monitoring/alerting/rules.yml`.

Интерфейс: `http://localhost:9090`.

### Grafana

Интерфейс: `http://localhost:3000` (admin / admin).  
Data source Prometheus добавляется автоматически через volume.

### Kafka UI

Интерфейс: `http://localhost:8090`.  
Позволяет просматривать топики `bid_placed`, `bid_result`, `auction_ended` и содержимое сообщений.

---

## Структура проекта

```text
.
├── cmd/
│   └── app/
│       └── main.go                 # Точка входа
├── config/                         # config.yaml
├── infra/                          # .env и инфраструктурные файлы
├── internal/
│   ├── app/                        # запуск, миграции, graceful shutdown
│   ├── config/                     # загрузка конфигурации
│   ├── controller/
│   │   └── http/
│   │       └── v1/                 # Echo handlers, DTO, middleware, mappers
│   ├── entity/                     # доменные модели (Auction, Bid)
│   ├── infrastruct/
│   │   ├── circuitbreaker/         # обёртка над sony/gobreaker
│   │   ├── kafka/                  # producer, consumer, DTO событий
│   │   └── retry/                  # кастомный Retryer
│   ├── metrics/                    # Prometheus-метрики
│   ├── repo/
│   │   ├── dto/                    # DTO репозиториев
│   │   ├── errors/                 # ошибки репозиториев
│   │   └── pgdb/                   # PostgreSQL
│   ├── service/
│   │   ├── dto/                    # DTO сервисов
│   │   ├── errors/                 # ошибки сервисов
│   │   └── mappers/                # мапперы между слоями
│   └── worker/                     # фоновые задачи
├── migrations/                     # SQL-миграции
├── monitoring/
│   ├── alerting/
│   │   └── rules.yml
│   └── prometheus/
│       └── prometheus.yml
├── pkg/                            # переиспользуемые пакеты
│   ├── errors/
│   ├── httpserver/
│   ├── logger/
│   ├── postgres/
│   └── validator/
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## Разработка

### Сборка

```bash
go build -o auction-app ./cmd/app
```

### Тесты

```bash
go test ./...
```

### Статический анализ

```bash
go vet ./...
```

### Форматирование

```bash
gofmt -w .
```

### Миграции

Применить:

```bash
migrate -path migrations -database "postgres://auction:auction@localhost:5432/auction?sslmode=disable" up
```

Откатить:

```bash
migrate -path migrations -database "postgres://auction:auction@localhost:5432/auction?sslmode=disable" down
```

Создать новую:

```bash
migrate create -ext sql -dir migrations -seq <migration_name>
```

### Логи

Логи пишутся в stdout в JSON-формате с полями `method`, `path`, `status`, `latency`, `remote_ip`.  
Уровень логирования задаётся через `LOG_LEVEL` (`debug`, `info`, `warn`, `error`).

---

## Безопасность

- Circuit Breaker защищает от каскадных отказов PostgreSQL и Kafka.
- Retry с экспоненциальной задержкой снижает нагрузку на деградирующие сервисы.
- Rate Limiter (глобальный и per-IP) защищает API от перегрузки.
- Redis-лок предотвращает гонки при обработке ставок по одному аукциону.
- Валидация входных данных через `go-playground/validator` на уровне HTTP-DTO.
- Не коммитьте `infra/.env`, `config/config.yaml` и реальные секреты.
- Используйте разные ключи и пароли для dev/stage/prod.
- Ограничьте сетевой доступ к PostgreSQL, Redis и Kafka.
- Для production включите TLS для PostgreSQL и Kafka.

---

## Roadmap

- [ ] Аутентификация и авторизация HTTP API (JWT).
- [ ] OpenAPI/Swagger-документация.
- [ ] Idempotency-Key для безопасного ретрая ставок.
- [ ] Outbox-паттерн для гарантированной публикации событий.
- [ ] Оптимизация `GetExpired` через `FOR UPDATE SKIP LOCKED` для мультиинстансного воркера.
- [ ] Отдельные Kafka consumer-группы для `bid_placed` и `auction_ended`.
- [ ] Кэширование `GetByID` в Redis с TTL.
- [ ] Graceful drain Kafka consumer при shutdown.
- [ ] Unit- и integration-тесты (testcontainers).
- [ ] CI/CD pipeline (GitHub Actions).
- [ ] Distributed tracing (OpenTelemetry).
- [ ] Rate limiting на базе Redis для мультиинстансного деплоя.

---
