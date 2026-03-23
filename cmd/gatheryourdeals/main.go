package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/handler"
	"github.com/gatheryourdeals/data/internal/logger"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gatheryourdeals/data/internal/repository/postgres"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/telemetry"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configPath string

var stdinReader = bufio.NewReader(os.Stdin)

func main() {
	root := &cobra.Command{
		Use:   "gatheryourdeals",
		Short: "GatherYourDeals data service",
	}

	root.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to the config file")

	root.AddCommand(serveCmd())
	root.AddCommand(initCmd())
	root.AddCommand(adminCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Database abstraction
// ---------------------------------------------------------------------------

// repos holds all repository implementations created from a single database.
type repos struct {
	Users        repository.UserRepository
	Meta         repository.MetaFieldRepository
	Receipts     repository.ReceiptRepository
	RefreshStore auth.RefreshTokenStore
	closer       io.Closer
}

// Close closes the underlying database connection.
func (r *repos) Close() error {
	return r.closer.Close()
}

// openDatabase loads the config and opens the configured database,
// returning all repository implementations ready to use.
func openDatabase() (*config.Config, *repos, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	var r *repos
	switch cfg.Database.Driver {
	case "postgres":
		slog.Info("database: using postgres")
		db, err := postgres.New(cfg.Database.EffectiveDSN())
		if err != nil {
			return nil, nil, fmt.Errorf("open database: %w", err)
		}
		metaRepo := postgres.NewMetaFieldRepo(db)
		r = &repos{
			Users:        postgres.NewUserRepo(db),
			Meta:         metaRepo,
			Receipts:     postgres.NewReceiptRepo(db, metaRepo),
			RefreshStore: postgres.NewRefreshTokenStore(db),
			closer:       db,
		}
	default: // "sqlite"
		slog.Info("database: using sqlite", "path", cfg.Database.Path)
		db, err := sqlite.New(cfg.Database.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("open database: %w", err)
		}
		metaRepo := sqlite.NewMetaFieldRepo(db)
		r = &repos{
			Users:        sqlite.NewUserRepo(db),
			Meta:         metaRepo,
			Receipts:     sqlite.NewReceiptRepo(db, metaRepo),
			RefreshStore: sqlite.NewRefreshTokenStore(db),
			closer:       db,
		}
	}
	return cfg, r, nil
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

// serveCmd starts the HTTP server.
func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Signal-aware context: cancelled on SIGTERM or SIGINT (FR-017).
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Telemetry: noop when OTEL_EXPORTER_OTLP_ENDPOINT is unset (FR-003).
			otelProv, err := telemetry.Setup(ctx)
			if err != nil {
				return fmt.Errorf("init telemetry: %w", err)
			}

			cfg, r, err := openDatabase()
			if err != nil {
				return err
			}
			defer func() { _ = r.Close() }()

			// Logging: OTel log bridge added as second sink when configured (FR-005).
			appLogger, err := logger.New(logger.Config{
				Dir:      cfg.Log.Dir,
				Prefix:   "gatheryourdeals",
				MaxBytes: int64(cfg.Log.MaxSizeMB) * 1024 * 1024,
				MaxFiles: 2,
			}, otelProv.LoggerProvider())
			if err != nil {
				return fmt.Errorf("init logger: %w", err)
			}
			defer func() { _ = appLogger.Close() }()
			slog.SetDefault(appLogger.Logger)

			// Auth
			secret, err := cfg.JWTSecret()
			if err != nil {
				return err
			}
			authService := auth.NewService(r.Users)

			accessExp, err := cfg.Auth.GetAccessTokenDuration()
			if err != nil {
				return fmt.Errorf("parse access_token_exp: %w", err)
			}
			refreshExp, err := cfg.Auth.GetRefreshTokenDuration()
			if err != nil {
				return fmt.Errorf("parse refresh_token_exp: %w", err)
			}
			tokenService := auth.NewTokenService(secret, accessExp, refreshExp, r.RefreshStore)

			// Guard: require admin to exist before serving traffic
			hasAdmin, err := authService.HasAdmin(ctx)
			if err != nil {
				return fmt.Errorf("check admin: %w", err)
			}
			if !hasAdmin {
				return fmt.Errorf("no admin account found — run 'gatheryourdeals init' first")
			}

			// Handlers + router
			authHandler := handler.NewAuthHandler(authService, tokenService)
			userHandler := handler.NewUserHandler(r.Users)
			metaHandler := handler.NewMetaHandler(r.Meta)
			receiptHandler := handler.NewReceiptHandler(r.Receipts)
			router := handler.NewRouter(
				authHandler, userHandler, metaHandler, receiptHandler,
				tokenService, appLogger.Writer(), otelProv.TracerProvider(),
			)

			addr := fmt.Sprintf(":%s", cfg.Server.Port)
			slog.Info("server starting", "addr", addr)

			// Run server in a goroutine so we can wait for the signal below.
			srv := &http.Server{Addr: addr, Handler: router}
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("server error", "error", err)
				}
			}()

			// Block until SIGTERM or SIGINT.
			<-ctx.Done()
			stop() // release signal capture to allow a second signal to force-kill

			// Graceful shutdown: flush OTel first, then drain HTTP (FR-017).
			// Each operation gets its own independent deadline so OTel flush
			// time does not eat into the HTTP server's drain budget.
			otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer otelCancel()
			if err := otelProv.Shutdown(otelCtx); err != nil {
				slog.Warn("otel shutdown error", "error", err)
			}

			srvCtx, srvCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer srvCancel()
			if err := srv.Shutdown(srvCtx); err != nil {
				slog.Warn("server shutdown error", "error", err)
			}
			return nil
		},
	}
}

