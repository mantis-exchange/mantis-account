package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mantis-exchange/mantis-account/internal/config"
	"github.com/mantis-exchange/mantis-account/internal/handler"
	"github.com/mantis-exchange/mantis-account/internal/middleware"
	"github.com/mantis-exchange/mantis-account/internal/model"
	"github.com/mantis-exchange/mantis-account/internal/service"
)

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	userRepo := model.NewUserRepo(pool)
	balanceRepo := model.NewBalanceRepo(pool)

	expiry, _ := time.ParseDuration(cfg.JWTExpiry)
	if expiry == 0 {
		expiry = 24 * time.Hour
	}

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, expiry)
	balanceService := service.NewBalanceService(balanceRepo)

	h := handler.New(authService, balanceService)

	r := gin.Default()

	// Health check with DB ping.
	r.GET("/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	r.POST("/api/v1/account/register", h.Register)
	r.POST("/api/v1/account/login", h.Login)

	// Authenticated routes
	auth := r.Group("/api/v1")
	auth.Use(middleware.JWTAuth(authService))
	{
		auth.GET("/account", h.GetProfile)
		auth.GET("/account/balances", h.GetBalances)
	}

	// Internal routes (no auth, service-to-service)
	internal := r.Group("/internal/v1/balance")
	{
		internal.POST("/freeze", h.FreezeBalance)
		internal.POST("/unfreeze", h.UnfreezeBalance)
		internal.POST("/credit", h.CreditBalance)
		internal.POST("/deduct-frozen", h.DeductFrozenBalance)
	}

	// Admin routes (internal, no auth)
	admin := r.Group("/internal/v1")
	{
		admin.GET("/users", h.ListUsers)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("mantis-account starting on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
