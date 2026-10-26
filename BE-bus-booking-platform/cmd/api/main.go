package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bus-booking/BE-bus-booking-platform/internal/config"
	"github.com/bus-booking/BE-bus-booking-platform/internal/database"
	"github.com/bus-booking/BE-bus-booking-platform/internal/handlers"
	"github.com/bus-booking/BE-bus-booking-platform/internal/middleware"
	"github.com/bus-booking/BE-bus-booking-platform/internal/payments"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/whatsapp"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// FORCE REBUILD 2026-10-26-v3 - IMMEDIATE STARTUP LOG - confirms binary is running
	log.Println("[MAIN] ===== SERVER STARTING - MAIN() ENTRY POINT =====")

	// Load .env file if it exists
	godotenv.Load()

	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to WhatsApp
	whatsapp.DB = db
	whatsapp.QRSecret = cfg.JWTSecret
	whatsapp.AppURL = cfg.AppURL
	log.Printf("[MAIN] AppURL set to: %s", whatsapp.AppURL) // Debug logging
	whatsapp.FlexPay = payments.NewFlexPayProvider()
	whatsapp.StartPaymentTimeoutWorker()

	// Check database URL before starting WhatsApp
	// Fix for PgBouncer/Neon pooler: add statement_cache_mode=describe
	whatsappDBURL := cfg.DatabaseURL
	if !strings.Contains(whatsappDBURL, "statement_cache_mode=") {
		if strings.Contains(whatsappDBURL, "?") {
			whatsappDBURL += "&statement_cache_mode=describe"
		} else {
			whatsappDBURL += "?statement_cache_mode=describe"
		}
	}
	log.Printf("[MAIN] DatabaseURL for WhatsApp (with PgBouncer param): %s", maskDatabaseURL(whatsappDBURL))

	// Initialize WhatsApp in BACKGROUND GOROUTINE so HTTP server starts immediately for health checks
	log.Println("[MAIN] 🚀 LAUNCHING WhatsApp initialization goroutine...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[WHATSAPP-INIT] ❌ PANIC: %v", r)
			}
		}()
		log.Println("[WHATSAPP-INIT] Starting WhatsApp connection in background...")
		if err := whatsapp.Connect(whatsappDBURL); err != nil {
			log.Printf("[WHATSAPP-INIT] ❌ ERROR: %v", err)
			log.Printf("[WHATSAPP-INIT] ⚠️  WhatsApp will not be available. Check logs above for QR code or connection errors.")
		} else {
			log.Println("[WHATSAPP-INIT] ✅ WhatsApp connection established successfully")
		}
	}()

	// Small sleep to allow goroutine to start and log first message
	time.Sleep(100 * time.Millisecond)
	log.Println("[MAIN] ✓ WhatsApp goroutine launched")
	if err := database.RunMigrations(db); err != nil {
		log.Printf("Warning: migration error (may be OK if already applied): %v", err)
	}

	// Clean up test agencies and accounts (except super admin) for fresh registrations
	// ONLY run in non-production environments to prevent accidental data loss
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := database.CleanupTestData(db); err != nil {
			log.Printf("Warning: failed to cleanup test data: %v", err)
		}
	} else {
		log.Println("[MAIN] Skipping CleanupTestData in production environment")
	}

	// Ensure super admin account exists on every startup (idempotent UPSERT)
	if err := database.EnsureSuperAdmin(db); err != nil {
		log.Printf("Warning: failed to ensure super admin account: %v", err)
	}

	// Initialize services
	emailSvc := services.NewEmailService(
		cfg.SMTPHost, cfg.SMTPPort,
		cfg.SMTPUsername, cfg.SMTPPassword,
		cfg.SMTPFrom, cfg.AppURL,
	)

	// Storage service (PostgreSQL BYTEA by default, S3 as production option)
	var storageSvc services.StorageService
	if cfg.AWSBucket != "" {
		s3, err := services.NewS3Storage(cfg.AWSRegion, cfg.AWSBucket, "bus-images/")
		if err != nil {
			log.Printf("Warning: S3 not available, falling back to PG storage: %v", err)
		} else {
			storageSvc = s3
		}
	}
	if storageSvc == nil {
		storageSvc = services.NewPgStorage(db, cfg.AppURL)
	}

	// Initialize handlers
	b2bHandler := handlers.NewB2BHandler(db, storageSvc, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
	b2cHandler := handlers.NewB2CHandler(db, cfg.JWTSecret, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, emailSvc)
	guestHandler := handlers.NewGuestHandler(db, cfg.JWTSecret, emailSvc, cfg.AppURL)
	analyticsHandler := handlers.NewAnalyticsHandler(db)
	paymentHandler := handlers.NewPaymentHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	// Rate limiter for auth endpoints (login, register, forgot-password, etc.)
	authRateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.CORSAllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	r.Use(func(c *gin.Context) {
		corsMiddleware.HandlerFunc(c.Writer, c.Request)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// WhatsApp QR code endpoint
	r.GET("/api/whatsapp/qr", func(c *gin.Context) {
		qr := whatsapp.GetCurrentQR()
		client := whatsapp.Client
		if qr == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":           "QR code not available",
				"client_nil":      client == nil,
				"qr_nil":          true,
				"status":          "QR code not yet generated - check WhatsApp initialization logs",
			})
			return
		}
		c.Header("Content-Type", "image/png")
		c.Data(http.StatusOK, "image/png", qr)
	})

	// WhatsApp status endpoint for diagnostics
	r.GET("/api/whatsapp/status", func(c *gin.Context) {
		if whatsapp.Client == nil {
			c.JSON(http.StatusOK, gin.H{
				"connected":  false,
				"logged_in":  false,
				"jid":        "",
				"error":      "WhatsApp client not initialized",
			})
			return
		}
		jid := ""
		if whatsapp.Client.Store.ID != nil {
			jid = whatsapp.Client.Store.ID.String()
		}
		c.JSON(http.StatusOK, gin.H{
			"connected":  whatsapp.Client.IsConnected(),
			"logged_in":  whatsapp.Client.IsLoggedIn(),
			"jid":        jid,
		})
	})

	// Payment webhooks
	api := r.Group("/api")
	{
		api.POST("/payments/flexpay/callback", paymentHandler.FlexPayCallback)
		api.POST("/payments/flexpay/approve", paymentHandler.FlexPayApprove)
		api.POST("/payments/flexpay/cancel", paymentHandler.FlexPayCancel)
		api.POST("/payments/flexpay/decline", paymentHandler.FlexPayDecline)

		// Auth routes (rate-limited)
		auth := api.Group("/auth")
		auth.Use(authRateLimiter.Middleware())
		{
			auth.POST("/register", b2cHandler.Register)
			auth.POST("/login", b2cHandler.Login)
			auth.POST("/refresh", b2cHandler.RefreshToken)
			auth.POST("/forgot-password", b2cHandler.ForgotPassword)
			auth.POST("/reset-password", b2cHandler.ResetPassword)
		}
		// Auth GET (verify-email) - no rate limit needed
		api.GET("/auth/verify-email", b2cHandler.VerifyEmail)

		// Auth profile endpoint (alias for /api/user/profile) - requires auth
		authProfile := api.Group("/auth")
		authProfile.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			authProfile.GET("/profile", b2cHandler.GetProfile)
		}

		// User profile routes (require auth)
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			user.GET("/profile", b2cHandler.GetProfile)
			user.PUT("/profile", b2cHandler.UpdateProfile)
			user.PUT("/password", b2cHandler.ChangePassword)
			user.DELETE("/account", b2cHandler.DeleteAccount)
		}

		// Email verification routes
		email := api.Group("/email")
		{
			email.POST("/verification", b2cHandler.SendVerificationEmail)
			email.GET("/verification", b2cHandler.VerifyEmail)
		}

		// Search routes
		api.GET("/search/trips", b2cHandler.SearchTrips)

		// Booking routes
		booking := api.Group("/booking")
		booking.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			booking.GET("/journeys/:id/seats", b2cHandler.GetJourneySeats)
			booking.POST("", b2cHandler.BookTickets)
		}

		// Ticket management routes
		ticket := api.Group("/ticket")
		ticket.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			ticket.PUT("/:id", b2cHandler.UpdateTicket)
			ticket.DELETE("/:id", b2cHandler.CancelTicket)
			ticket.GET("/download/:token", b2cHandler.DownloadTicket)
		}

		// Booking management routes
		bookingMgmt := api.Group("/bookings")
		bookingMgmt.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			bookingMgmt.GET("", b2cHandler.GetUserBookings)
			bookingMgmt.DELETE("/:id", b2cHandler.DeleteBooking)
			bookingMgmt.POST("/:id/cancel", b2cHandler.CancelBooking)
		}

		// Guest checkout (no auth required)
		api.POST("/guest/checkout", guestHandler.GuestCheckout)

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		admin.Use(middleware.AdminMiddleware())
		{
			// Dashboard and analytics
			admin.GET("/dashboard", analyticsHandler.GetDashboardStats)
			admin.GET("/analytics", analyticsHandler.GetRevenueReport)

			// Exchange rate management
			admin.GET("/exchange-rate", adminHandler.GetExchangeRate)
			admin.PUT("/exchange-rate", adminHandler.UpdateExchangeRate)

			// Agency management
			admin.GET("/agencies", b2bHandler.ListAgencies)

			// User management (Super Admin only)
			users := admin.Group("/users")
			users.Use(middleware.RoleMiddleware("platform_admin", "super_admin"))
			{
				users.GET("", adminHandler.GetUsers)
				users.POST("", adminHandler.CreateUser)
				users.GET("/:id", adminHandler.GetUserById)
				users.PUT("/:id", adminHandler.UpdateUser)
				users.DELETE("/:id", adminHandler.DeleteUser)
				users.PUT("/:id/status", adminHandler.ToggleUserStatus)
			}

			// Placeholder for admin broadcast functionality
			admin.POST("/broadcast", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "broadcast endpoint not implemented"})
			})
		}

		// Analytics routes (require organization auth - owner, platform_admin, or super_admin)
		analytics := api.Group("/analytics")
		analytics.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		analytics.Use(middleware.RoleMiddleware("owner", "platform_admin", "super_admin"))
		{
			analytics.GET("/dashboard", analyticsHandler.GetDashboardStats)
			analytics.GET("/report", analyticsHandler.GetRevenueReport)
		}

		// B2B-aligned routes (require organization auth) - moved to root /api/ level to match frontend expectations
		b2b := api.Group("")
		b2b.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		b2b.Use(middleware.RoleMiddleware("owner"))
		{
			// Bus management
			buses := b2b.Group("/buses")
			{
				buses.POST("", b2bHandler.CreateBus)
				buses.GET("", b2bHandler.ListBuses)
				buses.GET("/:id", b2bHandler.GetBus)
				buses.PUT("/:id", b2bHandler.UpdateBus)
				buses.DELETE("/:id", b2bHandler.DeleteBus)
				buses.POST("/:id/image", b2bHandler.UploadBusImage)
			}

			// Route management
			routes := b2b.Group("/routes")
			{
				routes.POST("", b2bHandler.CreateRoute)
				routes.GET("", b2bHandler.ListRoutes)
				routes.GET("/:id", b2bHandler.GetRoute)
				routes.DELETE("/:id", b2bHandler.DeleteRoute)
			}

			// Journey management
			journeys := b2b.Group("/journeys")
			{
				journeys.POST("", b2bHandler.CreateJourney)
				journeys.GET("", b2bHandler.ListJourneys)
				journeys.PUT("/:id/status", b2bHandler.UpdateJourneyStatus)
			}

			// Ticket management (moved from /organization/tickets to match frontend expectation)
			b2b.GET("/tickets", b2bHandler.ListOrganizationTickets)
			b2b.POST("/impersonate-owner/:orgId", b2bHandler.ImpersonateOwner)

			// QR ticket scanning
			b2b.POST("/scan-ticket", b2bHandler.ScanTicket)
		}

		// Platform admin functions (platform_admin and super_admin roles required)
		platformAdmin := api.Group("/platform-admin")
		platformAdmin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		platformAdmin.Use(middleware.RoleMiddleware("platform_admin", "super_admin"))
		{
			platformAdmin.GET("/pending-owners", b2bHandler.ListPendingOwners)
			platformAdmin.PUT("/pending-owners/:id/approve", b2bHandler.ApproveOwner)
			platformAdmin.PUT("/pending-owners/:id/reject", b2bHandler.RejectOwner)
			platformAdmin.GET("/agencies", b2bHandler.ListAgencies)
			platformAdmin.GET("/pending-agencies", b2bHandler.ListPendingAgencies)
			platformAdmin.PUT("/pending-agencies/:id/approve", b2bHandler.ApproveAgency)
			platformAdmin.PUT("/pending-agencies/:id/reject", b2bHandler.RejectAgency)
			platformAdmin.PUT("/agencies/:id/active", b2bHandler.SetAgencyActive)
			// Impersonate owner as platform admin (alias for /api/impersonate-owner/:orgId)
			platformAdmin.POST("/agencies/:id/impersonate", b2bHandler.ImpersonateOwner)
		}

		// Admin user management (within organization)
		adminUsers := api.Group("/admin-users")
		adminUsers.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		adminUsers.Use(middleware.RoleMiddleware("owner"))
		{
			adminUsers.POST("", b2bHandler.CreateAdminUser)
			adminUsers.GET("", b2bHandler.ListAdminUsers)
			adminUsers.DELETE("/:id", b2bHandler.DeleteAdminUser)
		}

		// Public webhook endpoint
		public := api.Group("/public")
		{
			public.POST("/webhook/whatsapp", func(c *gin.Context) {
				// Placeholder for webhook handler
				c.JSON(http.StatusOK, gin.H{"status": "webhook endpoint not implemented"})
			})
		}
	}

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// maskDatabaseURL masks the password in database URL for logging
func maskDatabaseURL(url string) string {
	// Simple masking: replace password part with ***
	if idx := strings.Index(url, "://"); idx != -1 {
		rest := url[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx != -1 {
			userPass := rest[:atIdx]
			if colonIdx := strings.Index(userPass, ":"); colonIdx != -1 {
				return url[:idx+3] + userPass[:colonIdx+1] + "***" + rest[atIdx:]
			}
		}
	}
	return url
}