// initCmd creates the database and prompts for admin credentials.
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database and create the admin account",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, r, err := openDatabase()
			if err != nil {
				return err
			}
			defer func() { _ = r.Close() }()

			svc := auth.NewService(r.Users)

			ctx := context.Background()
			exists, err := svc.HasAdmin(ctx)
			if err != nil {
				return err
			}
			if exists {
				fmt.Println("Admin account already exists. No changes made.")
				return nil
			}

			username, password, err := promptCredentials("Admin username: ", "Admin password: ")
			if err != nil {
				return err
			}

			user, err := svc.CreateAdmin(ctx, username, password)
			if err != nil {
				return fmt.Errorf("create admin: %w", err)
			}

			fmt.Printf("Admin account created.\n  ID:       %s\n  Username: %s\n", user.ID, user.Username)
			return nil
		},
	}
}

// adminCmd groups admin management subcommands.
func adminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative operations",
	}
	cmd.AddCommand(resetPasswordCmd())
	return cmd
}

func resetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password",
		Short: "Reset a user's password",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, r, err := openDatabase()
			if err != nil {
				return err
			}
			defer func() { _ = r.Close() }()

			svc := auth.NewService(r.Users)

			username, err := promptInput("Username: ")
			if err != nil {
				return err
			}
			password, err := promptPasswordWithConfirm("New password: ")
			if err != nil {
				return err
			}

			ctx := context.Background()
			if err := svc.ResetPassword(ctx, username, password); err != nil {
				return fmt.Errorf("reset password: %w", err)
			}

			fmt.Printf("Password for '%s' has been reset.\n", username)
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Input helpers
// ---------------------------------------------------------------------------

// promptCredentials asks for a username and a confirmed password.
func promptCredentials(usernameLabel, passwordLabel string) (string, string, error) {
	username, err := promptInput(usernameLabel)
	if err != nil {
		return "", "", err
	}
	password, err := promptPasswordWithConfirm(passwordLabel)
	if err != nil {
		return "", "", err
	}
	return username, password, nil
}

// promptPasswordWithConfirm asks for a password twice and validates it.
func promptPasswordWithConfirm(label string) (string, error) {
	password, err := promptPassword(label)
	if err != nil {
		return "", err
	}
	confirm, err := promptPassword("Confirm password: ")
	if err != nil {
		return "", err
	}
	if password != confirm {
		return "", fmt.Errorf("passwords do not match")
	}
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	return password, nil
}

func promptInput(label string) (string, error) {
	fmt.Print(label)
	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func promptPassword(label string) (string, error) {
	fmt.Print(label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return promptInput(label)
	}
	return strings.TrimSpace(string(b)), nil
}
