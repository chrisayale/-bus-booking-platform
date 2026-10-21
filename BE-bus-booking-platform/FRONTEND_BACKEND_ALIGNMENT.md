# Fix for Frontend-Backend Route Alignment

## Problem
There was a mismatch between the routes called by the frontend and those exposed by the backend:

**Frontend was calling:**
- `GET /api/buses`
- `GET /api/journeys` 
- `GET /api/routes`
- `GET /api/tickets`

**Backend was exposing (before fix):**
- `GET /api/b2b/buses`
- `GET /api/b2b/journeys`
- `GET /api/b2b/routes`
- `GET /api/b2b/organization/tickets`

This caused 404 Not Found errors when the frontend tried to access these endpoints.

## Solution
I adjusted the routing in `cmd/api/main.go` to align the backend-exposed routes with what the frontend expects by moving the B2B-related routes from under the `/api/b2b/` prefix to the root `/api/` level, while maintaining the same authentication and authorization requirements.

### Changes Made

#### Moved B2B Routes to Root `/api/` Level
**Before:**
```go
b2b := api.Group("/b2b")
b2b.Use(middleware.AuthMiddleware(cfg.JWTSecret))
b2b.Use(middleware.RoleMiddleware("owner"))
{
    // Bus management
    buses := b2b.Group("/buses")
    {
        buses.POST("", b2bHandler.CreateBus)
        buses.GET("", b2bHandler.ListBuses)      // → /api/b2b/buses
        buses.GET("/:id", b2bHandler.GetBus)     // → /api/b2b/buses/:id
        // ... etc
    }
    
    // Route management  
    routes := b2b.Group("/routes")
    {
        routes.POST("", b2bHandler.CreateRoute)
        routes.GET("", b2bHandler.ListRoutes)    // → /api/b2b/routes
        // ... etc
    }
    
    // Journey management
    journeys := b2b.Group("/journeys")
    {
        journeys.POST("", b2bHandler.CreateJourney)
        journeys.GET("", b2bHandler.ListJourneys) // → /api/b2b/journeys
        // ... etc
    }
    
    // Ticket management
    b2b.GET("/organization/tickets", b2bHandler.ListOrganizationTickets) // → /api/b2b/organization/tickets
}
```

**After:**
```go
// B2B-aligned routes (require organization auth) - moved to root /api/ level
b2b := api.Group("")
b2b.Use(middleware.AuthMiddleware(cfg.JWTSecret))
b2b.Use(middleware.RoleMiddleware("owner"))
{
    // Bus management
    buses := b2b.Group("/buses")
    {
        buses.POST("", b2bHandler.CreateBus)
        buses.GET("", b2bHandler.ListBuses)      // → /api/buses
        buses.GET("/:id", b2bHandler.GetBus)     // → /api/buses/:id
        // ... etc
    }

    // Route management
    routes := b2b.Group("/routes")
    {
        routes.POST("", b2bHandler.CreateRoute)
        routes.GET("", b2bHandler.ListRoutes)    // → /api/routes
        // ... etc
    }

    // Journey management
    journeys := b2b.Group("/journeys")
    {
        journeys.POST("", b2bHandler.CreateJourney)
        journeys.GET("", b2bHandler.ListJourneys) // → /api/journeys
        // ... etc
    }

    // Ticket management (moved to match frontend expectation)
    b2b.GET("/tickets", b2bHandler.ListOrganizationTickets) // → /api/tickets
    // ... etc
}
```

#### Preserved Authentication and Authorization
All moved routes maintain the same security requirements:
- `middleware.AuthMiddleware(cfg.JWTSecret)` - Valid JWT token required
- `middleware.RoleMiddleware("owner")` - Organization owner role required

#### Preserved Organization Scoping
Despite the URL path change, organization scoping is preserved because:
- Handler functions still extract `orgID := c.GetString("org_id")` from JWT claims
- All database queries remain filtered by `WHERE organization_id=$1`
- Users only see data for their own organization

### Routes Now Matching Frontend Expectations

| Frontend Call | Backend Endpoint | Handler Function |
|---------------|------------------|------------------|
| `GET /api/buses` | `GET /api/buses` | `b2bHandler.ListBuses` |
| `GET /api/buses/:id` | `GET /api/buses/:id` | `b2bHandler.GetBus` |
| `POST /api/buses` | `POST /api/buses` | `b2bHandler.CreateBus` |
| `PUT /api/buses/:id` | `PUT /api/buses/:id` | `b2bHandler.UpdateBus` |
| `DELETE /api/buses/:id` | `DELETE /api/buses/:id` | `b2bHandler.DeleteBus` |
| `GET /api/journeys` | `GET /api/journeys` | `b2bHandler.ListJourneys` |
| `GET /api/journeys/:id` | `GET /api/journeys/:id` | `b2bHandler.GetJourney` |
| `POST /api/journeys` | `POST /api/journeys` | `b2bHandler.CreateJourney` |
| `PUT /api/journeys/:id/status` | `PUT /api/journeys/:id/status` | `b2bHandler.UpdateJourneyStatus` |
| `GET /api/routes` | `GET /api/routes` | `b2bHandler.ListRoutes` |
| `GET /api/routes/:id` | `GET /api/routes/:id` | `b2bHandler.GetRoute` |
| `POST /api/routes` | `POST /api/routes` | `b2bHandler.CreateRoute` |
| `PUT /api/routes/:id` | `PUT /api/routes/:id` | `b2bHandler.UpdateRoute` |
| `DELETE /api/routes/:id` | `DELETE /api/routes/:id` | `b2bHandler.DeleteRoute` |
| `GET /api/tickets` | `GET /api/tickets` | `b2bHandler.ListOrganizationTickets` |
| `POST /api/scan-ticket` | `POST /api/scan-ticket` | `b2bHandler.ScanTicket` |
| `POST /api/impersonate-owner/:orgId` | `POST /api/impersonate-owner/:orgId` | `b2bHandler.ImpersonateOwner` |

### Verification
- ✅ Application builds successfully: `go build ./cmd/api` (no compilation errors)
- ✅ All routes are properly registered and accessible at the paths expected by the frontend
- ✅ Authentication and authorization requirements are preserved
- ✅ Organization data scoping remains intact (users only see their own organization's data)
- ✅ Backward compatibility consideration: The old `/api/b2b/...` paths are no longer available, but this aligns with the frontend expectations as requested

### Note on API Consistency
This change makes the API more intuitive and RESTful by:
- Using plural resource names (`/buses`, `/journeys`, `/routes`, `/tickets`) at the root API level
- Following common REST conventions where collections are accessed at `/plural-resource`
- Maintaining consistency with other API endpoints like `/auth/*`, `/user/*`, `/ticket/*`, etc.
- Aligning with what the frontend developers naturally expect to call

The frontend should now receive successful responses (200 OK) for calls to:
- `GET /api/buses`
- `GET /api/journeys` 
- `GET /api/routes`
- `GET /api/tickets`

Instead of 404 Not Found errors.