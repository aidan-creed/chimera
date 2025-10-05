// cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jjckrbbt/chimera/backend/internal/api"
	"github.com/jjckrbbt/chimera/backend/internal/config"
	"github.com/jjckrbbt/chimera/backend/internal/connections"
	"github.com/jjckrbbt/chimera/backend/internal/ingestion"
	"github.com/jjckrbbt/chimera/backend/internal/logger"
	"github.com/jjckrbbt/chimera/backend/internal/processing"
	"github.com/jjckrbbt/chimera/backend/internal/rag"
	"github.com/jjckrbbt/chimera/backend/internal/repository"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func slogPanicRecoverMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					reqLogger := logger.With("request_id", c.Get("request_id"))
					reqLogger.ErrorContext(c.Request().Context(), "PANIC recovered",
						slog.Any("error", err),
						slog.String("stack", string(debug.Stack())),
					)
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}

func main() {
	// 1. Load application configuration FIRST.
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Sentry and then Sentry's handler
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.AppEnv,
		TracesSampleRate: 1.0,
		Debug:            false,
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}
	defer sentry.Flush(2 * time.Second)

	// 3. Initialize the Logger.
	logger.InitLogger(cfg.AppEnv)
	appLogger := logger.L() // Get the configured logger instance

	appLogger.Info("Application starting up...", "environment", cfg.AppEnv)

	// 4. Connect to the Database.
	dbClient, err := connections.ConnectDB(cfg.DatabaseURL, appLogger.With("component", "database_connector"))
	if err != nil {
		appLogger.Error("Failed to connect to database at startup", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbClient.Close()
	appLogger.Info("Database connection established.")

	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		appLogger.Error("Failed to create GCS client on startup", slog.Any("error", err))
		os.Exit(1)
	}
	appLogger.Info("GCS client initialized.")

	// 5. Initialize Core Application Components.
	platformQuerier := repository.New(dbClient.Pool)

	apiLogger := appLogger.With("service", "api_handlers")

	ingestionService, err := ingestion.NewService(platformQuerier, gcsClient, cfg, apiLogger)
	if err != nil {
		appLogger.Error("Failed to initialize ingestion service", slog.Any("error", err))
		os.Exit(1)
	}
	appLogger.Info("Ingestion service initialized.")

	configLoader, err := processing.NewConfigLoader("./backend/configs")
	if err != nil {
		appLogger.Error("Failed to load configs", slog.Any("error", err))
		os.Exit(1)
	}
	appLogger.Info("catalyst Config Loader initialized.")

	processorLogger := appLogger.With("service", "catalyst_data_processor")
	processingService := processing.NewService(ingestionService, configLoader, platformQuerier, gcsClient, processorLogger, cfg, dbClient.Pool)
	ragService := rag.NewRAGService(cfg.EMBEDDING_SERVICE_URL, cfg.AIAPIKey, cfg.LLMURL, apiLogger)
	appLogger.Info("Processing service initialized.")

	fetcherRegistry := api.NewFetcherRegistry()

	// Initialize your HTTP API handlers.

	itemHandler := api.NewItemHandler(platformQuerier, dbClient.Pool, apiLogger, fetcherRegistry)
	uploadHandler := api.NewUploadHandler(ingestionService, processingService, ragService, configLoader, apiLogger)
	triageHandler := api.NewTriageHandler(dbClient.Pool, platformQuerier, apiLogger)

	appLogger.Info("API handlers initialized.")

	// 6. Initialize Echo.
	e := echo.New()

	// Configure Echo's logger to use our slog instance.
	e.Logger.SetOutput(io.Discard)
	e.Logger.SetLevel(0)   // Set to 0 to disable logging, we use slog
	e.Logger.SetHeader("") // Remove default header, slog adds better ones

	// 7. Register Middleware.
	// Recover middleware: Recovers from panics anywhere in the chain and handles the error.
	e.Use(slogPanicRecoverMiddleware(appLogger))
	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"}, // Replace with your React dev server URL
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Accept", "Authorization"},
		// Add AllowCredentials: true if you send cookies/credentials
	}))

	// --- Auth Middleware Setup ---
	apiGroup := e.Group("/api")

	if cfg.AppEnv == "development" {
		appLogger.Warn("!!!!!!!!!! AUTHENTICATION MIDDLEWARE IS DISABLED IN DEVELOPMENT MODE !!!!!!!!!!")
		// This is our mock middleware for local development.
		apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// Hardcode a user ID for development. User ID 1 is usually the first admin.
				const devUserID int64 = 1
				// Create a new context with the hardcoded user ID.
				ctxWithUser := context.WithValue(c.Request().Context(), "userID", devUserID)
				// Set the new context on the request.
				c.SetRequest(c.Request().WithContext(ctxWithUser))

				appLogger.Debug("Bypassing auth and setting dev user", "user_id", devUserID)
				return next(c)
			}
		})
	} else {
		// In staging or production, use the real Identity Provider middleware.
		appLogger.Info("Initializing Identity Provider middleware for production environment.")
		//		authMiddleware := api.NewAuthMiddleware(cfg.IDENTITY_PROVIDER_DOMAIN, cfg.IDENTITY_PROVIDER_AUDIENCE, platformQuerier, appLogger)
		if err != nil {
			appLogger.Error("Failed to initialize auth middleware", slog.Any("error", err))
			os.Exit(1)
		}
		//		apiGroup.Use(authMiddleware.ValidateRequest)
	}
	// --- End Auth Middleware Setup ---
	// Request Logger Middleware (For consistent request logging)
	// This logs basic request info using our slog instance.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := uuid.New().String() // Generate/extract request ID
			c.Set("requestID", reqID)    // Store request ID in context for later access

			start := time.Now()

			if hub := sentryecho.GetHubFromContext(c); hub != nil {
				hub.Scope().SetTag("request_id", c.Get("requestID").(string))
			}

			err := next(c)
			stop := time.Now()

			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				}
			}

			// Log the request summary with context
			appLogger.InfoContext(c.Request().Context(), "HTTP Request",
				"request_id", reqID,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"latency_ms", stop.Sub(start).Milliseconds(),
				"user_agent", c.Request().UserAgent(),
				"ip", c.RealIP(),
			)
			return err
		}
	})

	e.Use(sentryecho.New(sentryecho.Options{
		Repanic: true,
	}))

	// 8. Register Routes.

	// Health check endpoint (simple GET)
	e.GET("/health", func(c echo.Context) error {
		// Log using a logger with request context
		reqLogger := appLogger.With("request_id", c.Get("requestID")) // Retrieve request ID from context
		reqLogger.InfoContext(c.Request().Context(), "Health check requested", "ip", c.RealIP())

		if err := dbClient.Ping(); err != nil {
			reqLogger.ErrorContext(c.Request().Context(), "Database ping failed during health check", slog.Any("error", err))

			sentry.CaptureException(err)

			return c.String(http.StatusInternalServerError, "DB Not Ready") // Return string response for error
		}
		return c.String(http.StatusOK, "OK") // Return string response for success
	})

	//Upload group
	apiGroup.POST("/upload/:reportType", uploadHandler.HandleUpload)

	// Triage group
	triageHandler.RegisterRoutes(apiGroup)

	//Items group
	itemRoutes := apiGroup.Group("/items")
	itemRoutes.GET("", itemHandler.HandleGetItems)
	itemRoutes.GET("/:id", itemHandler.HandleGetItems)
	itemRoutes.GET("/history/:id", itemHandler.HandleGetHistory)
	itemRoutes.POST("", itemHandler.HandleCreateItem)
	itemRoutes.PATCH("/:id", itemHandler.HandleUpdateItem)

	//Dashbord group
	//	apiGroup.GET("/dashboard", dashboardHandler.HandleGetDashboardStats)

	e.GET("/foo", func(ctx echo.Context) error {
		// sentryecho handler will catch it just fine. Also, because we attached "someRandomTag"
		// in the middleware before, it will be sent through as well
		panic("y tho")
	})

	//	for _, route := range e.Routes() {
	//		appLogger.Info("Registered Route", "method", route.Method, "path", route.Path)
	//	}

	// 9. Start the HTTP server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := fmt.Sprintf("0.0.0.0:%s", port)

	appLogger.Info("HTTP Server starting on port", "port", port)

	// e.Start blocks until the server is shut down or an error occurs.
	if err := e.Start(address); err != nil && err != http.ErrServerClosed {
		// Only log fatal if it's not a graceful shutdown error.
		appLogger.Error("HTTP Server failed to start", slog.Any("error", err))
		os.Exit(1)
	}
	// This message would appear after a graceful shutdown.
	appLogger.Info("HTTP Server stopped gracefully.")
}
