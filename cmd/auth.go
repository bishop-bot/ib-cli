package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/bishop-bot/ibcli/internal/api"
	"github.com/bishop-bot/ibcli/internal/auth"
	"github.com/bishop-bot/ibcli/internal/config"
	"github.com/bishop-bot/ibcli/internal/models"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with the IB Client Portal Gateway.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the gateway",
	Long: `Authenticate with the IB Client Portal Gateway.

Examples:
  # Interactive login (prompts for credentials and master password)
  ib-cli auth login

  # Login with credentials (will prompt for master password)
  ib-cli auth login --username myuser --password mypass

  # Login with existing session
  ib-cli auth login --restore
`,
	RunE: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "End the session",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE:  runStatus,
}

var (
	loginParams struct {
		Username   string
		Password   string
		MasterPass string
		Save       bool
		Restore    bool
	}
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	loginCmd.Flags().StringVarP(&loginParams.Username, "username", "u", "", "IB account username")
	loginCmd.Flags().StringVarP(&loginParams.Password, "password", "p", "", "IB account password")
	loginCmd.Flags().StringVar(&loginParams.MasterPass, "master-password", "", "Master password for session encryption")
	loginCmd.Flags().BoolVarP(&loginParams.Save, "save", "s", false, "Save session for later use")
	loginCmd.Flags().BoolVarP(&loginParams.Restore, "restore", "r", false, "Restore existing session")
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	// Handle restore mode
	if loginParams.Restore {
		return restoreSession(client, cfg)
	}

	// Get credentials from flags or config
	username := loginParams.Username
	password := loginParams.Password

	if username == "" {
		username = cfg.Auth.Username
	}
	if password == "" {
		password = cfg.Auth.Password
	}

	// Interactive credential prompt
	if username == "" {
		fmt.Print("Username: ")
		fmt.Scanln(&username)
	}
	if password == "" {
		fmt.Print("Password: ")
		pw, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password = string(pw)
		fmt.Println()
	}

	if username == "" || password == "" {
		return fmt.Errorf("credentials required: use --username and --password, or set in config")
	}

	info("Authenticating with gateway...")

	resp, err := client.Login(ctx, username, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if !resp.IsAuthenticated() {
		return fmt.Errorf("authentication failed: %s", resp.Message)
	}

	info("Authentication successful ✓")

	// Save session if requested
	if loginParams.Save || cfg.Auth.Username != "" {
		masterPass := loginParams.MasterPass
		if masterPass == "" {
			fmt.Print("Master password for session encryption (min 8 chars): ")
			pw, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("reading master password: %w", err)
			}
			masterPass = string(pw)
			fmt.Println()

			if len(masterPass) < 8 {
				return fmt.Errorf("master password must be at least 8 characters")
			}
		}

		if err := saveSession(client, masterPass, username); err != nil {
			return fmt.Errorf("saving session: %w", err)
		}
		info("Session saved to %s", getSessionPath())
	}

	if verbose {
		data, _ := json.MarshalIndent(resp, "", "  ")
		info("Full response:\n%s", string(data))
	}

	return nil
}

func restoreSession(client *api.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get master password
	masterPass := loginParams.MasterPass
	if masterPass == "" {
		fmt.Print("Master password for session decryption: ")
		pw, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("reading master password: %w", err)
		}
		masterPass = string(pw)
		fmt.Println()
	}

	store, err := auth.NewStore(masterPass)
	if err != nil {
		return fmt.Errorf("creating session store: %w", err)
	}

	session, err := store.Load()
	if err != nil {
		if err == auth.ErrNoSession {
			return fmt.Errorf("no saved session found. Run 'ib-cli auth login --save' first")
		}
		if err == auth.ErrSessionExpired {
			return fmt.Errorf("session has expired. Please login again")
		}
		return fmt.Errorf("loading session: %w", err)
	}

	// Set tokens on client
	client.SetSession(session.SessionID, session.Token)

	// Verify session is still valid
	authStatus, err := client.AuthStatus(ctx)
	if err != nil {
		return fmt.Errorf("verifying session: %w", err)
	}

	if !authStatus.IsAuthenticated() {
		// Try to delete expired session
		store.Delete()
		return fmt.Errorf("session expired or invalid. Please login again")
	}

	info("Session restored successfully ✓")
	info("Authenticated as: %s", session.Username)

	return nil
}

func saveSession(client *api.Client, masterPass, username string) error {
	store, err := auth.NewStore(masterPass)
	if err != nil {
		return fmt.Errorf("creating session store: %w", err)
	}

	session := &models.SavedSession{
		SessionID: client.SessionID(),
		Token:     client.Token(),
		Username:  username,
		SavedAt:   time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Default 24h expiration
	}

	if err := store.Save(session); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	info("Logging out...")

	// Try to logout from gateway
	_ = client.Logout(ctx)

	// Delete saved session
	masterPass := loginParams.MasterPass
	if masterPass == "" && len(os.Args) > 2 && os.Args[2] == "logout" {
		// For logout, try without master password to delete the file
		// In a real implementation, you'd want to prompt or use a keyring
		store, _ := auth.NewStore("dummy")
		_ = store.Delete()
	}

	info("Logged out successfully")

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if verbose {
		info("Config: %s", cfg.Gateway.BaseURL())
	}

	client := api.NewClient(cfg)

	// Check saved session
	store, _ := auth.NewStore("check")
	hasSession := store.Exists()

	if verbose {
		info("Checking auth status at %s...", cfg.Gateway.BaseURL())
	}

	// Check auth status
	resp, err := client.AuthStatus(ctx)
	if err != nil {
		if verbose {
			info("Connection error: %v", err)
			info("Gateway URL: %s", cfg.Gateway.BaseURL())
		}
		// Gateway unreachable or not authenticated
		if hasSession {
			info("Status: Session saved (gateway unreachable)")
			info("Hint: Gateway may be offline. URL: %s", cfg.Gateway.BaseURL())
			return nil
		}
		info("Status: Not authenticated")
		info("Gateway: %s (unreachable)", cfg.Gateway.BaseURL())
		return nil
	}

	// Output
	if verbose {
		info("Authenticated: %v, Connected: %v, Failed: %v",
			resp.IsAuthenticated(), resp.IsConnected(), resp.IsFailed())
		info("Gateway: %s", cfg.Gateway.BaseURL())
		info("Saved session exists: %v", hasSession)
	} else {
		if resp.IsAuthenticated() {
			info("Status: Authenticated ✓")
		} else if resp.IsConnected() {
			info("Status: Connected (not authenticated)")
		} else if hasSession {
			info("Status: Session saved (not connected)")
		} else {
			info("Status: Not authenticated")
		}
	}

	return nil
}

func getSessionPath() string {
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.ib-cli/sessions/session.enc", home)
}