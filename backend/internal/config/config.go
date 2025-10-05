package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv" // You'll need to run: go get github.com/joho/godotenv
)

// Config holds all application-wide configuration loaded from environment variables.
type Config struct {
	DatabaseURL                string
	IDENTITY_PROVIDER_DOMAIN   string
	IDENTITY_PROVIDER_AUDIENCE string
	AppEnv                     string
	GCSBucketName              string
	SentryDSN                  string
	AIAPIKey                   string
	LLMURL                     string
	EMBEDDING_SERVICE_URL      string
}

// LoadConfig reads configuration from environment variables or a .env file.
// It is the single source of truth for application configuration.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists. This is great for local development.
	// In production, these will be set directly in the environment.
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("FATAL: DATABASE_URL environment variable not set")
	}

	IDENTITY_PROVIDER_DOMAIN := os.Getenv("IDENTITY_PROVIDER_DOMAIN")
	if IDENTITY_PROVIDER_DOMAIN == "" {
		return nil, fmt.Errorf("FATAL: IDENTITY_PROVIDER_DOMAIN environment variable not set")
	}

	IDENTITY_PROVIDER_AUDIENCE := os.Getenv("IDENTITY_PROVIDER_AUDIENCE")
	if IDENTITY_PROVIDER_AUDIENCE == "" {
		return nil, fmt.Errorf("FATAL: IDENTITY_PROVIDER_AUDIENCE environment variable not set")
	}

	gcsBucketName := os.Getenv("GCS_BUCKET_NAME")
	if gcsBucketName == "" {
		return nil, fmt.Errorf("FATAL: GCS_BUCKET_NAME environment variable not set")
	}

	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN == "" {
		return nil, fmt.Errorf("FATAL: SENTRY_DSN environment variable not set")
	}

	AIKey := os.Getenv("AI_API_KEY")
	if AIKey == "" {
		return nil, fmt.Errorf("FATAL: AI_API_KEY environment variable not set")
	}

	LLM_URL := os.Getenv("LLM_URL")
	if LLM_URL == "" {
		return nil, fmt.Errorf("FATAL: LLM_URL environment variable not set")
	}

	// AppEnv can have a default value
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	EMBEDDING_SERVICE_URL := os.Getenv("EMBEDDING_SERVICE_URL")
	if EMBEDDING_SERVICE_URL == "" {
		return nil, fmt.Errorf("FATAL: EMBEDDING_SERVICE_URL environment variable not set")
	}

	return &Config{
		DatabaseURL:                dbURL,
		IDENTITY_PROVIDER_DOMAIN:   IDENTITY_PROVIDER_DOMAIN,
		IDENTITY_PROVIDER_AUDIENCE: IDENTITY_PROVIDER_AUDIENCE,
		AppEnv:                     appEnv,
		GCSBucketName:              gcsBucketName,
		SentryDSN:                  sentryDSN,
		AIAPIKey:                   AIKey,
		LLMURL:                     LLM_URL,
		EMBEDDING_SERVICE_URL:      EMBEDDING_SERVICE_URL,
	}, nil
}
