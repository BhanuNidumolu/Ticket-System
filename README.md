# Ticket System (Go)

A small backend service where a user can register, log in, create tickets,
view only their own tickets, and update the status of their own tickets.

Built with the Go standard library only — no external Go modules — so
`go build` / `docker build` work fully offline with no module-proxy
dependency.

## Tech notes

- **Router**: Go 1.22+ `net/http.ServeMux` with method + path-parameter
  patterns (`"GET /tickets/{id}"`), no third-party router.
- **JWT**: hand-rolled HS256 implementation (`internal/auth/jwt.go`) using
  `crypto/hmac` / `crypto/sha256`.
- **Password hashing**: PBKDF2-HMAC-SHA256, 100,000 iterations, random
  16-byte salt per user (`internal/auth/password.go`). Not bcrypt, because
  bcrypt lives outside the standard library (`golang.org/x/crypto`); this
  keeps the module dependency-free. Swapping to bcrypt later only touches
  that one file.
- **Storage**: in-memory, guarded by `sync.RWMutex`. Data does not
  survive a restart — see "Assumptions" below. The `Store` interfaces in
  `internal/user` and `internal/ticket` are designed so a SQLite/Postgres
  implementation can be dropped in without touching handler code.

## Project layout

```
cmd/server/main.go       - wiring: config, routes, server start
internal/auth            - JWT issuing/parsing, password hashing, auth middleware
internal/user            - user model, store, register/login handlers
internal/ticket          - ticket model, store, CRUD + status handlers
internal/httpx           - shared JSON request/response helpers
internal/idgen           - random ID generation
```

## Local run (Go directly)

```bash
go run ./cmd/server
```

## Local run (Docker) — matches the required contract

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-string ticket-system
curl http://localhost:8080/health
```

Expected health response:

```json
{"status": "ok"}
```

## Environment variables

See `.env.example`. None are required to boot (a random `JWT_SECRET` is
generated per-process if unset), but you should always set `JWT_SECRET`
explicitly outside of local testing — otherwise every restart invalidates
all previously issued tokens.

## Deployed application

- **Deployed URL**: `<fill in after deploying>`
- **Public health check**: `<fill in>/health`

Deployed to [Render](https://render.com) (free Web Service, Docker
runtime). Steps:

1. Push this repo to GitHub.
2. On Render: New → Web Service → connect the repo → environment:
   **Docker** (Render auto-detects the `Dockerfile`).
3. Add environment variable `JWT_SECRET` (generate a long random value).
4. Deploy. Render assigns a public `https://<service>.onrender.com` URL;
   `/health` is reachable there with no auth.

Note: Render's free tier spins the instance down after inactivity, so
the first request after idle time may take a few seconds (cold start).

## API

All request/response bodies are JSON. Protected endpoints require
`Authorization: Bearer <token>`.

| Method | Endpoint | Auth | Purpose |
|---|---|---|---|
| GET | `/health` | — | Health check |
| POST | `/auth/register` | — | Register user |
| POST | `/auth/login` | — | Login, returns JWT |
| POST | `/tickets` | ✅ | Create ticket |
| GET | `/tickets` | ✅ | List your tickets |
| GET | `/tickets/{id}` | ✅ | Get one of your tickets |
| PATCH | `/tickets/{id}/status` | ✅ | Update status of your ticket |

### Register

```bash
curl -X POST localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"a@example.com","password":"password123"}'
```

### Login

```bash
curl -X POST localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"a@example.com","password":"password123"}'
# => {"token": "..."}
```

### Create a ticket

```bash
curl -X POST localhost:8080/tickets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Printer on fire","description":"Send help"}'
```

### List your tickets

```bash
curl localhost:8080/tickets -H "Authorization: Bearer <token>"
```

### Update status

```bash
curl -X PATCH localhost:8080/tickets/<id>/status \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

Valid flow: `open -> in_progress -> closed`. `closed` is terminal — any
transition out of it, or any transition not in that sequence, returns
`400 Bad Request`.

## Assumptions

- Storage is in-memory by design (per the assignment's "in-memory,
  SQLite, or Postgres" allowance); all data is lost on restart. Swapping
  to SQLite is a matter of writing a new `Store` implementation.
- Requesting `GET /tickets/{id}` or `PATCH /tickets/{id}/status` for a
  ticket that exists but belongs to another user returns `404 Not Found`
  (not `403`), to avoid leaking whether a given ticket ID exists.
- Email is treated case-insensitively and normalized to lowercase at
  registration/login.
- Passwords must be at least 8 characters (not specified in the brief;
  a reasonable minimal validation).
- No refresh-token flow — a single JWT with a configurable TTL
  (`TOKEN_TTL_MINUTES`, default 60 minutes).
