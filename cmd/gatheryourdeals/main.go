package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/handler"
	"github.com/gatheryourdeals/data/internal/logger"
	"github.com/gatheryourdeals/data/internal/repository"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configPath string

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
	RefreshStore *sqlite.RefreshTokenStore // TODO: extract interface when adding postgres
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

	// Currently only SQLite is supported. When adding PostgreSQL,
	// switch on a cfg.Database.Driver field here.
	db, err := sqlite.New(cfg.Database.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}

	metaRepo := sqlite.NewMetaFieldRepo(db)

	r := &repos{
		Users:        sqlite.NewUserRepo(db),
		Meta:         metaRepo,
		Receipts:     sqlite.NewReceiptRepo(db, metaRepo),
		RefreshStore: sqlite.NewRefreshTokenStore(db),
		closer:       db,
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
			cfg, r, err := openDatabase()
			if err != nil {
				return err
			}
			defer func() { _ = r.Close() }()

			// Logging
			appLogger, err := logger.New(logger.Config{
				Dir:      cfg.Log.Dir,
				Prefix:   "gatheryourdeals",
				MaxBytes: int64(cfg.Log.MaxSizeMB) * 1024 * 1024,
				MaxFiles: 2,
			})
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
			ctx := context.Background()
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
			router := handler.NewRouter(authHandler, userHandler, metaHandler, receiptHandler, tokenService, appLogger.Writer())

			addr := fmt.Sprintf(":%s", cfg.Server.Port)
			slog.Info("server starting", "addr", addr)
			return router.Run(addr)
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
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	input, err := reader.ReadString('\n')
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
