package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/services"
)

type Server struct {
	config              *config.Config
	router              *gin.Engine
	httpServer          *http.Server
	formService         *services.FormService
	actionService       *services.ActionService
	notificationService *services.NotificationService
	analyticsService    *services.AnalyticsService
}

func NewServer(cfg *config.Config) *Server {
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	
	// Initialize services
	formService := services.NewFormService(cfg)
	actionService := services.NewActionService(cfg)
	notificationService := services.NewNotificationService(cfg)
	analyticsService := services.NewAnalyticsService(cfg, formService.GetDB()) // We need to expose the DB

	server := &Server{
		config:              cfg,
		router:              router,
		formService:         formService,
		actionService:       actionService,
		notificationService: notificationService,
		analyticsService:    analyticsService,
	}

	server.setupMiddleware()
	server.setupRoutes()

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}

func (s *Server) setupMiddleware() {
	// CORS
	corsConfig := cors.Config{
		AllowOrigins:     s.config.Server.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	s.router.Use(cors.New(corsConfig))

	// Rate limiting
	if s.config.Server.RateLimiting.Enabled {
		s.router.Use(rateLimitMiddleware(s.config.Server.RateLimiting.RequestsPerMinute))
	}

	// Request logging
	s.router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// Recovery middleware
	s.router.Use(gin.Recovery())
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)
	
	// API routes
	api := s.router.Group("/api")
	{
		// Form submission
		api.POST("/forms/submit", s.submitForm)
		
		// Form management
		forms := api.Group("/forms")
		{
			forms.GET("/", s.listForms)
			forms.GET("/:id", s.getForm)
			forms.GET("/:id/submissions", s.getFormSubmissions)
		}
		
		// Analytics routes
		analytics := api.Group("/analytics")
		{
			analytics.POST("/submit", s.submitAnalytics)
			analytics.GET("/summary", s.getAnalyticsSummary)
		}
		
		// Auth routes
		api.POST("/admin/login", s.adminLogin)
		
		// Webhook endpoints (no auth required for external integrations)
		api.POST("/admin/send-welcome-email", s.sendWelcomeEmail)
		
		// Admin routes (require authentication)
		admin := api.Group("/admin")
		admin.Use(s.requireAdminAuth())
		{
			admin.GET("/submissions", s.getAllSubmissions)
			admin.GET("/submissions/:id", s.getSubmission)
			admin.DELETE("/submissions/:id", s.deleteSubmission)
			admin.POST("/submissions/:id/reprocess", s.reprocessSubmission)
			admin.POST("/analytics/cleanup", s.cleanupAnalytics)
			
			// Feedback specific routes
			admin.GET("/feedback", s.getFeedbackSubmissions)
		}
	}

	// Static files (for form preview/testing)
	s.router.Static("/static", "./static")
	s.router.LoadHTMLGlob("templates/*")
	s.router.GET("/", s.indexPage)
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func rateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiter
	// In production, you might want to use Redis or similar
	clients := make(map[string][]time.Time)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		
		// Clean old entries
		if timestamps, exists := clients[clientIP]; exists {
			var validTimestamps []time.Time
			for _, timestamp := range timestamps {
				if now.Sub(timestamp) < time.Minute {
					validTimestamps = append(validTimestamps, timestamp)
				}
			}
			clients[clientIP] = validTimestamps
		}
		
		// Check rate limit
		if len(clients[clientIP]) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}
		
		// Add current request
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}