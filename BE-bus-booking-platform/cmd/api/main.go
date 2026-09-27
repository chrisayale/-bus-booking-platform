package main

import (
	"log"
	"net/http"

	"github.com/bus-booking/BE-bus-booking-platform/internal/config"
	"github.com/bus-booking/BE-bus-booking-platform/internal/database"
	"github.com/bus-booking/BE-bus-booking-platform/internal/whatsapp"
	"github.com/bus-booking/BE-bus-booking-platform/internal/handlers"
	"github.com/bus-booking/BE-bus-booking-platform/internal/middleware"
	"github.com/bus-booking/BE-bus-booking-platform/internal/services"
	"github.com/bus-booking/BE-bus-booking-platform/internal/payments"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
		// Connect to WhatsApp
		whatsapp.DB = db
	whatsapp.QRSecret = cfg.JWTSecret
	whatsapp.AppURL = cfg.AppURL
	whatsapp.FlexPay = payments.NewFlexPayProvider()
	whatsapp.StartPaymentTimeoutWorker()
	go func() {
		if err := whatsapp.Connect(cfg.DatabaseURL); err != nil {
			log.Printf("Warning: WhatsApp connection failed: %v", err)
		}
	}()
	if err := database.RunMigrations(db); err != nil {
		log.Printf("Warning: migration error (may be OK if already applied): %v", err)
	}

	// Auto-seed with demo data on first run
	if err := database.RunSeed(db); err != nil {
		log.Printf("Warning: seed error: %v", err)
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

	r := gin.Default()

	// CORS
	// CORS — wildcard origin is safe because auth uses Bearer tokens, not cookies.
	// AllowCredentials is omitted to comply with the CORS spec (cannot combine * with credentials).
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization"},
	})
	r.Use(func(c *gin.Context) {
		corsHandler.HandlerFunc(c.Writer, c.Request)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve bus images from database (only used with PgStorage)
	r.GET("/api/images/:filename", func(c *gin.Context) {
		if _, ok := storageSvc.(*services.PgStorage); !ok {
			c.Status(http.StatusNotFound)
			return
		}
		fileName := c.Param("filename")
		var imageData []byte
		var contentType string
		err := db.QueryRow(`SELECT image_data, content_type FROM bus_images WHERE file_name = $1`, fileName).Scan(&imageData, &contentType)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, contentType, imageData)
	})

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/api/whatsapp/qr", func(c *gin.Context) {
		png := whatsapp.GetCurrentQR()
		if png == nil {
			c.String(http.StatusNotFound, "Pas de QR code disponible pour le moment (déjà connecté ou pas encore généré)")
			return
		}
		c.Data(http.StatusOK, "image/png", png)
	})
	// Public routes (no auth required)
	api := r.Group("/api")
	{
		// Auth — rate-limited POST endpoints to prevent brute-force and spam
		authLimited := api.Group("/auth")
		authLimited.Use(authRateLimiter.Middleware())
		{
			authLimited.POST("/register", b2cHandler.Register)
			authLimited.POST("/login", b2cHandler.Login)
			authLimited.POST("/refresh", b2cHandler.RefreshToken)
			authLimited.POST("/forgot-password", b2cHandler.ForgotPassword)
			authLimited.POST("/reset-password", b2cHandler.ResetPassword)
		}
		// Auth GET (verify-email) — no rate limit needed
		api.GET("/auth/verify-email", b2cHandler.VerifyEmail)

		// Search (no auth required)
		api.GET("/trips/search", b2cHandler.SearchTrips)
		api.GET("/journeys/:id/seats", b2cHandler.GetJourneySeats)

		// Guest booking & ticket download (no auth required)
		api.POST("/guest/checkout", guestHandler.GuestCheckout)
		api.POST("/payments/flexpay/callback", paymentHandler.FlexPayCallback)
		api.GET("/tickets/download/:token", b2cHandler.DownloadTicket)
	}

	// Authenticated routes (both owner and passenger)
	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// B2C: authenticated passenger
		auth.GET("/bookings", b2cHandler.GetUserBookings)
		auth.POST("/bookings", b2cHandler.BookTickets)
		auth.PATCH("/bookings/:id/cancel", b2cHandler.CancelBooking)
		auth.DELETE("/bookings/:id", b2cHandler.DeleteBooking)

		// Ticket management
		auth.PUT("/tickets/:id", b2cHandler.UpdateTicket)
		auth.PATCH("/tickets/:id/cancel", b2cHandler.CancelTicket)

		// Email verification (trigger resend)
		auth.POST("/auth/send-verification", b2cHandler.SendVerificationEmail)

		// User settings (both roles)
		auth.GET("/auth/profile", b2cHandler.GetProfile)
		auth.PUT("/auth/profile", b2cHandler.UpdateProfile)
		auth.POST("/auth/change-password", b2cHandler.ChangePassword)
		auth.DELETE("/auth/account", b2cHandler.DeleteAccount)
	}

	// B2B routes (owner only)
	b2b := api.Group("")
	b2b.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	b2b.Use(middleware.RoleMiddleware("owner"))
	{
		// Buses
		b2b.POST("/buses", b2bHandler.CreateBus)
		b2b.GET("/buses", b2bHandler.ListBuses)
		b2b.GET("/buses/:id", b2bHandler.GetBus)
		b2b.PUT("/buses/:id", b2bHandler.UpdateBus)
		b2b.DELETE("/buses/:id", b2bHandler.DeleteBus)
		b2b.POST("/buses/:id/upload-image", b2bHandler.UploadBusImage)
		b2b.GET("/tickets", b2bHandler.ListOrganizationTickets)

		// Routes
		b2b.POST("/routes", b2bHandler.CreateRoute)
		b2b.GET("/routes", b2bHandler.ListRoutes)
		b2b.GET("/routes/:id", b2bHandler.GetRoute)
		b2b.DELETE("/routes/:id", b2bHandler.DeleteRoute)

		// Journeys
		b2b.POST("/journeys", b2bHandler.CreateJourney)
		b2b.GET("/journeys", b2bHandler.ListJourneys)
		b2b.PATCH("/journeys/:id/status", b2bHandler.UpdateJourneyStatus)

		// QR Scan
		b2b.POST("/scan", b2bHandler.ScanTicket)

				// Admin management (owner-only)
		b2b.POST("/admin/users", b2bHandler.CreateAdminUser)
		b2b.GET("/admin/users", b2bHandler.ListAdminUsers)
		b2b.DELETE("/admin/users/:id", b2bHandler.DeleteAdminUser)
		// Analytics
		b2b.GET("/analytics/dashboard", analyticsHandler.GetDashboardStats)
		b2b.GET("/analytics/revenue", analyticsHandler.GetRevenueReport)
	}

	// Admin routes (super-admin only)
	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret), middleware.RoleMiddleware("platform_admin"))
	{
		admin.GET("/pending-owners", b2bHandler.ListPendingOwners)
		admin.POST("/owners/:id/approve", b2bHandler.ApproveOwner)
		admin.POST("/owners/:id/reject", b2bHandler.RejectOwner)
		admin.GET("/pending-agencies", b2bHandler.ListPendingAgencies)
		admin.POST("/agencies/:id/approve", b2bHandler.ApproveAgency)
		admin.POST("/agencies/:id/reject", b2bHandler.RejectAgency)
		admin.GET("/agencies", b2bHandler.ListAgencies)
		admin.PATCH("/agencies/:id/status", b2bHandler.SetAgencyActive)
		admin.POST("/agencies/:id/impersonate", b2bHandler.ImpersonateOwner)
		admin.GET("/exchange-rate", adminHandler.GetExchangeRate)
		admin.PUT("/exchange-rate", adminHandler.UpdateExchangeRate)
	}

	// Platform routes (super-admin only)
	platform := r.Group("/api/platform")
	platform.Use(middleware.AuthMiddleware(cfg.JWTSecret), middleware.RoleMiddleware("platform_admin"))
	{
		platform.GET("/agencies", b2bHandler.ListAgencies)
		platform.GET("/pending-agencies", b2bHandler.ListPendingAgencies)
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
