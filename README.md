# Система сбора логов

## Введение

Централизованная система сбора логов — это платформа для агрегации, хранения и анализа логов из множества источников (приложения, серверы, контейнеры, базы данных).

В рамках курсовой работы требуется реализовать прототип системы сбора логов, способный принимать логи различных форматов, индексировать их и предоставлять возможность поиска и анализа.

## Цель работы

Создать работающий прототип системы сбора логов, способный:

- собирать логи из разных источников и форматов
- парсить и структурировать логи
- индексировать данные для быстрого поиска
- обнаруживать аномалии и ошибки
- визуализировать метрики в аналитических дашбордах

## Постановка задачи

### Сбор данных

Необходимо реализовать получение логов из трёх разных источников:

- файл .log/.txt/.csv/.json (файлы загружаются через отдельный endpoint, который принимает запрос с multipart/form-data)
- HTTP
- Брокер сообщений

Система должна поддерживать разные форматы логов:

- JSON (структурированные логи)
- Syslog format (RFC5424)
- CLF (Common Log Format)

#### Генерация данных

В вашем проекте должны быть **файлы** с тестовыми логами для импорта в хранилище.

Помимо этого, система должна получать логи в реальном времени через брокер сообщений (Kafka/RabbitMQ) от генератора, запущенного в фоне и имитирующего работающие приложения, пишущие логи.

### Хранение данных

#### Примерная схема лога

> Может быть доработана на ваше усмотрение

