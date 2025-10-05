// internal/logger/logger.go
package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
	"time"
)

var globalLogger *slog.Logger // The globally accessible logger

func InitLogger(env string) {
	var handler slog.Handler
	var opts slog.HandlerOptions

	// Customize common handler options
	opts.AddSource = true // Always include file:line in logs for easy debugging
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {

		// Rename 'message' to 'msg' (common in structured logging)
		if a.Key == slog.MessageKey {
			a.Key = "msg"
		}
		// Format time to RFC3339Nano for precision and consistency
		if a.Key == slog.TimeKey {
			a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339Nano))
		}
		return a
	}

	switch env {
	case "development":
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, &opts)
	case "development-json":
		opts.Level = slog.LevelDebug
		handler = slog.NewJSONHandler(os.Stdout, &opts)
	case "production", "staging":
		opts.Level = slog.LevelInfo // Only info, warn, error, fatal in production
		opts.AddSource = false      // Optionally remove source in production for performance/log size
		handler = slog.NewJSONHandler(os.Stdout, &opts)
		// In a production environment, it's common to direct output to stderr
		// as many container orchestrators/logging agents collect stderr separately.
		// handler = slog.NewJSONHandler(os.Stderr, &opts)
	default:
		// Fallback for unknown environments, defaulting to production-like logging
		log.Printf("WARNING: Unknown APP_ENV '%s'. Defaulting to production logging.\n", env)
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, &opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger) // Set as the default logger for the whole application
}

// L returns the global slog logger instance.
// This function provides access to the configured logger.
// It includes a safety warning if called before InitLogger, though InitLogger should be
// called once at the very start of main().
func L() *slog.Logger {
	if globalLogger == nil {
		// This block should ideally not be hit if InitLogger is called first in main.
		// It's a fallback for safety/debugging during early development.
		InitLogger("development")
		log.Println("WARNING: Logger accessed before explicit initialization. Using default development logger.")
	}
	return globalLogger
}

// SetOutput sets the logger's output writer. Primarily useful for testing or redirecting output.
func SetOutput(w io.Writer) {
	currentOpts := slog.HandlerOptions{}
	if globalLogger != nil {
		if h, ok := globalLogger.Handler().(interface{ Options() slog.HandlerOptions }); ok {
			currentOpts = h.Options()
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(w, &currentOpts)))
}
