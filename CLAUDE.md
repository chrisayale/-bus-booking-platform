# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Bus Booking Platform** — A full-stack bus ticket booking system with passenger (B2C) and business owner (B2B) interfaces.

- **Frontend**: React 19, TypeScript 6, Vite 6, Redux Toolkit, React Router 7, Tailwind CSS 4
- **Backend**: Go 1.26, Gin web framework, PostgreSQL 16 (via sqlx)
- **Auth**: JWT (access + refresh tokens), role-based access control (passenger, owner, platform_admin, agent, controller)
- **Payments**: FlexPay integration with WhatsApp-based payment flow
- **Infrastructure**: Docker Compose (PostgreSQL, Go API, Nginx frontend)

## Repository Structure

```
-bus-booking-platform/
├── BE-bus-booking-platform/          # Backend (Go)
│   ├── cmd/api/main.go               # Entry point, routes, middleware setup
│   ├── internal/
│   │   ├── config/config.go          # Environment config loading
│   │   ├── database/
│   │   │   ├── database.go           # DB connection & migrations
│   │   │   └── migrations/           # 18 SQL migration files (001-018)
│   │   ├── handlers/
│   │   │   ├── b2c.go                # Passenger endpoints (search, book, tickets)
│   │   │   ├── b2b.go                # Owner endpoints (buses, routes, journeys, scan)
│   │   │   ├── guest.go              # Guest checkout
│   │   │   ├── analytics.go          # Dashboard stats
│   │   │   ├── payment.go            # FlexPay webhooks
│   │   │   └── admin.go              # Platform admin endpoints
│   │   ├── middleware/
│   │   │   ├── auth.go               # JWT auth + role middleware
│   │   │   └── ratelimit.go          # Rate limiting for auth endpoints
│   │   ├── models/models.go          # Data structures (User, Bus, Route, Journey, Booking, Ticket)
│   │   ├── payments/                 # FlexPay provider implementation
│   │   ├── services/                 # Email, storage (PG/S3), QR generation
│   │   ├── utils/                    # JWT, password hashing, QR utils
│   │   └── whatsapp/                 # WhatsApp payment flow handler (state machine)
│   ├── seed/
│   │   └── init.sql                  # Auto-run demo data on first DB start
│   ├── seed_setup.sql                # Manual seed (org, routes, buses, users)
│   ├── seed_update.sql               # Manual seed (journeys for fixed dates)
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── go.mod / go.sum
│   └── .env.example                  # Environment variable template
│
├── FE-bus-booking-platform/          # Frontend (React)
│   ├── src/
│   │   ├── api/client.ts             # HTTP client with automatic token refresh
│   │   ├── components/Layout.tsx     # Navigation layout (mobile bottom nav, desktop sidebar)
│   │   ├── context/LanguageContext   # i18n (English, Arabic, French) with RTL support
│   │   ├── hooks/useAuth.ts          # Auth state management
│   │   ├── store/                    # Redux slices (auth, bookings, ui)
│   │   ├── types/                    # TypeScript interfaces matching backend
│   │   ├── pages/
│   │   │   ├── Landing.tsx           # Marketing landing page
│   │   │   ├── Auth.tsx              # Login/Register
│   │   │   ├── b2c/                  # Passenger pages (Search, Booking, MyBookings)
│   │   │   └── b2b/                  # Owner pages (Dashboard, Buses, Routes, Journeys, Counter, QRScan, Admins, Revenue, Passengers)
│   │   └── lib/                      # PDF generation (jsPDF), utils
│   ├── Dockerfile
│   ├── nginx.conf                    # SPA routing + API proxy
│   ├── package.json
│   ├── vite.config.ts                # Dev proxy to localhost:8080
│   ├── tailwind.config.js
│   └── tsconfig.json
│
└── README.md                         # Project overview, quick start, API docs
```

## Common Development Commands

### Backend (Go)

```bash
cd -bus-booking-platform/BE-bus-booking-platform

# Install dependencies
go mod tidy

# Run locally (requires PostgreSQL on localhost:5432 or Docker)
go run ./cmd/api

# Run with Docker Compose (recommended)
docker compose up -d

# View API logs
docker compose logs -f api

# Run migrations manually
docker exec -i be-bus-booking-platform-db-1 psql -U postgres -d bus_booking < internal/database/migrations/001_initial_schema.sql

# Run seed manually
docker exec -i be-bus-booking-platform-db-1 psql -U postgres -d bus_booking < seed/init.sql

# Build binary
go build -o api ./cmd/api

# Test build
go build ./...
```

