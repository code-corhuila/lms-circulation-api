// Package config loads configuration from environment variables — the only source of
// environment-specific configuration (library-docs/05-architecture/cross-cutting.md).
package config

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Port string

	// DBHost/DBPort/DBUser/DBPassword/DBName point at MongoDB (loan_db), not
	// PostgreSQL — see ADR-005-mongodb-for-circulation-service.md. Field names
	// kept consistent with every other service's config for the same env vars
	// (DB_HOST, DB_PORT, ...), even though the engine differs.
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// JWTPublicKey validates a real Administrator session token, issued by
	// lms-access-api and signed RS256 — this service never holds the private
	// key (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7).
	JWTPublicKey *rsa.PublicKey
	// InternalJWTSecret is a second, deliberately separate secret: this
	// service both validates it (incoming calls) and mints it (outgoing
	// calls to lms-membership-api and lms-catalog-api).
	InternalJWTSecret string
	JWTExpiry         time.Duration

	LogLevel   string
	CORSOrigin string

	// MembershipServiceURL and CatalogServiceURL are where LoanRegistrationService
	// checks student eligibility/applies suspensions and adjusts book copies —
	// Circulation has no DB access to either table.
	MembershipServiceURL string
	CatalogServiceURL    string
}

func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	publicKeyPEM := getEnv("JWT_PUBLIC_KEY", "")
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY must be set")
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(strings.ReplaceAll(publicKeyPEM, `\n`, "\n")))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_PUBLIC_KEY: %w", err)
	}

	cfg := &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "27017"),
		DBUser:     getEnv("DB_USER", "lms_user"),
		DBPassword: getEnv("DB_PASSWORD", "lms_password"),
		DBName:     getEnv("DB_NAME", "loan_db"),

		JWTPublicKey:      publicKey,
		InternalJWTSecret: getEnv("INTERNAL_JWT_SECRET", ""),
		JWTExpiry:         jwtExpiry,

		LogLevel:   getEnv("LOG_LEVEL", "info"),
		CORSOrigin: getEnv("CORS_ORIGIN", "*"),

		MembershipServiceURL: getEnv("MEMBERSHIP_SERVICE_URL", "http://membership-service:8080"),
		CatalogServiceURL:    getEnv("CATALOG_SERVICE_URL", "http://catalog-service:8080"),
	}

	if cfg.InternalJWTSecret == "" {
		return nil, fmt.Errorf("INTERNAL_JWT_SECRET must be set")
	}

	return cfg, nil
}

// MongoURI builds the MongoDB connection string.
func (c *Config) MongoURI() string {
	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/%s?authSource=admin",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
