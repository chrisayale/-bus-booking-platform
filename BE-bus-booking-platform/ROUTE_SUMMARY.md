# API Routes Restoration Summary

## Problem
The application was experiencing 404 errors on API routes due to missing route definitions in `cmd/api/main.go` after refactoring.

## Solution
Restored all missing API routes in `cmd/api/main.go` by:

1. **Initializing all handler instances** with correct dependencies:
   - B2BHandler (with storage service)
   - B2CHandler (with email service)
   - GuestHandler
   - AnalyticsHandler
   - PaymentHandler
   - AdminHandler

2. **Setting up middleware**:
   - CORS middleware with proper configuration
   - Custom rate limiter for auth endpoints
   - JWT authentication middleware
   - Role-based middleware (admin, owner, platform_admin)

3. **Registering all missing API routes**:

### Auth Endpoints (`/api/auth`)
- POST `/register` - User registration
- POST `/login` - User login
- POST `/refresh` - Token refresh
- POST `/forgot-password` - Password reset initiation
- POST `/reset-password` - Password reset completion
- GET `/verify-email` - Email verification

### User Profile Endpoints (`/api/user`)
- GET `/profile` - Get user profile
- PUT `/profile` - Update user profile
- PUT `/password` - Change password
- DELETE `/account` - Delete user account

### Email Verification Endpoints (`/api/email`)
- POST `/verification` - Send verification email
- GET `/verification` - Verify email token

### Search Endpoints (`/api`)
- GET `/search/trips` - Search for available journeys

### Booking Endpoints (`/api/booking`)
- GET `/journeys/:id/seats` - Get seat availability for a journey
- POST `` - Create a new booking

### Ticket Management Endpoints (`/api/ticket`)
- PUT `/:id` - Update ticket details
- DELETE `/:id` - Cancel ticket
- GET `/download/:token` - Download ticket PDF

### Booking Management Endpoints (`/api/bookings`)
- GET `` - Get user's bookings
- DELETE `/:id` - Delete a booking
- POST `/:id/cancel` - Cancel a booking

### Guest Checkout (`/api`)
- POST `/guest/checkout` - Guest checkout (no auth required)

### Admin Endpoints (`/api/admin`)
- GET `/dashboard` - Get dashboard statistics
- GET `/analytics` - Get revenue report
- GET `/exchange-rate` - Get exchange rate
- PUT `/exchange-rate` - Update exchange rate

#### User Management (`/api/admin/users`) - platform_admin only
- GET `` - List users
- POST `` - Create user
- GET `/:id` - Get user by ID
- PUT `/:id` - Update user
- DELETE `/:id` - Delete user

### B2B Endpoints (`/api/b2b`) - owner authentication required

#### Bus Management (`/api/b2b/buses`)
- POST `` - Create bus
- GET `` - List buses
- GET `/:id` - Get bus by ID
- PUT `/:id` - Update bus
- DELETE `/:id` - Delete bus
- POST `/:id/image` - Upload bus image

#### Route Management (`/api/b2b/routes`)
- POST `` - Create route
- GET `` - List routes
- GET `/:id` - Get route by ID
- DELETE `/:id` - Delete route

#### Journey Management (`/api/b2b/journeys`)
- POST `` - Create journey
- GET `` - List journeys
- PUT `/:id/status` - Update journey status

#### Admin User Management (`/api/b2b/admin-users`)
- POST `` - Create admin user
- GET `` - List admin users
- DELETE `/:id` - Delete admin user

#### Platform Admin Functions (`/api/b2b/platform-admin`) - platform_admin only
- GET `/pending-owners` - List pending owner approvals
- PUT `/pending-owners/:id/approve` - Approve owner
- PUT `/pending-owners/:id/reject` - Reject owner
- GET `/agencies` - List agencies
- GET `/pending-agencies` - List pending agencies
- PUT `/pending-agencies/:id/approve` - Approve agency
- PUT `/pending-agencies/:id/reject` - Reject agency
- PUT `/agencies/:id/active` - Set agency active status

#### Organization-Specific Reports (`/api/b2b`)
- GET `/organization/tickets` - List organization tickets
- POST `/impersonate-owner/:orgId` - Impersonate organization owner
- POST `/scan-ticket` - Scan QR ticket

### Public Webhook Endpoint (`/api/public`)
- POST `/webhook/whatsapp` - WhatsApp webhook placeholder

### Payment Webhooks (`/api/payments/flexpay`)
- POST `/callback` - FlexPay payment callback
- POST `/approve` - FlexPay approval redirect
- POST `/cancel` - FlexPay cancel redirect
- POST `/decline` - FlexPay decline redirect

## Verification
- Successfully built the application with `go build ./cmd/api`
- Started the application and verified all routes were registered
- All 404 errors should now be resolved for the frontend and WhatsApp integration

## Dependencies Added
- Added `"github.com/bus-booking/BE-bus-booking-platform/internal/services"` import
- Removed unused `"time"` import
- Used existing internal rate limiter from `internal/middleware/ratelimit.go`