### Frontend (React)

```bash
cd -bus-booking-platform/FE-bus-booking-platform

# Install dependencies
npm install

# Development server (port 5173, proxies /api to localhost:8080)
npm run dev

# Build for production
npm run build

# Lint
npm run lint

# Preview production build
npm run preview

# Run with Docker Compose (from project root)
docker compose up -d frontend
```

### Docker Compose (Full Stack)

```bash
cd -bus-booking-platform/BE-bus-booking-platform

# Start all services
docker compose up -d

# Stop all services
docker compose down

# Reset everything (deletes database)
docker compose down -v && docker compose up -d

# View all logs
docker compose logs -f
```

## Environment Variables

Backend (`.env` in `BE-bus-booking-platform/`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/bus_booking?sslmode=disable` | PostgreSQL connection |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `JWT_EXPIRY` | `15m` | Access token lifetime |
| `REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `RATE_LIMIT_REQUESTS` | `20` | Auth endpoint rate limit |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit window |
| `SMTP_HOST` | `smtp.mailtrap.io` | SMTP server |
| `SMTP_PORT` | `587` | SMTP port |
| `SMTP_USERNAME` | — | SMTP username |
| `SMTP_PASSWORD` | — | SMTP password |
| `SMTP_FROM` | `noreply@busbooking.com` | From email |
| `AWS_BUCKET` | — | S3 bucket for bus images |
| `AWS_REGION` | `us-east-1` | AWS region |
| `APP_URL` | `http://localhost:5173` | Frontend URL (for email links) |
| `FLEXPAY_BASE_URL` | `https://backend.flexpay.cd` | FlexPay API base |
| `FLEXPAY_AUTHORIZATION` | — | FlexPay JWT token |
| `FLEXPAY_MERCHANT` | `KIVUGC` | FlexPay merchant ID |

## Key Architecture Patterns

### Backend

1. **Handler Registration**: All routes defined in `cmd/api/main.go` with middleware chains
2. **Authentication**: JWT tokens with claims (user_id, email, role, org_id) validated in `middleware/auth.go`
3. **Role-Based Access**: `RoleMiddleware("role1", "role2")` — roles: `passenger`, `owner`, `platform_admin`, `agent`, `controller`
4. **Database**: sqlx with embedded migrations (`database/migrations/*.sql`) auto-run on startup
5. **Seed Data**: `seed/init.sql` runs automatically if `organizations` table is empty
6. **Services**: Email (SMTP), Storage (PG BYTEA or S3), QR code generation
7. **WhatsApp Payments**: Complex state machine in `internal/whatsapp/handler.go` for FlexPay integration

### Frontend

1. **API Client**: `src/api/client.ts` — Axios-like wrapper with automatic token refresh on 401
2. **Auth State**: `src/hooks/useAuth.ts` + Redux `authSlice` — persists tokens in localStorage
3. **Routing**: React Router 7 with protected routes (`PrivateRoute`, `OwnerRoute`, `PlatformAdminRoute`)
3. **i18n**: `LanguageContext` with `en`, `ar`, `fr` — RTL support for Arabic
4. **Layout**: `Layout.tsx` — responsive (mobile bottom nav, desktop sidebar), role-aware navigation
5. **State**: Redux Toolkit slices for auth, bookings, UI (loading, toasts)

## API Endpoints Summary

### Public
- `GET /api/trips/search?origin=...&destination=...&date=...` — Search journeys
- `GET /api/journeys/:id/seats` — Get booked seats for seat map
- `POST /api/auth/login` — Login (returns access + refresh tokens)
- `POST /api/auth/register` — Register (creates org if role=owner)
- `POST /api/guest/checkout` — Guest booking

### Authenticated (Passenger)
- `GET /api/user/profile` — Get profile
- `PUT /api/user/profile` — Update profile
- `GET /api/bookings` — My bookings
- `POST /api/bookings` — Create booking
- `DELETE /api/tickets/:id` — Cancel ticket
- `GET /api/tickets/download/:token` — Download PDF ticket

### Owner (Organization)
- `GET /api/buses` / `POST /api/buses` — Fleet management
- `GET /api/routes` / `POST /api/routes` — Route management
- `GET /api/journeys` / `POST /api/journeys` — Journey scheduling
- `PUT /api/journeys/:id/status` — Update journey status
- `POST /api/scan-ticket` — Validate QR ticket
- `GET /api/tickets` — Organization tickets
- `GET /api/analytics/dashboard` — Dashboard stats
- `GET /api/analytics/report` — Revenue report
- `POST /api/admin-users` — Manage org admins

### Platform Admin
- `GET /api/platform-admin/pending-owners` — Approve/reject owner registrations
- `GET /api/platform-admin/agencies` — Manage agencies
- `GET /api/admin/users` — User management

### Webhooks
- `POST /api/payments/flexpay/callback` — FlexPay payment callback
- `POST /api/payments/flexpay/approve` — Manual approval
- `POST /api/payments/flexpay/cancel` — Cancel payment
- `POST /api/payments/flexpay/decline` — Decline payment

## Database Schema (Key Tables)

- **organizations** — Bus companies (B2B)
- **users** — Passengers, owners, admins, agents, controllers (role-based)
- **buses** — Fleet vehicles with seat configuration (JSON)
- **routes** — Origin→destination with intermediate stops
- **stops** — Intermediate stops on routes with pricing
- **journeys** — Scheduled trips (route + bus + datetime + status)
- **bookings** — Purchase records (user or guest)
- **tickets** — Individual seats on journeys with QR tokens
- **exchange_rates** — Currency conversion rates

## Important Notes

### WhatsApp Payment Handler
The `internal/whatsapp/handler.go` contains a complex state machine for FlexPay payments via WhatsApp. Key states:
- `StepLanguageChoice` → `StepWelcome` → `StepMainMenu` → `StepFlexPayAmount` → `StepFlexPayPhoneInput` → `StepPaymentProcessing`
- Fixed syntax error (unexpected EOF) by adding `StepFlexPayPhoneInput` case and removing unreachable cases
- Payment timeout worker runs in background (`StartPaymentTimeoutWorker`)

### Token Refresh Flow
1. Access token (15m) expires → API returns 401
2. Frontend `api/client.ts` intercepts 401, calls `POST /api/auth/refresh` with refresh token
3. On success, retries original request with new access token
4. On failure, clears auth state, redirects to login

### Seed Data
Demo accounts created by `seed/init.sql`:
- **Owner**: `admin@busbook.com` / `admin123` (role: owner)
- **Passenger**: `passenger@test.com` / `admin123` (role: passenger)

### Ports
| Service | Docker Port | Local Port |
|---------|-------------|------------|
| PostgreSQL | 5432 | 5433 |
| API | 8080 | 8081 |
| Frontend | 80 | 3000 |
| Frontend Dev | — | 5173 |

## Testing

No formal test suite currently exists. Manual testing via:
- Swagger/OpenAPI: Not configured
- Postman/curl: Use test accounts above
- Frontend: `npm run dev` → http://localhost:5173

## Troubleshooting

1. **Build errors**: Run `go mod tidy` (backend) or `npm install` (frontend)
2. **Database connection**: Ensure PostgreSQL is running and `DATABASE_URL` is correct
3. **Token issues**: Check `JWT_SECRET` matches between services
4. **Frontend proxy**: Vite proxies `/api` to `localhost:8080` — ensure backend runs on 8080
5. **WhatsApp**: Requires valid FlexPay credentials and WhatsApp Business API setup
6. **Migrations**: Run automatically on startup; check logs for "migration error (may be OK if already applied)"

## File Locations for Common Tasks

| Task | File |
|------|------|
| Add new API endpoint | `cmd/api/main.go` + handler in `internal/handlers/` |
| Add database column | `internal/database/migrations/XXX_description.sql` |
| Modify auth logic | `internal/middleware/auth.go`, `internal/utils/jwt.go` |
| Add frontend page | `src/pages/b2c/` or `src/pages/b2b/` + route in `App.tsx` |
| Modify UI components | `src/components/Layout.tsx` |
| Change styling | `tailwind.config.js`, `src/index.css`, component files |
| Update i18n | `src/context/LanguageContext.tsx`, `src/locales/*.json` |