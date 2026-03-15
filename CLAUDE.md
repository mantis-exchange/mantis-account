# mantis-account

Mantis Exchange user account, authentication, and balance management service.

## Architecture

- `internal/model/user.go` — User model + PostgreSQL CRUD
- `internal/model/balance.go` — Balance model with freeze/unfreeze/credit/deduct operations
- `internal/service/auth.go` — Register, Login, JWT generation/validation, API key management
- `internal/service/balance.go` — Balance operations (used internally by order service)
- `internal/handler/handler.go` — HTTP handlers (register, login, balances)
- `internal/middleware/jwt.go` — JWT authentication middleware

## API Endpoints

- `POST /api/v1/account/register` — Register new user
- `POST /api/v1/account/login` — Login, returns JWT
- `GET /api/v1/account` — Get profile (auth required)
- `GET /api/v1/account/balances` — Get balances (auth required)

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `50053` | HTTP server port |
| `DB_URL` | `postgres://mantis:mantis@localhost:5432/mantis_account?sslmode=disable` | PostgreSQL |
| `JWT_SECRET` | `changeme` | JWT signing secret |
| `JWT_EXPIRY` | `24h` | JWT token expiry duration |

## Build & Run

```bash
go build -o mantis-account ./cmd/account
./mantis-account
```
