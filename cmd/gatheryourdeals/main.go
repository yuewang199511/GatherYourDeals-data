package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gatheryourdeals/data/internal/auth"
	"github.com/gatheryourdeals/data/internal/config"
	"github.com/gatheryourdeals/data/internal/handler"
	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	dbPath     string
	configPath string
)

func main() {
	root := &cobra.Command{
		Use:   "gatheryourdeals",
		Short: "GatherYourDeals data service",
	}

	defaultConfig := os.Getenv("GYD_CONFIG")
	if defaultConfig == "" {
		defaultConfig = "config.yaml"
	}
	root.PersistentFlags().StringVar(&configPath, "config", defaultConfig, "path to the config file")
	root.PersistentFlags().StringVar(&dbPath, "db", "", "path to the SQLite database file (overrides config)")

	root.AddCommand(serveCmd())
	root.AddCommand(initCmd())
	root.AddCommand(adminCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// serveCmd starts the HTTP server.
func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			dbFile := resolveDBPath()

			db, err := sqlite.New(dbFile)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			userRepo := sqlite.NewUserRepo(db)
			clientRepo := sqlite.NewClientRepo(db)
			authService := auth.NewService(userRepo)

			// Seed clients from config if the database has none.
			ctx := context.Background()
			if err := seedClients(ctx, clientRepo, cfg); err != nil {
				return fmt.Errorf("seed clients: %w", err)
			}

			// Setup OAuth2 with database-backed client store
			oauthManager, err := auth.NewOAuthManager(cfg, clientRepo)
			if err != nil {
				return fmt.Errorf("setup oauth2: %w", err)
			}
			oauthServer := auth.NewOAuthServer(oauthManager, authService)

			hasAdmin, err := authService.HasAdmin(ctx)
			if err != nil {
				return fmt.Errorf("check admin: %w", err)
			}
			if !hasAdmin {
				return fmt.Errorf("no admin account found. Run 'gatheryourdeals init' to create one before starting the server")
			}

			authHandler := handler.NewAuthHandler(authService, oauthServer, clientRepo)
			adminHandler := handler.NewAdminHandler(clientRepo)
			r := handler.NewRouter(authHandler, adminHandler, oauthManager, userRepo)

			addr := fmt.Sprintf(":%s", cfg.Server.Port)
			log.Printf("server starting on %s", addr)
			return r.Run(addr)
		},
	}
	return cmd
}

// seedClients inserts clients from config.yaml into the database
// if no clients exist yet. This handles first-time setup.
func seedClients(ctx context.Context, clients *sqlite.ClientRepo, cfg *config.Config) error {
	hasClients, err := clients.HasClients(ctx)
	if err != nil {
		return err
	}
	if hasClients {
		return nil
	}

	for _, c := range cfg.OAuth2.Clients {
		client := &model.OAuthClient{
			ID:     c.ID,
			Secret: c.Secret,
			Domain: c.Domain,
		}
		if err := clients.CreateClient(ctx, client); err != nil {
			return fmt.Errorf("seed client %q: %w", c.ID, err)
		}
		log.Printf("seeded OAuth2 client: %s", c.ID)
	}
	return nil
}

// initCmd creates the database and prompts for admin credentials.
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database and create the admin account",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFile := resolveDBPath()

			db, err := sqlite.New(dbFile)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			userRepo := sqlite.NewUserRepo(db)
			svc := auth.NewService(userRepo)

			ctx := context.Background()
			exists, err := svc.HasAdmin(ctx)
			if err != nil {
				return err
			}
			if exists {
				fmt.Println("Admin account already exists. No changes made.")
				return nil
			}

			username, err := promptInput("Admin username: ")
			if err != nil {
				return err
			}

			password, err := promptPassword("Admin password: ")
			if err != nil {
				return err
			}
			confirm, err := promptPassword("Confirm password: ")
			if err != nil {
				return err
			}
			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}
			if len(password) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}

			user, err := svc.CreateAdmin(ctx, username, password)
			if err != nil {
				return fmt.Errorf("create admin: %w", err)
			}

			fmt.Printf("Admin account created successfully.\n  ID:       %s\n  Username: %s\n", user.ID, user.Username)
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

// resetPasswordCmd resets a user's password interactively.
func resetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password",
		Short: "Reset a user's password",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFile := resolveDBPath()

			db, err := sqlite.New(dbFile)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			userRepo := sqlite.NewUserRepo(db)
			svc := auth.NewService(userRepo)

			username, err := promptInput("Username: ")
			if err != nil {
				return err
			}

			password, err := promptPassword("New password: ")
			if err != nil {
				return err
			}
			confirm, err := promptPassword("Confirm password: ")
			if err != nil {
				return err
			}
			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}
			if len(password) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
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

func promptInput(label string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// resolveDBPath returns the database path from the CLI flag if set,
// otherwise from the GYD_DB env var, otherwise from the config file,
// otherwise the default.
func resolveDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	if envDB := os.Getenv("GYD_DB"); envDB != "" {
		return envDB
	}
	cfg, err := config.Load(configPath)
	if err == nil && cfg.Database.Path != "" {
		return cfg.Database.Path
	}
	return "gatheryourdeals.db"
}

func promptPassword(label string) (string, error) {
	fmt.Print(label)
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return promptInput(label)
	}
	return strings.TrimSpace(string(bytes)), nil
}
