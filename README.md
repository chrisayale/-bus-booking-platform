# 🚌 Bus Booking Platform

A full-stack bus ticket booking system with passenger (B2C) and business owner (B2B) interfaces.

## ✨ Features

### For Passengers (B2C)
- **Search trips** — Find buses by origin, destination, and date
- **Interactive seat map** — Select your seats visually
- **Book tickets** — Create bookings with passenger details
- **QR tickets** — Each ticket has a QR code for check-in
- **Download PDF** — Save tickets as PDF for offline access
- **Guest checkout** — Book without creating an account

### For Business Owners (B2B)
- **Dashboard** — View analytics (revenue, occupancy, popular routes)
- **Fleet management** — Add/edit/delete buses with driver assignments
- **Route management** — Create routes with intermediate stops
- **Journey scheduling** — Schedule bus departures with status tracking
- **QR scanner** — Validate passenger tickets via camera or manual entry
- **Admin management** — Add/remove admin users for your organization

### Tech Highlights
- 🌐 **Multi-language** — English, Arabic, French (RTL support)
- 🔄 **Real-time seat updates** — Live polling every 15 seconds
- 📱 **Mobile responsive** — Full mobile layout with bottom navigation
- 🎨 **Modern UI** — Tailwind CSS with smooth animations

## 🏗 Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | React 19, TypeScript 6, Vite 6, Redux Toolkit, React Router 7 |
| **Styling** | Tailwind CSS 4, Lucide React icons, Recharts |
| **Backend** | Go 1.25, Gin web framework |
| **Database** | PostgreSQL 16 (via sqlx) |
| **Auth** | JWT (access + refresh tokens) |
| **Infrastructure** | Docker Compose |

## 🚀 Quick Start

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)

### Run the app

```bash
# 1. Clone the repository
git clone <your-repo-url>
cd <project-directory>/BE-bus-booking-platform

# 2. Start all services
docker compose up -d

# 3. Open the app
open http://localhost:3000
```

The database is automatically seeded with demo data on first start.

### Test Accounts

| Role | Email | Password |
|------|-------|----------|
| 🛡️ **Owner** | `admin@busbook.com` | `admin123` |
| 🧑‍💼 **Passenger** | `passenger@test.com` | `admin123` |

## 📂 Project Structure

```
BE-bus-booking-platform/          # Backend (Go)
├── cmd/api/main.go               # Entry point, routes
├── internal/
│   ├── config/config.go          # Environment config
│   ├── database/
│   │   ├── database.go           # DB connection & migrations
│   │   └── migrations/           # SQL migration files
│   ├── handlers/                 # API handlers
│   │   ├── b2c.go                # Passenger endpoints
│   │   ├── b2b.go                # Owner endpoints
│   │   ├── guest.go              # Guest checkout
│   │   └── analytics.go          # Dashboard stats
│   ├── middleware/
│   │   ├── auth.go               # JWT authentication
│   │   └── ratelimit.go          # Rate limiting
│   ├── models/models.go          # Data structures
│   ├── services/                 # Email, storage, tokens
│   └── utils/                    # JWT, password, QR
├── seed/
│   └── init.sql                  # Auto-run on first DB start
├── seed_setup.sql                # Manual seed (org, routes, buses, users)
├── seed_update.sql                # Manual seed (journeys for fixed dates)
├── Dockerfile
└── docker-compose.yml

FE-bus-booking-platform/          # Frontend (React)
├── src/
│   ├── api/client.ts             # HTTP client with token refresh
│   ├── components/Layout.tsx     # Navigation layout
│   ├── context/LanguageContext   # i18n (en, ar, fr)
│   ├── hooks/useAuth.ts          # Auth state hook
│   ├── store/                    # Redux slices
│   ├── types/                    # TypeScript interfaces
│   ├── pages/
│   │   ├── Landing.tsx           # Landing page
│   │   ├── Auth.tsx              # Login/Register
│   │   ├── b2c/                  # Passenger pages
│   │   └── b2b/                  # Owner pages
│   └── lib/                      # PDF generation, utils
├── Dockerfile
└── nginx.conf
```

## 🔧 Environment Variables

The backend supports these environment variables (see `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/bus_booking?sslmode=disable` | PostgreSQL connection |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `JWT_EXPIRY` | `15m` | Access token lifetime |
| `REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `SMTP_HOST` | `smtp.mailtrap.io` | SMTP server (email) |
| `SMTP_PORT` | `587` | SMTP port |
| `AWS_BUCKET` | — | S3 bucket for bus images |
| `APP_URL` | `http://localhost:5173` | Frontend URL (for email links) |

## 🐳 Docker Services

| Service | Image | Port | Description |
|---------|-------|------|-------------|
| `db` | postgres:16-alpine | 5433 | PostgreSQL database |
| `api` | Go (custom) | 8080 | REST API |
| `frontend` | Nginx (custom) | 3000 | React SPA |

### Useful docker commands

```bash
# View logs
docker compose logs -f api

# Run seed manually (if needed)
docker exec -i be-bus-booking-platform-db-1 psql -U postgres -d bus_booking < seed/init.sql

# Stop everything
docker compose down

# Reset everything (deletes database)
docker compose down -v && docker compose up -d
```

## 🌐 API Overview

### Public endpoints
- `GET /api/trips/search?origin=...&destination=...&date=...` — Search journeys
- `GET /api/journeys/:id/seats` — Get booked seats for a journey
- `POST /api/auth/login` — Login
- `POST /api/auth/register` — Register
- `POST /api/guest/checkout` — Guest booking

### Authenticated (passenger)
- `GET /api/bookings` — My bookings
- `POST /api/bookings` — Create booking
- `PATCH /api/tickets/:id/cancel` — Cancel ticket

### Owner only
- `POST /api/buses` — Create bus
- `GET /api/routes` — List routes
- `POST /api/journeys` — Create journey
- `POST /api/scan` — Validate QR ticket
- `PATCH /api/journeys/:id/status` — Update journey status
- `GET /api/analytics/dashboard` — Dashboard stats

## 📝 License

MIT
