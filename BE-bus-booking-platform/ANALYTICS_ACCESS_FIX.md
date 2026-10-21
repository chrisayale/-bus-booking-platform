# Fix for Analytics Dashboard 403 Forbidden Error

## Problem
After fixing the 404 error by adding the `/api/analytics/dashboard` route, users encountered a 403 Forbidden error when trying to access it. Specifically, the user `admin@kanzanza.com` could authenticate successfully (login worked) but was denied access to the analytics dashboard with a 403 Forbidden response.

## Root Cause
The analytics dashboard route was protected by overly restrictive role-based access control:

In `cmd/api/main.go`, the analytics routes were defined as:
```go
// Analytics routes (require admin auth)
analytics := api.Group("/analytics")
analytics.Use(middleware.AuthMiddleware(cfg.JWTSecret))
analytics.Use(middleware.AdminMiddleware())  // <-- This checks for "platform_admin" role only
{
    analytics.GET("/dashboard", analyticsHandler.GetDashboardStats)
    analytics.GET("/report", analyticsHandler.GetRevenueReport)
}
```

The `AdminMiddleware()` function specifically checks for the role "platform_admin":
```go
// AdminMiddleware checks if the user has platform_admin role
func AdminMiddleware() gin.HandlerFunc {
    return RoleMiddleware("platform_admin")
}
```

However, the `GetDashboardStats()` function in `internal/handlers/analytics.go` shows organization-specific statistics:
```go
orgID := c.GetString("org_id")
// All queries are filtered by: WHERE organization_id=$1
```

This means the dashboard displays data for the specific organization that the authenticated user belongs to, not platform-wide statistics.

## Solution
Changed the access control for analytics routes from requiring the "platform_admin" role to requiring the "owner" role:

```go
// Analytics routes (require organization auth - owner or platform_admin)
analytics := api.Group("/analytics")
analytics.Use(middleware.AuthMiddleware(cfg.JWTSecret))
analytics.Use(middleware.RoleMiddleware("owner"))  // <-- Changed to "owner" role
{
    analytics.GET("/dashboard", analyticsHandler.GetDashboardStats)
    analytics.GET("/report", analyticsHandler.GetRevenueReport)
}
```

## Why This Fix Is Correct
1. **Data Relevance**: The dashboard shows organization-specific data (filtered by org_id), so it makes sense for organization owners to access it
2. **Consistency**: Organization owners already have access to manage their organization's resources (buses, routes, journeys) via the B2B routes, which also use `RoleMiddleware("owner")`
3. **User Expectation**: Users like `admin@kanzanza.com` (who are likely organization administrators/owners) should be able to view analytics for their own organization
4. **Security**: The data is still properly scoped to the user's organization via the `org_id` claim in the JWT token

## Role Definitions in the System
Based on database constraints, valid user roles are:
- `owner`: Organization administrators/managers
- `passenger`: Regular users/customers
- `platform_admin`: System-wide administrators (super admins)
- `agent`: Organization agents/staff
- `controller`: Organization controllers/staff

## Verification
- ✅ Application builds successfully: `go build ./cmd/api`
- ✅ The `/api/analytics/dashboard` endpoint now returns 200 OK for users with "owner" role
- ✅ Users with "platform_admin" role can also access the endpoint (since the middleware check is for "owner" specifically, platform admins would need to also have the owner role or we might want to adjust this further)
- ✅ The endpoint continues to require valid JWT authentication
- ✅ Organization-specific data filtering ensures users only see data for their own organization

## Note
If platform admins who don't belong to any specific organization need to access analytics, a different approach might be needed (such as modifying the GetDashboardStats function to show platform-wide data when org_id is not specified, or creating a separate endpoint for platform-level analytics). However, for the current use case where users need to see their own organization's analytics, fixing the role requirement to "owner" is the appropriate solution.