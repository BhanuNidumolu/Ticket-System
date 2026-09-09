# 🎫 Ticket System — My Go Backend Intern Assignment

Hey! 👋 This is my submission for the Backend Intern assignment — a REST API
for a ticket system, built in Go, with JWT auth, strict ownership rules, a
Docker image, and a live deployment. I had two days, and I wanted to spend
them on getting the *fundamentals* rock solid rather than bolting on extra
features — so no admin roles, no ticket assignment, no comments. Just a
clean, correct, secure API that does exactly what it says on the tin.

**🔗 Live API:** https://ticket-system-ix75.onrender.com
**🔗 Health check:** https://ticket-system-ix75.onrender.com/health
**📦 Repo:** https://github.com/BhanuNidumolu/Ticket-System

![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)
![Deployed](https://img.shields.io/badge/deployed-Render-46E3B7)
![Dependencies](https://img.shields.io/badge/external%20deps-zero-brightgreen)
![Tests](https://img.shields.io/badge/tests-passing-success)

---

## 👋 What I set out to build

The brief was simple to state and (deceptively) easy to get subtly wrong:
register, login, create tickets, only ever see *your own* tickets, and move
a ticket through `open → in_progress → closed` without letting it sneak
backwards. Sounds small. It isn't — ownership checks and auth are exactly
the kind of thing that look done at a glance and fall apart under a real
test suite.

So I set myself three non-negotiables before writing a single handler:

1. **The auth flow has to be genuinely secure** — hashed passwords, signed
   and verified JWTs, no shortcuts.
2. **Ownership can't be bypassed, ever** — not through a clever ID guess,
   not through a missing check I forgot to write.
3. **It has to run identically locally and in production** — if it works on
   my machine but not in Docker on Render, it doesn't count as working.

Everything in this README is tested against the **live deployed instance**,
not just `localhost`, because that's the whole point of the "deployment
readiness" part of the brief.

## ✨ Features

- User registration with hashed passwords (never, ever plaintext)
- JWT-based login (`Authorization: Bearer <token>`)
- Create / list / get tickets — scoped strictly to the logged-in user
- Enforced status flow: `open → in_progress → closed`, and **closed tickets
  can never be reopened**
- Ownership violations return `404`, not `403` — so an attacker can't even
  learn whether a ticket ID exists if it isn't theirs
- Thread-safe in-memory storage (`sync.RWMutex`-guarded)
- Fully Dockerized, one command to build and run
- Deployed and publicly reachable (link above — go ahead and hit `/health`)
- A real automated test suite (`go test ./... -v`), not just manual curl
  checks — 25 tests, all green
- Zero third-party Go dependencies — a deliberate choice, not me avoiding
  `go get` (explained below, I promise it's a good reason)

## 🛠️ Tech Stack

| | |
|---|---|
| **Language** | Go 1.22+ |
| **HTTP** | `net/http` (stdlib routing, no framework) |
| **Auth** | Hand-rolled JWT, HS256 |
| **Password hashing** | PBKDF2-HMAC-SHA256 |
| **Storage** | In-memory (interface-based, swappable) |
| **Containerization** | Docker, multi-stage build |
| **Deployment** | Render (free Docker Web Service) |
| **Dependencies** | Go standard library only |

## 🧠 Design decisions (and the "why" behind each)

**Why zero external dependencies?**
I could've reached for `golang-jwt/jwt`, `gin`, and `golang.org/x/crypto/bcrypt`
and been done in five minutes. I chose not to, on purpose:

- Go 1.22 shipped method+path routing straight into `net/http.ServeMux`
  (`"PATCH /tickets/{id}/status"`), so a router library wasn't buying me much.
- I wrote a minimal HS256 JWT encoder/decoder by hand
  (`internal/auth/jwt.go`) using just `crypto/hmac` + `crypto/sha256`. It was
  genuinely the best way to actually *understand* what a JWT is, instead of
  just importing a library and trusting it blindly.
- For password hashing I implemented PBKDF2-HMAC-SHA256 from scratch
  (100,000 iterations, random 16-byte salt, 32-byte derived key) instead of
  bcrypt, since bcrypt isn't in the standard library. It's a legitimate,
  well-known KDF — just not the default everyone reaches for.
- **Practical bonus:** `go build` and `docker build` never need to reach
  `proxy.golang.org`. No flaky module downloads mid-build, one less thing to
  break on a free-tier deploy pipeline.

To be clear: if I were shipping this for real production traffic tomorrow,
bcrypt and a battle-tested JWT library would be the safer call (see
Roadmap). This was a "prove I understand the primitives" choice for a
two-day assignment, not "I distrust every library forever."

**Why in-memory storage?**
The brief explicitly allows it, and two days is tight. But I still designed
`Store` as an interface in both `internal/user` and `internal/ticket`
specifically so a SQLite or Postgres implementation is a drop-in swap
later — handlers never touch storage details directly.

**Why 404 instead of 403 on an ownership mismatch?**
If ticket `abc123` belongs to someone else and I return `403 Forbidden`,
I've just confirmed to an attacker that `abc123` *exists*. Returning
`404 Not Found` for both "doesn't exist" and "not yours" leaks nothing.
Small detail, but it's the kind of detail that separates "it works" from
"it's actually secure."

**Why sort tickets by creation time?**
Go's map iteration order is randomized on purpose. `GET /tickets` sorts
newest-first explicitly rather than relying on map order, so two identical
requests always return the same order.

## 🏗️ Project Structure

```text
ticket-system/
│
├── cmd/
│   └── server/
│       └── main.go              # wiring: config, routes, middleware, server start
│
├── internal/
│   ├── auth/
│   │   ├── jwt.go                # HS256 sign/verify, stdlib only
│   │   ├── middleware.go         # Bearer token auth middleware
│   │   └── password.go           # PBKDF2-HMAC-SHA256 hashing
│   │
│   ├── httpx/
│   │   └── json.go               # shared JSON request/response helpers
│   │
│   ├── idgen/
│   │   └── idgen.go              # random ID generation
│   │
│   ├── ticket/
│   │   ├── handler.go            # create/list/get/status handlers
│   │   ├── model.go              # ticket struct + status machine
│   │   └── store.go              # in-memory ticket store
│   │
│   └── user/
│       ├── handler.go            # register/login handlers
│       ├── model.go
│       └── store.go
│
├── .env.example
├── .gitignore
├── Dockerfile
├── go.mod
└── README.md
```

I kept the layering deliberately boring: `handler → store interface →
in-memory impl`. Nothing clever, nothing hidden — a reviewer should be able
to trace any request end-to-end in about five minutes.

## 🚀 Getting Started

### Requirements

- Go 1.22+ for local development
- Docker (optional, but this is the required grading contract)

### Run locally with Go

```bash
go run ./cmd/server
```

Runs on `http://localhost:8080` by default.

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Run with Docker (the required contract)

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-string ticket-system
curl http://localhost:8080/health
```

## ⚙️ Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the HTTP server listens on |
| `JWT_SECRET` | random per process | Secret used to sign JWTs |
| `TOKEN_TTL_MINUTES` | `60` | JWT token lifetime, in minutes |

```env
PORT=8080
JWT_SECRET=replace-with-a-long-random-secret
TOKEN_TTL_MINUTES=60
```

`JWT_SECRET` should **always** be explicitly set in a deployed environment.
If it's not provided, a random secret is generated at startup — meaning
every restart invalidates every previously issued token.

## 📡 API Reference

All bodies are JSON. Protected routes need `Authorization: Bearer <token>`.

| Method | Endpoint | Auth | Purpose |
|---|---|---|---|
| GET | `/health` | – | Health check |
| POST | `/auth/register` | – | Register a new user |
| POST | `/auth/login` | – | Log in, get a JWT back |
| POST | `/tickets` | ✅ | Create a ticket |
| GET | `/tickets` | ✅ | List *your* tickets |
| GET | `/tickets/{id}` | ✅ | Get one of *your* tickets |
| PATCH | `/tickets/{id}/status` | ✅ | Update status of *your* ticket |

### Register

```bash
curl -X POST https://ticket-system-ix75.onrender.com/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

```json
{
  "id": "255c4e6951a89baa190b2845089a670b",
  "email": "user@example.com",
  "created_at": "2026-09-09T17:13:20.448183386Z"
}
```

Validation: email must be a valid format (normalized to lowercase),
password must be at least 8 characters. Registering an already-used email
returns `409 Conflict`.

### Login

```bash
curl -X POST https://ticket-system-ix75.onrender.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

```json
{ "token": "YOUR_JWT_TOKEN" }
```

### Create a ticket

```bash
curl -X POST https://ticket-system-ix75.onrender.com/tickets \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Login issue","description":"I am unable to log in to my account."}'
```

```json
{
  "id": "116123bd7289207ff6c2e3a037a07655",
  "user_id": "255c4e6951a89baa190b2845089a670b",
  "title": "Login issue",
  "description": "I am unable to log in to my account.",
  "status": "open",
  "created_at": "2026-09-09T17:15:49.550466093Z",
  "updated_at": "2026-09-09T17:15:49.550466093Z"
}
```

New tickets always start as `open`.

### List your tickets

```bash
curl https://ticket-system-ix75.onrender.com/tickets \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Only returns tickets owned by the authenticated user.

### Get a ticket

```bash
curl https://ticket-system-ix75.onrender.com/tickets/TICKET_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

If the ticket doesn't exist, or belongs to another user:

```json
{ "error": "ticket not found" }
```

→ `404 Not Found` (never `403` — see design decisions above for why)

### Update ticket status

```bash
curl -X PATCH https://ticket-system-ix75.onrender.com/tickets/TICKET_ID/status \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

```json
{
  "id": "116123bd7289207ff6c2e3a037a07655",
  "user_id": "255c4e6951a89baa190b2845089a670b",
  "title": "Deployment test",
  "description": "Testing the deployed ticket API",
  "status": "in_progress",
  "created_at": "2026-09-09T17:15:49.550466093Z",
  "updated_at": "2026-09-09T17:20:41.431321282Z"
}
```

## 🔁 Ticket Status Flow

```text
open → in_progress → closed
```

`closed` is terminal.

| Transition | Allowed |
|---|---|
| `open → in_progress` | ✅ |
| `in_progress → closed` | ✅ |
| `open → closed` | ❌ |
| `in_progress → open` | ❌ |
| `closed → open` | ❌ |
| `closed → in_progress` | ❌ |

Any disallowed transition returns `400 Bad Request`, e.g.:

```json
{ "error": "invalid status transition from closed to open" }
```

## ❗ Error Catalog

Every error response uses the same envelope: `{"error": "message"}`.

| Situation | Status | Body |
|---|---|---|
| Missing `Authorization` header | 401 | `{"error": "missing Authorization header"}` |
| Invalid or expired token | 401 | `{"error": "invalid or expired token"}` |
| Invalid status value | 400 | `{"error": "status must be one of: open, in_progress, closed"}` |
| Disallowed status transition | 400 | `{"error": "invalid status transition from closed to open"}` |
| Ticket missing or not owned | 404 | `{"error": "ticket not found"}` |
| Duplicate email at registration | 409 | `{"error": "email already registered"}` |

## 🔐 Password Security

Passwords are never stored as plaintext:

```text
PBKDF2-HMAC-SHA256, 100,000 iterations, 16-byte random salt, 32-byte derived key
```

Stored format:

```text
pbkdf2-sha256$<iterations>$<salt-hex>$<hash-hex>
```

Verification uses a constant-time comparison (`crypto/subtle`) so a
malicious client can't learn anything from response timing.

## 🔑 JWT Authentication

Hand-implemented HS256 using `crypto/hmac` + `crypto/sha256` + base64url
encoding. Claims:

```json
{ "sub": "user-id", "iat": 1788974057, "exp": 1788977657 }
```

Lifetime controlled by `TOKEN_TTL_MINUTES` (default 60).

## 💾 Storage

In-memory, thread-safe (`sync.RWMutex`), via `user.MemoryStore` and
`ticket.MemoryStore`.

**Limitation:** data does not persist — a restart means all users and
tickets are lost. This is explicitly allowed by the assignment brief, and
the `Store` interface is deliberately kept separate from the HTTP handlers
so a SQLite/Postgres implementation can be swapped in later without
touching a single handler.

## 🐳 Docker

Multi-stage build:

- **Build stage:** `golang:1.23-alpine`, compiled with
  `CGO_ENABLED=0 GOOS=linux` → fully static binary
- **Runtime stage:** `alpine:3.20` — small, no libc dependency issues

## ☁️ Deployment

Deployed on **[Render](https://render.com)** — free Docker Web Service.

1. Pushed the project to GitHub.
2. Created a new Web Service on Render, connected the repo.
3. Selected Docker as the runtime — Render auto-detects the `Dockerfile`.
4. Set `JWT_SECRET` as an environment variable.
5. Deployed.
6. Tested `/health`.
7. Tested registration and login.
8. Tested the protected ticket endpoints end-to-end.

Render injects its own `PORT` env var, which the app reads automatically
(`getEnv("PORT", "8080")`).

⚠️ Free tier spins the instance down after ~15 min idle — the first request
after that can take 20-30 seconds to wake back up. Not broken, just cold-starting!

## 🧪 Testing

I didn't want to just *say* this works — I wanted to prove it. The project
ships with a real automated suite covering:

- JWT generation and validation
- Expired and tampered JWT rejection
- Password hashing and verification (including "two users, same password,
  different hashes")
- Ticket status transitions (every legal and illegal move in the matrix)
- Ticket ownership and access control (cross-user access attempts)
- Authentication requirements on every protected route
- Request validation (missing title, invalid status, etc.)
- Full ticket lifecycle: create, list, get, and status updates

Run it yourself:

```bash
go test ./... -v
```

Latest run — **25 tests, all passing**:

```text
=== RUN   TestGenerateAndParseToken_RoundTrip
--- PASS: TestGenerateAndParseToken_RoundTrip (0.00s)
=== RUN   TestParseToken_WrongSecretRejected
--- PASS: TestParseToken_WrongSecretRejected (0.00s)
=== RUN   TestParseToken_ExpiredTokenRejected
--- PASS: TestParseToken_ExpiredTokenRejected (0.00s)
=== RUN   TestParseToken_MalformedTokenRejected
--- PASS: TestParseToken_MalformedTokenRejected (0.00s)
=== RUN   TestParseToken_TamperedPayloadRejected
--- PASS: TestParseToken_TamperedPayloadRejected (0.00s)
=== RUN   TestHashPassword_ProducesExpectedFormat
--- PASS: TestHashPassword_ProducesExpectedFormat (0.02s)
=== RUN   TestHashPassword_NeverStoresPlaintext
--- PASS: TestHashPassword_NeverStoresPlaintext (0.01s)
=== RUN   TestVerifyPassword_CorrectPassword
--- PASS: TestVerifyPassword_CorrectPassword (0.03s)
=== RUN   TestVerifyPassword_WrongPassword
--- PASS: TestVerifyPassword_WrongPassword (0.03s)
=== RUN   TestVerifyPassword_TwoUsersSamePassword_DifferentHashes
--- PASS: TestVerifyPassword_TwoUsersSamePassword_DifferentHashes (0.03s)
=== RUN   TestVerifyPassword_MalformedHash
--- PASS: TestVerifyPassword_MalformedHash (0.00s)
PASS
ok      ticket-system/internal/auth     (cached)
=== RUN   TestStatus_Valid
--- PASS: TestStatus_Valid (0.00s)
=== RUN   TestCanTransition
    --- PASS: TestCanTransition/open_to_in_progress (0.00s)
    --- PASS: TestCanTransition/in_progress_to_closed (0.00s)
    --- PASS: TestCanTransition/open_to_closed_directly (0.00s)
    --- PASS: TestCanTransition/in_progress_back_to_open (0.00s)
    --- PASS: TestCanTransition/closed_to_open (0.00s)
    --- PASS: TestCanTransition/closed_to_in_progress (0.00s)
    --- PASS: TestCanTransition/closed_to_closed (0.00s)
    --- PASS: TestCanTransition/open_to_open (0.00s)
    --- PASS: TestCanTransition/unknown_from_status (0.00s)
=== RUN   TestTicketLifecycle_CreateListGetUpdate
--- PASS: TestTicketLifecycle_CreateListGetUpdate (0.03s)
=== RUN   TestTicketOwnership_CannotAccessAnotherUsersTicket
--- PASS: TestTicketOwnership_CannotAccessAnotherUsersTicket (0.05s)
=== RUN   TestTicketEndpoints_RequireAuth
--- PASS: TestTicketEndpoints_RequireAuth (0.00s)
=== RUN   TestCreateTicket_MissingTitleRejected
--- PASS: TestCreateTicket_MissingTitleRejected (0.03s)
=== RUN   TestUpdateStatus_InvalidStatusValueRejected
--- PASS: TestUpdateStatus_InvalidStatusValueRejected (0.03s)
PASS
ok      ticket-system/internal/ticket   0.154s
PASS
ok      ticket-system/internal/user
```

## 🧪 Production Testing

I also verified the full flow against the **live deployed URL**, not just
`localhost`:

```text
Health Check → Register → Login → Receive JWT → Create Ticket
  → List Tickets → Get Ticket → Update Status (open → in_progress)
  → Confirm closed tickets can't reopen → Confirm cross-user 404
```

Every step returned the expected status code and payload.

## 🤔 Assumptions

- Storage is in-memory (allowed by the brief) — data resets on restart. The
  `Store` interface makes swapping to SQLite/Postgres a contained change,
  not a rewrite.
- Emails are normalized to lowercase before storage/lookup.
- Passwords need to be at least 8 characters (not specified in the brief,
  but shipping with no minimum felt wrong to me).
- No refresh-token flow — a single JWT with a configurable TTL, since the
  brief didn't ask for refresh tokens and I'd rather ship a correct simple
  thing than a half-finished complex one.

## 🛣️ If I had more time (roadmap)

I kept scope tight on purpose, per the brief's "don't over-engineer" note.
If this were going further, here's what I'd tackle next, roughly in
priority order:

1. **SQLite persistence** — the `Store` interface is already built for it;
   this would make data survive a restart/redeploy.
2. **CI via GitHub Actions** — `go vet`, `go build`, and the test suite on
   every push, so this README's claims are enforced, not just asserted.
3. **Swap hand-rolled crypto for standard libraries** — bcrypt and
   `golang-jwt/jwt` for production use, keeping my versions around as a
   documented learning reference.
4. **Structured logging** (`log/slog`) with a request ID per request, so a
   real production incident is actually debuggable.
5. **Pagination** on `GET /tickets` once ticket volume isn't trivial.
6. **OpenAPI/Swagger spec** so the API contract is machine-readable, not
   just documented in prose.
7. **Graceful shutdown** so in-flight requests aren't dropped mid-redeploy.
8. Role-based authorization, ticket search/filtering, tighter request
   validation, and real production monitoring.

---

Thanks for reading this far — I had a genuinely good time building this.
If you have questions about any of the design choices above, happy to walk
through the reasoning on a call. 🙂
