# 🎫 Ticket System — Go Backend Intern Assignment

> A REST API for a ticket system with JWT auth and strict ownership rules — built in Go, containerized with Docker, and deployed live on Render.

**🔗 Live API:** https://ticket-system-ix75.onrender.com
**🔗 Health check:** https://ticket-system-ix75.onrender.com/health
**📦 Repo:** https://github.com/BhanuNidumolu/Ticket-System

![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)
![Deployed](https://img.shields.io/badge/deployed-Render-46E3B7)
![Dependencies](https://img.shields.io/badge/external%20deps-zero-brightgreen)

---

## 👋 What this is

This is my submission for the Backend Intern assignment — build a small ticket
system where users can register, log in, create tickets, and manage the
status of *only their own* tickets. Two-day timeline, JWT auth, Docker,
real deployment. No admin roles, no fancy features — just clean,
correct, ownership-safe REST APIs.

I went in wanting three things to be true by the end: **the auth flow is
actually secure, ownership checks can't be bypassed, and the whole thing
runs the same locally and in production.** I think I got there — details
below.

## ✨ Features

- [x] User registration with hashed passwords (no plaintext, ever)
- [x] JWT-based login (`Authorization: Bearer <token>`)
- [x] Create / list / get tickets — scoped strictly to the logged-in user
- [x] Enforced status flow: `open → in_progress → closed`, and **closed
      tickets can never be reopened**
- [x] Ownership violations return `404`, not `403` — so you can't even
      tell whether a ticket ID exists if it isn't yours
- [x] Fully Dockerized, one command to build and run
- [x] Deployed and publicly reachable (see link above)
- [x] Zero third-party Go dependencies (explained below — this was a
      deliberate choice, not me avoiding `go get`!)

## 🧠 Design decisions (and why)

**Why zero external dependencies?**
I could've reached for `golang-jwt/jwt`, `gin`, and `golang.org/x/crypto/bcrypt`
in about five minutes. I chose not to, on purpose:
- Go 1.22 shipped method+path routing straight into `net/http.ServeMux`
  (`"PATCH /tickets/{id}/status"`), so a router library wasn't buying me
  much.
- I wrote a minimal HS256 JWT encoder/decoder by hand
  (`internal/auth/jwt.go`) using just `crypto/hmac` + `crypto/sha256`.
  It was genuinely a great way to actually understand what a JWT *is*
  instead of just calling a library.
- For password hashing I implemented PBKDF2-HMAC-SHA256 from scratch
  (100,000 iterations, random salt per user) instead of bcrypt, since
  bcrypt isn't in the standard library. It's a legitimate, well-known
  KDF — just not the one everyone defaults to.
- **Practical upside:** `go build` and `docker build` never depend on
  reaching `proxy.golang.org`. No flaky module downloads mid-build,
  ever. On a free-tier deploy pipeline that's one less thing to break.

If I were shipping this for real traffic tomorrow, bcrypt and a
battle-tested JWT lib would be the safer call — I say exactly that in
the roadmap below. This was a "prove I understand the primitives"
choice for an assignment, not a "reinvent crypto forever" philosophy.

**Why in-memory storage?**
The brief explicitly allows it, and the two-day scope is tight. I did
design the `Store` as an interface in both `internal/user` and
`internal/ticket` specifically so a SQLite or Postgres implementation
is a drop-in swap later — the handlers never touch storage details
directly.

**Why 404 instead of 403 on ownership mismatch?**
If ticket `abc123` belongs to another user and I return `403 Forbidden`,
I've just confirmed to an attacker that `abc123` *exists*. Returning
`404 Not Found` for both "doesn't exist" and "not yours" leaks nothing.
Small detail, but it's the kind of thing I wanted to get right rather
than just pass the happy-path tests.

## 🏗️ Architecture

```
cmd/server/main.go     → wiring: config, routes, middleware, server start
internal/auth/         → JWT issue/verify, password hashing, auth middleware
internal/user/         → user model, in-memory store, register/login handlers
internal/ticket/       → ticket model, store, status machine, CRUD handlers
internal/httpx/        → shared JSON request/response helpers
internal/idgen/        → random ID generation
```

Layering is deliberately boring: `handler → store interface → in-memory
impl`. Nothing clever, nothing hidden — I wanted this to be easy for
someone reviewing it to trace in five minutes.

## 🚀 Getting started

### Run locally with Go

```bash
go run ./cmd/server
```

### Run with Docker (this is the required contract)

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-string ticket-system
curl http://localhost:8080/health
```

Expected response:
```json
{"status": "ok"}
```

### Environment variables

See `.env.example`. Everything has a sane default except that you
should **always set `JWT_SECRET` explicitly** outside of quick local
testing — without it, a random secret is generated per-process, which
means every restart invalidates all previously issued tokens.

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

<details>
<summary><b>Register</b></summary>

```bash
curl -X POST https://ticket-system-ix75.onrender.com/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"password123"}'
```
</details>

<details>
<summary><b>Login</b></summary>

```bash
curl -X POST https://ticket-system-ix75.onrender.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"password123"}'
# => {"token": "..."}
```
</details>

<details>
<summary><b>Create a ticket</b></summary>

```bash
curl -X POST https://ticket-system-ix75.onrender.com/tickets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Printer on fire","description":"Send help"}'
```
</details>

<details>
<summary><b>List your tickets</b></summary>

```bash
curl https://ticket-system-ix75.onrender.com/tickets \
  -H "Authorization: Bearer <token>"
```
</details>

<details>
<summary><b>Update ticket status</b></summary>

```bash
curl -X PATCH https://ticket-system-ix75.onrender.com/tickets/<id>/status \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

Valid flow: `open → in_progress → closed`. `closed` is terminal —
anything else returns `400 Bad Request`.
</details>

## ☁️ Deployment

Deployed on **[Render](https://render.com)** — free Web Service, Docker
runtime, zero cost.

1. Pushed this repo to GitHub.
2. Render → New → Web Service → connected the repo → environment:
   **Docker** (Render auto-detects the `Dockerfile`, no extra config).
3. Set `JWT_SECRET` as an environment variable in the Render dashboard.
4. Deployed. Render assigns a public HTTPS URL — `/health` is reachable
   with no auth, as required.

⚠️ Heads-up: Render's free tier spins the instance down after ~15 min
of inactivity, so the *first* request after idle time can take 20-30
seconds to wake up. Don't be alarmed if the first `curl` looks slow —
it's cold-starting, not broken.

## 🧪 What I tested manually

Before calling this done, I walked the full flow end-to-end: register →
duplicate-email rejection → login → wrong-password rejection → create
ticket → list → get by ID → `open → in_progress → closed` transitions →
confirmed a closed ticket can't reopen → confirmed a second user gets
`404` trying to touch the first user's ticket. All of it matches the
contract.

## 🤔 Assumptions I made

- Storage is in-memory (allowed by the brief) — data resets on
  restart. The `Store` interface makes swapping to SQLite/Postgres a
  contained change, not a rewrite.
- Emails are normalized to lowercase before storage/lookup.
- Passwords need to be at least 8 characters (not specified in the
  brief, but shipping with *no* minimum felt wrong).
- No refresh-token flow — a single JWT with a configurable TTL
  (`TOKEN_TTL_MINUTES`, defaults to 60).

## 🛣️ If I had more time (roadmap)

I kept scope tight on purpose per the brief's "don't over-engineer"
note, but here's what I'd do next, roughly in priority order:

1. **Automated tests** — table-driven unit tests for the status
   machine and ownership checks, plus a handful of `httptest`-based
   integration tests hitting the handlers directly.
2. **Swap in SQLite** — the `Store` interface is already there for it;
   this would make data survive a restart/redeploy.
3. **CI via GitHub Actions** — `go vet`, `go build`, and the test suite
   running on every push, so the README's claims are enforced, not just
   asserted.
4. **Swap the hand-rolled crypto for standard libraries** — bcrypt
   (`golang.org/x/crypto/bcrypt`) and `golang-jwt/jwt` for production
   use, keeping my versions as a fallback/learning reference.
5. **Structured logging** (`log/slog`) instead of plain `log.Printf`,
   and a request ID per request for traceability.
6. **Pagination** on `GET /tickets` once ticket volume isn't trivial.
7. **OpenAPI/Swagger spec** so the contract is machine-readable, not
   just documented in this README.
8. **Graceful shutdown** (`context` + `http.Server.Shutdown`) so
   in-flight requests aren't dropped on redeploy.

---
