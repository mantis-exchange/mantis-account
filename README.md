# mantis-account

User account, authentication, and balance management for [Mantis Exchange](https://github.com/mantis-exchange).

## Features

- **Registration** with bcrypt password hashing
- **JWT login** with configurable expiry
- **TOTP 2FA** (enable/verify/disable)
- **API key** generation
- **Balance management** — available/frozen per asset, freeze/unfreeze/credit/deduct
- **Faucet** — testnet token distribution

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/account/register` | No | Register |
| POST | `/api/v1/account/login` | No | Login → JWT |
| GET | `/api/v1/account` | JWT | Profile |
| GET | `/api/v1/account/balances` | JWT | Balances |
| POST | `/api/v1/account/api-keys` | JWT | Generate API keys |
| POST | `/api/v1/account/totp/enable` | JWT | Enable 2FA |
| POST | `/faucet` | JWT | Request test tokens |

### Internal (service-to-service, no auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/internal/v1/balance/freeze` | Freeze balance |
| POST | `/internal/v1/balance/unfreeze` | Unfreeze balance |
| POST | `/internal/v1/balance/credit` | Credit balance |
| POST | `/internal/v1/balance/deduct-frozen` | Deduct frozen |

## Quick Start

```bash
go build -o mantis-account ./cmd/account
DB_URL=postgres://mantis:mantis@localhost:5432/mantis ./mantis-account
```

## Part of [Mantis Exchange](https://github.com/mantis-exchange)

MIT License
