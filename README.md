# Payment Processing Platform

A backend payment processing service built with Go and PostgreSQL.

The project focuses on designing a reliable backend system with explicit domain rules, well-defined application boundaries, persistent storage, and idempotent payment creation.

It is being developed as a practical engineering project to explore backend architecture, Domain-Driven Design (DDD), and the challenges of building payment-related systems.

> **Status:** Work in progress 🚧

## Overview

The service provides an HTTP API for creating, retrieving, and managing payments.

Each payment has a lifecycle with explicit state transitions:

```text
             ┌──────────────┐
             │    PENDING   │
             └──────┬───────┘
                    │
                    ▼
             ┌──────────────┐
             │  PROCESSING  │
             └──────┬───────┘
                    │
             ┌──────┴───────┐
             ▼              ▼
      ┌────────────┐  ┌────────────┐
      │ COMPLETED  │  │   FAILED   │
      └────────────┘  └────────────┘

PENDING / PROCESSING → CANCELLED
```

Invalid state transitions are rejected by the domain layer.

## Features

- **Payment lifecycle management** — explicit payment states and validated state transitions.
- **Idempotent payment creation** — prevents duplicate payments when clients retry requests using the same idempotency key.
- **Payload validation** — validates payment amounts, currencies, and identifiers.
- **Money representation** — stores monetary values as integers in minor currency units to avoid floating-point precision issues.
- **Currency support** — USD, EUR, and JPY, with currency-specific decimal precision.
- **PostgreSQL persistence** — stores payments and idempotency records.
- **Graceful shutdown** — handles SIGINT and SIGTERM and allows the HTTP server to shut down gracefully.
- **Structured logging** — JSON logs using Go's standard `log/slog` package.
- **Automated tests** — unit and HTTP handler tests.

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Backend development |
| PostgreSQL | Persistent storage |
| pgx | PostgreSQL driver and connection pooling |
| Chi | HTTP routing and middleware |
| UUID | Payment identifiers |
| Docker / Kubernetes | Future development goals |

## Architecture

The project follows a layered architecture inspired by Domain-Driven Design and Clean Architecture principles.

```text
                 ┌─────────────────────┐
                 │      HTTP API       │
                 │  Transport Layer    │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │  Application Layer  │
                 │   Payment Service   │
                 └──────────┬──────────┘
                            │
                  ┌─────────┴─────────┐
                  ▼                   ▼
        ┌─────────────────┐  ┌─────────────────┐
        │  Domain Layer   │  │ Infrastructure  │
        │                 │  │                 │
        │ Payment         │  │ PostgreSQL      │
        │ Money           │  │ Repository      │
        │ Currency        │  │                 │
        └─────────────────┘  └─────────────────┘
```

### Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── domain/
│   │   └── payment/
│   │       ├── payment.go
│   │       ├── money.go
│   │       └── currency.go
│   │
│   ├── application/
│   │   └── payment/
│   │       ├── service.go
│   │       ├── request.go
│   │       ├── errors.go
│   │       └── payload_hash.go
│   │
│   ├── infrastructure/
│   │   └── postgres/
│   │       ├── postgres.go
│   │       └── payment_repository.go
│   │
│   └── transport/
│       └── http/
│           ├── router.go
│           ├── handlers.go
│           └── response.go
│
├── migrations/
│   ├── 000001_create_payments.up.sql
│   ├── 000001_create_payments.down.sql
│   ├── 000002_create_idempotency_keys.up.sql
│   └── 000002_create_idempotency_keys.down.sql
│
├── go.mod
└── README.md
```

### Layer Responsibilities

**Domain Layer**

Contains core business concepts and rules:

- Payment entity.
- Payment status transitions.
- Money value object.
- Currency validation and precision rules.

The domain layer does not depend on HTTP, PostgreSQL, or external infrastructure.

**Application Layer**

Coordinates payment-related use cases:

- Creating payments.
- Retrieving payments.
- Processing payments.
- Completing payments.
- Cancelling payments.

Defines the repository interface used by the application service.

**Infrastructure Layer**

Implements persistence using PostgreSQL and pgx.

Responsible for database operations, transactions, and idempotency handling.

**Transport Layer**

Exposes the application through HTTP endpoints using Chi.

Handles incoming requests, response serialization, and HTTP status codes.

## API

Base URL:

```text
http://localhost:8080
```

### Health Check

```http
GET /health
```

Returns:

```text
ok
```

### Version

```http
GET /version
```

Returns:

```text
1.0.0
```

### Create Payment

```http
POST /payments
Content-Type: application/json
Idempotency-Key: unique-request-key
```

Request:

```json
{
  "user_id": "user-123",
  "amount": "100.50",
  "currency": "USD"
}
```

Successful response (`201 Created`):

```json
{
  "id": "generated-payment-uuid",
  "status": "PENDING",
  "amount": "100.50",
  "currency": "USD"
}
```

The `Idempotency-Key` header is required.

If the same key is reused with the same payment payload, the original payment is returned instead of creating a duplicate.

If the same key is reused with a different payload, the API returns `409 Conflict`.

### Get Payment

```http
GET /payments/{id}
```

Returns the payment with the specified identifier.

### Move Payment to Processing

```http
POST /payments/{id}/process
```

Transitions:

```text
PENDING → PROCESSING
```

### Complete Payment

```http
POST /payments/{id}/complete
```

Transitions:

```text
PROCESSING → COMPLETED
```

### Cancel Payment

```http
POST /payments/{id}/cancel
```

Transitions:

```text
PENDING / PROCESSING → CANCELLED
```

Invalid state transitions return `409 Conflict`.

## Getting Started

### Prerequisites

- Go 1.26.2 or compatible version.
- PostgreSQL.
- Git.

### 1. Clone the repository

```bash
git clone https://github.com/YOUR_USERNAME/payment-processing-platform.git

cd payment-processing-platform
```

### 2. Configure environment variables

Create a `.env` file in the project root:

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/payments?sslmode=disable
TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/payments_test?sslmode=disable
```

Adjust the credentials and database names to match your PostgreSQL configuration.

The `.env` file should never be committed to Git.

### 3. Create databases

Create the application and test databases:

```sql
CREATE DATABASE payments;
CREATE DATABASE payments_test;
```

### 4. Apply database migrations

Run the migrations in order:

```bash
psql "$DATABASE_URL" -f migrations/000001_create_payments.up.sql

psql "$DATABASE_URL" -f migrations/000002_create_idempotency_keys.up.sql
```

Apply the same migrations to the test database if running integration tests.

### 5. Install dependencies

```bash
go mod download
```

### 6. Run the application

```bash
go run ./cmd/api
```

The HTTP server starts on:

```text
localhost:8080
```

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## Engineering Goals

This project is being developed incrementally, with an emphasis on understanding the engineering decisions behind each component rather than simply implementing features.

Areas of focus include:

- Domain modeling and business invariants.
- Separation of concerns and dependency boundaries.
- Database transactions and data consistency.
- Idempotency and safe request retries.
- Error handling and observability.
- Automated testing.
- Production-oriented Go development.

## Roadmap

Planned areas of development:

- [ ] Improve request validation and API error responses.
- [ ] Add integration tests with PostgreSQL.
- [ ] Add database migration tooling.
- [ ] Add Docker support and Docker Compose.
- [ ] Add configuration management.
- [ ] Improve observability and metrics.
- [ ] Add asynchronous payment processing.
- [ ] Explore Kafka-based event-driven architecture.
- [ ] Deploy the application to Kubernetes.
- [ ] Explore cloud deployment and AWS infrastructure.

---

**This project is a work in progress and is intended for educational and engineering portfolio purposes.**