// Command api is the entry point for circulation-service — see
// library-docs/09-microservices/services/05-circulation-service/README.md.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi"
	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/handler"
	catalogclient "github.com/code-corhuila/lms-circulation-api/internal/adapter/out/catalogclient"
	membershipclient "github.com/code-corhuila/lms-circulation-api/internal/adapter/out/membershipclient"
	"github.com/code-corhuila/lms-circulation-api/internal/adapter/out/persistence"
	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/config"
	"github.com/code-corhuila/lms-circulation-api/internal/infrastructure/logger"

	"github.com/code-corhuila/lms-circulation-api/internal/application/service"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		println("config error:", err.Error())
		return err
	}

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = zapLog.Sync() }()
	log := zapLog.Sugar()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoClient, err := persistence.NewClient(ctx, cfg.MongoURI())
	if err != nil {
		log.Errorw("failed to connect to database", "error", err)
		return err
	}
	defer func() { _ = mongoClient.Disconnect(ctx) }()

	// Indexes are no longer created here — lms-circulation-db's own Liquibase
	// migrations own them now (rules/2-anexos/B-db-mongo.md; the -api creating
	// schema was flagged critical on lms-circulation-api#2's review). This
	// service just assumes they already exist by the time it starts.
	db := mongoClient.Database(cfg.DBName)

	loanRepo := persistence.NewLoanRepository(db)
	studentClient := membershipclient.NewClient(cfg.MembershipServiceURL, cfg.InternalJWTSecret)
	bookClient := catalogclient.NewClient(cfg.CatalogServiceURL, cfg.InternalJWTSecret)
	idempotencyStore := persistence.NewIdempotencyStore(db)

	loanRegistrationService := service.NewLoanRegistrationService(studentClient, bookClient, loanRepo)
	registerLoanUseCase := usecase.NewRegisterLoan(loanRegistrationService, loanRepo, idempotencyStore)
	returnLoanUseCase := usecase.NewReturnLoan(loanRegistrationService)
	searchLoansUseCase := usecase.NewSearchLoans(loanRepo)
	overdueLoansUseCase := usecase.NewOverdueLoans(loanRepo)
	loanHandler := handler.NewLoanHandler(registerLoanUseCase, returnLoanUseCase, searchLoansUseCase, overdueLoansUseCase)

	router := httpserver.NewRouter(httpserver.RouterConfig{
		DB:                mongoClient,
		JWTPublicKey:      cfg.JWTPublicKey,
		InternalJWTSecret: cfg.InternalJWTSecret,
		CORSOrigin:        cfg.CORSOrigin,
		Loans:             loanHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infow("circulation-service listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorw("server error", "error", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
