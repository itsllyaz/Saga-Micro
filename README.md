# Saga-Micro

A Go microservices e-commerce backend built around **gRPC** internal services and a single **REST/HTTP API Gateway**. It demonstrates a basic **saga (choreography-style compensation)** pattern for placing an order: the Order service reserves stock on the Product service and rolls back the order if the reservation fails.

---

## Architecture

```
                     ┌─────────────────────┐
   HTTP/JSON  ─────▶ │     API Gateway      │  (Gin, port 3000)
                     │        (cmd/)         │
                     └──────────┬───────────┘
                                │ gRPC
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐      ┌────────────────┐      ┌────────────────┐
│  Auth Service  │      │ Product Service │      │  Order Service  │
│   (:50051)     │      │    (:50052)     │      │    (:50053)     │
└───────┬────────┘      └────────┬────────┘      └────────┬────────┘
        │                        │                        │ gRPC
        ▼                        ▼                        ▼ (calls Product)
   auth_svc (Postgres)   product_svc (Postgres)     order_svc (Postgres)
```

- **API Gateway** (`cmd/`) — a Gin HTTP server that authenticates requests (via the Auth service) and proxies them over gRPC to the Auth, Product, and Order services. This is the only service a client talks to directly.
- **Auth Service** (`services/auth/`) — user & admin registration/login, password hashing, and JWT issuing/validation.
- **Product Service** (`services/product/`) — product catalog (create/list/find) and stock management.
- **Order Service** (`services/order/`) — creates orders by calling the Product service to check stock and decrement it, compensating (deleting the order) if stock reservation fails.

Each service is an independent Go module with its own `go.mod`, `Dockerfile`, `Makefile`, gRPC bindings (`pkg/pb`), config, DB layer (GORM + Postgres), and service logic.

### The saga: `CreateOrder`

1. Gateway receives `POST /order` (user must be authenticated).
2. Order service asks Product service for the product (`FindOne`) and checks stock.
3. Order service creates the `Order` row.
4. Order service calls Product service's `DecreaseStock`.
   - If that succeeds → order is `201 Created`.
   - If it fails/conflicts → the order row is **deleted** (compensating action) and an error is returned to the client.

This is the core of the "saga" in the project's name — a distributed transaction across two services with a manual compensating step instead of a 2-phase commit.

---

## Tech stack

| Concern            | Library / Tool                                   |
|---------------------|---------------------------------------------------|
| HTTP Gateway         | [Gin](https://github.com/gin-gonic/gin)            |
| Inter-service RPC    | [gRPC](https://grpc.io/) + Protocol Buffers        |
| Config               | [Viper](https://github.com/spf13/viper) (`.env` files) |
| ORM / DB             | [GORM](https://gorm.io/) + PostgreSQL              |
| Auth                 | [golang-jwt](https://github.com/golang-jwt/jwt), bcrypt password hashing |
| Containerization      | Docker + Docker Compose                            |
| CI/CD                | GitHub Actions (unit tests, ECR build/deploy)      |

---

## Repository layout

```
Saga-Micro/
├── cmd/                      # API Gateway (Gin, HTTP, port 3000)
│   ├── pkg/auth/             #   proxies to Auth service + JWT middleware
│   ├── pkg/order/            #   proxies to Order service
│   ├── pkg/product/          #   proxies to Product service
│   └── pkg/config/           #   gateway env config
│
├── services/
│   ├── auth/                 # Auth microservice (gRPC, port 50051)
│   ├── product/               # Product microservice (gRPC, port 50052)
│   └── order/                 # Order microservice (gRPC, port 50053)
│       each with: cmd/main.go, pkg/{services,models,db,config,pb}
│
├── docker-compose.yml        # spins up all 4 services + 3 Postgres instances
├── Makefile                  # top-level build/run helpers (Windows-oriented)
├── main.go / application/    # ⚠️ legacy scaffold, not part of the microservices (see below)
└── .github/workflows/        # CI (test.yml) and CD (deploy.yml, pushes to ECR)
```

---

## Prerequisites

- Go 1.19+ (root module declares 1.26.3, but the services/gateway modules target 1.19)
- Docker & Docker Compose (recommended, easiest way to run everything)
- `protoc` + the Go/gRPC plugins, only if you plan to regenerate `.proto` files
- PostgreSQL, if you want to run services outside of Docker

---

## Running with Docker Compose (recommended)

The compose file builds/pulls 4 app images and 3 Postgres databases (one per service that needs one).

```bash
docker compose up
```

This starts:

| Service       | Port (host) | Notes                                   |
|---------------|-------------|------------------------------------------|
| api-gateway   | 3000        | public entry point                       |
| auth-svc      | 50051       | gRPC, backed by `auth-db` (5433 → 5432)  |
| product-svc   | 50052       | gRPC, backed by `product-db` (5434 → 5432) |
| order-svc     | 50053       | gRPC, backed by `order-db` (5435 → 5432) |

> **Note:** the images referenced in `docker-compose.yml` (`amalmadhu06/ecom-*`) are not this repo's own images — you'll likely want to build local images first (see [Known Issues](#known-issues--gotchas)).

---

## Running services locally (without Docker)

Each service is its own Go module. You'll need a local Postgres instance and to create the three databases (`auth_svc`, `product_svc`, `order_svc`), matching the `dev.env` files under each service's `pkg/config/envs/`.

```bash
# Auth service
cd services/auth
go run cmd/main.go        # listens on :50051

# Product service
cd services/product
go run cmd/main.go        # listens on :50052

# Order service
cd services/order
go run cmd/main.go        # listens on :50053
```

Each service reads its config via Viper from `pkg/config/envs/dev.env` (or real environment variables, since `viper.AutomaticEnv()` is enabled):

**Auth service**
```
PORT=:50051
DB_URL=postgres://postgres:postgres@localhost:5432/auth_svc
JWT_SECRET_KEY=h28dh582fcu390
```

**Product service**
```
PORT=:50052
DB_URL=postgres://postgres:postgres@localhost:5432/product_svc
```

**Order service**
```
PORT=:50053
DB_URL=postgres://postgres:postgres@localhost:5432/order_svc
PRODUCT_SVC_URL=localhost:50052
```

**API Gateway** (`cmd/pkg/config/envs/dev.env`)
```
PORT=:3000
AUTH_SUV_URL=localhost:50051
PRODUCT_SUV_URL=localhost:50052
ORDER_SUV_URL=localhost:50053
```

The gateway currently has no committed `main.go` wiring these routes together at `cmd/` — see [Known Issues](#known-issues--gotchas) for what's needed to finish it.

---

## API (via the Gateway)

| Method | Path            | Auth required | Description                       |
|--------|-----------------|----------------|-------------------------------------|
| POST   | `/auth/register`| —              | Register a new user                |
| POST   | `/auth/login`   | —              | Log in, returns a JWT              |
| POST   | `/admin/login`  | —              | Admin login, returns a JWT         |
| GET    | `/product/`     | —              | List all in-stock products          |
| GET    | `/product/:id`  | —              | Get a single product                |
| POST   | `/product/`     | Admin (Bearer) | Create a product                    |
| POST   | `/order/`       | User (Bearer)  | Create an order (runs the saga)     |

Authenticated routes expect `Authorization: Bearer <token>`, validated by the gateway against the Auth service.

Example — create an order:

```bash
curl -X POST http://localhost:3000/order/ \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"productId": 1, "quantity": 1}'
```

---