- `id` *UInt64* — уникальный идентификатор лог-записи
- `created_at` *DateTime* — время создания лога (время создания в источнике)
- `received_at` *DateTime* — время получения системой
- `level` *UInt8*/*String* — уровень (Critical, Info, Error, Warning, Debug и др.)
- `source` *String* — источник лога (имя сервиса/приложения)
- `host` *String* — хост, с которого пришел лог
- `environment` *String* — окружение (dev/staging/production)
- `message` *String* — основное сообщение лога
- `payload` *JSON* — дополнительные структурированные данные
  - `user_id` *UInt64* — пользователь, если применимо
  - `duration_ms` *UInt32* — длительность операции
  - `http_status_code` *UInt16* — HTTP-код статуса запрос (200, 404 и др.)
  - `error_type` *String* — тип ошибки (логическая/исключение/паника и т.д.)
  - `stack_trace` *String* — стек-трейс ошибки (необязательно)

### REST API для взаимодействия с приложением

> Должно быть, **как минимум**, **2 эндпоинта**.
> Можно добавлять и другие на свое усмотрение, например, для удобства отладки или демонстрации работы.

#### Добавление логов

`POST /logs`

Принимает один или несколько логов.

Пример запроса:

```json
[
    {
        "created_at": "2025-01-01T12:00:00Z",
        "level": "error",
        "source": "payment-service",
        "host": "prod-server-01.com",
        "environment": "production",
        "message": "Payment processing failed",
        "payload": {
            "user_id": 1001,
            "http_status_code": 504,
            "error_type": "TimeoutException",
            "duration_ms": 5000
        }
    }
]
```

#### Поиск логов

`GET /logs/search`

Получение логов по фильтрам с пагинацией.

Пример запроса:

```plaintext
GET /logs/level=error&source=payment-service&from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z&limit=100&offset=0
```

> ❗️ Реализуйте фильтрацию по тексту сообщения в логах

Пример ответа:

```json
{
    "total": 42,
    "logs": [
        {
            "id": 123456,
            "created_at": "2025-01-01T12:00:00Z",
            "level": "error",
            "source": "payment-service",
            "message": "Payment processing failed",
            "payload": {
                "error_type": "TimeoutException"
            }
        }
    ]
}
```

#### Загрузка файлов

`GET /logs/upload`

Отправка файла с логами c через multipart/form-data

### Мониторинг

Используйте [Prometheus](https://prometheus.io/), [Grafana](https://grafana.com/)
для настройки мониторинга работы сервиса и инфраструктуры.

> Метрики мониторинга могут быть кастомизированы на ваше усмотрение
Рекомендуется настроить
- RPS приложения, response time, latency
- Время парсинга логов (min, avg, max)
- Использование ресурсов (CPU, оперативная и постоянная память, сеть)
- Статус компонентов системы (БД, брокеры и т.д.)

# Реализация

## Сборка и запуск

### Требования
- Go 1.25.6
- Docker и Docker Compose (для запуска с контейнерами)

### Сборка и запуск

```bash
# Сборка всех компонентов (генерация openapi, swagger страницы, сборка collector и generator)
make

# Запуск тестов
make test

# Сборка только collector
make build-collector

# Сборка только generator
make build-generator

# Запуск всех сервисов в Docker (RabbitMQ, ClickHouse, Collector, Generator)
make docker-run

# Остановка всех сервисов
make docker-stop
```

### Конфигурация

Конфигурация задается через YAML файлы и переменные окружения:

- `configs/config.yaml` - Конфигурация для запуска в Docker
- `configs/local.yaml` - Конфигурация для локальной разработки
- `configs/generator.yaml` - Конфигурация генератора логов для Docker
- `configs/generator.local.yaml` - Конфигурация генератора для локальной разработки

Переменные окружения для конфигурации:
- `COLLECTOR_CONFIG_PATH` - Путь к конфигурационному файлу collector (по умолчанию: `/etc/collector/config.yaml`)
- `GENERATOR_CONFIG_PATH` - Путь к конфигурационному файлу generator (по умолчанию: `/etc/generator/config.yaml`)
- `HTTP_PORT`, `HTTP_HOST` - Настройки HTTP сервера
- `CLICKHOUSE_*` - Настройки подключения к ClickHouse
- `RABBITMQ_*` - Настройки подключения к RabbitMQ
- `GENERATOR_TIMEOUT` - Задержка между сообщениями генератора
- `GENERATOR_MODE` - Режим работы генератора (`rabbitmq` или `stdout`)
- `GENERATOR_FORMAT` - Формат логов (`json`, `syslog`, `clf`)
- `GENERATOR_COUNT` - Количество логов для генерации (0 = бесконечно)

### Генератор логов

Генератор создает случайные логи в форматах JSON, Syslog (RFC5424) и CLF и публикует их в RabbitMQ или выводит в stdout.

```bash
# Запуск генератора с локальной конфигурацией (вывод в stdout)
./build/generator -config configs/generator.local.yaml

# Запуск генератора для Docker (публикация в RabbitMQ)
./build/generator -config configs/generator.yaml

# Переопределение формата через ENV
GENERATOR_FORMAT=syslog GENERATOR_MODE=stdout ./build/generator -config configs/generator.local.yaml
```

## Примеры

### Примеры логов

Примеры логов различных форматов находятся в директории `samples/`:

- `samples/logs.json` - Примеры JSON логов
- `samples/logs.syslog` - Примеры Syslog (RFC5424) логов
- `samples/logs.clf` - Примеры CLF (Common Log Format) логов

### Примеры запросов к API

#### Добавление логов (POST /logs)

```bash
curl -X POST http://localhost:8080/logs \
  -H "Content-Type: application/json" \
  -d '[
    {
      "created_at": "2025-01-01T12:00:00Z",
      "level": "error",
      "source": "payment-service",
      "host": "prod-server-01.com",
      "environment": "production",
      "message": "Payment processing failed",
      "payload": {
        "user_id": 1001,
        "http_status_code": 504,
        "error_type": "TimeoutException",
        "duration_ms": 5000
      }
    }
  ]'
```

#### Загрузка файла логов (POST /logs/upload)

```bash
# Загрузка JSON файла
curl -X POST http://localhost:8080/logs/upload \
  -F "file=@samples/logs.json"

# Загрузка Syslog файла
curl -X POST http://localhost:8080/logs/upload \
  -F "file=@samples/logs.syslog"

# Загрузка CLF файла
curl -X POST http://localhost:8080/logs/upload \
  -F "file=@samples/logs.clf"
```

#### Поиск логов (GET /logs/search)

```bash
# Поиск по уровню логов
curl "http://localhost:8080/logs/search?level=error"

# Поиск по источнику и времени
curl "http://localhost:8080/logs/search?source=payment-service&from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z"

# Поиск по тексту сообщения с пагинацией
curl "http://localhost:8080/logs/search?message=failed&limit=10&offset=0"

# Комплексный поиск
curl "http://localhost:8080/logs/search?level=error&source=payment-service&environment=production&limit=50&offset=0&sort_by=created_at&sort_order=desc"
```

## API Endpoints

### POST /logs
Добавление одного или нескольких логов в формате JSON.

**Запрос:**
```json
[
  {
    "created_at": "2025-01-01T12:00:00Z",
    "level": "error",
    "source": "payment-service",
    "host": "prod-server-01.com",
    "environment": "production",
    "message": "Payment processing failed",
    "payload": {
      "user_id": 1001,
      "duration_ms": 5000,
      "http_status_code": 504,
      "error_type": "TimeoutException",
      "stack_trace": "..."
    }
  }
]
```

**Ответ (201 Created):**
```json
{
  "message": "2 logs added successfully",
  "inserted_count": 2,
}
```

### POST /logs/upload
Загрузка файла с логами (JSON, Syslog, CLF) через multipart/form-data.

**Формат запроса:** multipart/form-data с полем `file`

**Ответ (201 Created):**
```json
{
  "message": "2 logs added successfully",
  "inserted_count": 2,
}
```

### GET /logs/search
Поиск логов по фильтрам с пагинацией.

**Параметры запроса:**
- `level` - Фильтр по уровню (debug, info, warning, error, critical)
- `source` - Фильтр по источнику (имя сервиса)
- `host` - Фильтр по хосту
- `message` - Фильтр по тексту сообщения (поддерживает частичное совпадение)
- `environment` - Фильтр по окружению (dev, staging, production)
- `from` - Начало временного диапазона (RFC3339)
- `to` - Конец временного диапазона (RFC3339)
- `limit` - Максимальное количество результатов (1-1000, default: 100)
- `offset` - Количество результатов для пропуска (default: 0)
- `sort_by` - Поле для сортировки (created_at, level, source, default: created_at)
- `sort_order` - Порядок сортировки (asc, desc, default: desc)

**Ответ (200 OK):**
```json
{
  "total": 42,
  "logs": [
    {
      "id": "550e8400-e29b-41d4-a716-4466554400000",
      "created_at": "2025-01-01T12:00:00Z",
      "received_at": "2025-01-01T12:00:01Z",
      "level": "error",
      "source": "payment-service",
      "host": "prod-server-01.com",
      "environment": "production",
      "message": "Payment processing failed",
      "payload": {
        "user_id": 1001,
        "http_status_code": 504,
        "error_type": "TimeoutException"
      }
    }
  ]
}
```

## Swagger UI

Интерактивная документация API доступна по адресу:
- Local: http://localhost:8080/swagger/

## Метрики (Prometheus)

Метрики доступны по адресу: http://localhost:8080/metrics


- RPS приложения, response time, latency
- Время парсинга логов (min, avg, max)
- Использование ресурсов (CPU, оперативная и постоянная память, сеть)
- Статус компонентов системы (БД, брокеры и т.д.)

