# Fix for Analytics Dashboard 404 Error

## Problem
The frontend was encountering a 404 error when calling `GET /api/analytics/dashboard`:
```
GET /api/analytics/dashboard -> 404 Not Found
```

Meanwhile, the login endpoint (`POST /api/auth/login`) was working correctly, returning 200 OK.

## Root Cause
In `cmd/api/main.go`, the analytics dashboard handler was only available under the `/admin` prefix:
- `GET /api/admin/dashboard` → `analyticsHandler.GetDashboardStats`
- `GET /api/admin/analytics` → `analyticsHandler.GetRevenueReport`

However, the frontend was expecting the dashboard endpoint at `/api/analytics/dashboard`.

## Solution
Added a new route group under `/api/analytics` with proper authentication middleware to provide the endpoints the frontend expects:

### New Routes Added
- `GET /api/analytics/dashboard` → `analyticsHandler.GetDashboardStats`
- `GET /api/analytics/report` → `analyticsHandler.GetRevenueReport`

Both routes are protected by:
- `middleware.AuthMiddleware(cfg.JWTSecret)` - JWT authentication
- `middleware.AdminMiddleware()` - Role-based access control (platform_admin required)

### Handler Details
- `GetDashboardStats()` returns a `DashboardStats` struct containing:
  - Total/active buses, routes, journeys
  - Booking statistics and revenue
  - Ticket status breakdown (active, checked in, missed, cancelled)
  - Seat occupancy percentage
  - Monthly revenue trends
  - Popular routes
  - Recent journeys

- `GetRevenueReport()` returns a `RevenueReport` struct containing:
  - Total revenue
  - Revenue breakdown by bus

## Verification
- ✅ Application builds successfully with `go build ./cmd/api`
- ✅ All existing routes remain functional (backward compatibility maintained)
- ✅ New analytics routes are properly registered and protected
- ✅ Frontend should now receive 200 OK responses for `GET /api/analytics/dashboard`

## Note
The original `/admin/dashboard` and `/admin/analytics` routes are preserved for backward compatibility. The frontend can now use either:
- `/api/admin/dashboard` (original) 
- `/api/analytics/dashboard` (new, matching frontend expectation)