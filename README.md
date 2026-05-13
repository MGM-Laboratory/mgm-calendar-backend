# mgm-calendar-backend

Backend for the MGM Laboratory Event Calendar — a Go (chi) REST API backed
by PostgreSQL, with AWS S3 file storage and automatic seeding of Indonesian
public holidays on first boot.

## Stack

| | |
|---|---|
| Language | Go 1.23 |
| Router | [`go-chi/chi/v5`](https://github.com/go-chi/chi) |
| DB | PostgreSQL 13+ (16 in dev), accessed with [`pgx/v5`](https://github.com/jackc/pgx) |
| Migrations | [`golang-migrate/migrate/v4`](https://github.com/golang-migrate/migrate) |
| Auth | Shared admin password + HS256 JWT (8-hour TTL by default) |
| Files | AWS S3 via `aws-sdk-go-v2` |
| Recurrence | [`teambition/rrule-go`](https://github.com/teambition/rrule-go) — iCal RRULE expansion |

## Quick start (Docker)

```bash
cp .env.example .env
# edit .env: at minimum set ADMIN_PASSWORD and JWT_SECRET. For uploads,
# fill in the AWS_* / S3 vars.
docker compose up --build
```

The API listens on `http://localhost:8080`. Migrations apply on startup; on
first boot the Indonesian-holiday seeder fetches the current + next year's
public holidays in the background.

## Quick start (bare metal)

```bash
docker compose up -d db                # or run your own Postgres
export $(grep -v '^#' .env | xargs)    # load env
go mod tidy
go run ./cmd/server
```

## Environment variables

See `.env.example` for the full list with comments. The required ones are:

| Var | Notes |
|---|---|
| `DATABASE_URL` | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `ADMIN_PASSWORD` | Shared admin login. Change before deploying. |
| `JWT_SECRET` | At least 16 chars. Used to sign session tokens. |

S3 vars are optional — if unset, the `/api/admin/upload` endpoint returns
`503 uploads not configured` but the rest of the API works.

## API

### Public

| Method | Path | Notes |
|---|---|---|
| `GET` | `/api/healthz` | Liveness probe |
| `GET` | `/api/events?month=YYYY-MM` | Published events whose window overlaps the month (defaults to current month) |
| `GET` | `/api/events/{id}` | Single published event |

### Admin (requires JWT — sent as `Authorization: Bearer <token>` or the `mgm_admin_token` cookie set by the login endpoint)

| Method | Path | Notes |
|---|---|---|
| `POST` | `/api/admin/auth` | Body: `{"password": "..."}` → sets `mgm_admin_token` httpOnly cookie + returns `{token, expires_at}` |
| `POST` | `/api/admin/logout` | Clears the cookie |
| `GET`  | `/api/admin/me` | Trivial "still authenticated" probe |
| `GET`  | `/api/admin/events?month=YYYY-MM` | Includes drafts |
| `GET`  | `/api/admin/events/{id}` | Includes drafts |
| `POST` | `/api/admin/events` | Create. If `recurrence_rule` is set on the parent, child instances are materialised up to 2 years out (capped by `recurrence_end_date` if set). |
| `PUT`  | `/api/admin/events/{id}` | Update. Old children are deleted and regenerated from the (possibly new) rule. Recurring *instances* cannot be edited directly — edit the parent. |
| `DELETE` | `/api/admin/events/{id}` | Delete. Cascades to children. The handler does **not** block deletion of seeded holidays — the frontend warns; the backend trusts the admin. |
| `POST` | `/api/admin/upload` | `multipart/form-data` with `file=...` (max 100 MiB) → `{url, name, type, size}` |

### Event payload (write)

```jsonc
{
  "title": "Rapat Koordinasi",
  "category": "internal",            // national_holiday | religious_holiday | joint_holiday | internal | big_event | midterm | final | seminar
  "color": "#3a6dc5",                // optional; defaults from category
  "description_json": { "type": "doc", "content": [/* TipTap nodes */] },
  "thumbnail_url": "https://...",
  "start_datetime": "2026-05-18T09:00:00+07:00",
  "end_datetime":   "2026-05-18T10:30:00+07:00",
  "is_all_day": false,
  "location": "Lab A",
  "location_type": "physical",       // physical | online | hybrid
  "meeting_link": null,
  "dresscode": "Smart casual",
  "attendees": ["Idham", "Bu Rina"],
  "attachments": [
    { "name": "agenda.pdf", "url": "https://...", "type": "application/pdf", "size": 18432 }
  ],
  "recurrence_rule": "FREQ=WEEKLY;BYDAY=MO",  // iCal RRULE; null for non-recurring
  "recurrence_end_date": "2026-12-31",        // YYYY-MM-DD or null
  "is_published": true
}
```

## Project layout

```
cmd/server/             main.go — wiring, migrations, graceful shutdown
internal/
  config/               env loader (godotenv + os.Getenv)
  db/                   pgxpool connector
  httpx/                JSON encode/decode helpers
  middleware/           CORS, request logger, RequireAdmin JWT gate
  model/                Event, Category, LocationKind, Attachment
  repository/           pgx queries
  service/
    auth.go             password check + JWT mint/verify
    event.go            create / update / delete + recurrence expansion
    recurrence.go       RRULE -> []time.Time
    s3.go               multipart upload to S3
    holiday_seed.go     first-boot Indonesian holiday seeder
  handler/              HTTP handlers + chi router
migrations/             *.up.sql / *.down.sql, applied on startup
```

## Indonesian holiday seeding

On first boot, for the current and next calendar year, the service fetches
`HOLIDAY_API_URL?year=YYYY` (default
[`api-harilibur.vercel.app/api`](https://api-harilibur.vercel.app/api)),
classifies each entry (religious / joint / national) by name and the
`is_national_holiday` flag, and inserts them as all-day events with
`is_seeded = true`. The `seeds_log` table records which years have been
seeded so re-runs are idempotent. Set `HOLIDAY_SEED_ENABLED=false` to skip.

Seeded events are normal rows — they can be edited or deleted by admin.
The frontend is responsible for warning before deletion of seeded rows.

## Testing the API quickly

```bash
# login
curl -s -X POST http://localhost:8080/api/admin/auth \
  -H 'content-type: application/json' \
  -d '{"password":"changeme"}' | jq

# list this month
curl -s 'http://localhost:8080/api/events?month=2026-05' | jq

# create an event
TOKEN=...
curl -s -X POST http://localhost:8080/api/admin/events \
  -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{
    "title": "Seminar TipTap",
    "category": "seminar",
    "start_datetime": "2026-05-20T13:00:00+07:00",
    "end_datetime":   "2026-05-20T15:00:00+07:00",
    "is_all_day": false,
    "location_type": "hybrid",
    "location": "Aula Lab",
    "meeting_link": "https://meet.example.com/abc",
    "attachments": [],
    "is_published": true
  }' | jq
```
