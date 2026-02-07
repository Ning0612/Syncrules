package gdrive

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	// DefaultTokenFile is the default path for storing OAuth tokens
	DefaultTokenFile = "gdrive-token.json"
	// DeviceAuthTimeout is the timeout for device authentication
	DeviceAuthTimeout = 5 * time.Minute
)

// AuthConfig holds OAuth2 configuration
type AuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Token represents a stored OAuth2 token
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// toOAuth2Token converts to golang.org/x/oauth2.Token
func (t *Token) toOAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}

// fromOAuth2Token creates Token from oauth2.Token
func fromOAuth2Token(t *oauth2.Token) *Token {
	return &Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}

// Authenticator handles OAuth2 authentication for Google Drive
type Authenticator struct {
	config    *oauth2.Config
	tokenPath string
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(clientID, clientSecret, tokenPath string) *Authenticator {
	if tokenPath == "" {
		configDir, err := os.UserConfigDir()
		if err == nil {
			tokenPath = filepath.Join(configDir, "syncrules", DefaultTokenFile)
		} else {
			tokenPath = DefaultTokenFile
		}
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			drive.DriveFileScope, // Full access to user's files
		},
		Endpoint: google.Endpoint,
	}

	return &Authenticator{
		config:    config,
		tokenPath: tokenPath,
	}
}

// GetClient returns an authenticated HTTP client
func (a *Authenticator) GetClient(ctx context.Context) (*oauth2.Token, error) {
	// Try to load existing token
	token, err := a.loadToken()
	if err != nil {
		return nil, fmt.Errorf("no token found, please run 'syncrules auth gdrive' first")
	}

	// If token is still valid, return it
	if token.Valid() {
		return token, nil
	}

	// Token expired but has refresh token - try to refresh
	if token.RefreshToken != "" {
		refreshedToken, err := a.RefreshToken(ctx, token)
		if err == nil {
			return refreshedToken, nil
		}
		// Refresh failed, fall through to re-auth message
	}

	return nil, fmt.Errorf("token expired and refresh failed, please run 'syncrules auth gdrive' to re-authenticate")
}

// generateRandomState generates a cryptographically secure random state string
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Authenticate performs device flow authentication
func (a *Authenticator) Authenticate(ctx context.Context) (*oauth2.Token, error) {
	// Generate cryptographically secure random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Generate authorization URL for manual flow
	// Note: For a proper device flow, you'd use the device endpoint
	// This is a simplified version using the standard authorization code flow
	authURL := a.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("\nTo authorize Syncrules to access Google Drive:\n\n")
	fmt.Printf("1. Visit this URL:\n   %s\n\n", authURL)
	fmt.Printf("2. Sign in with your Google account and authorize the application\n\n")
	fmt.Printf("3. Copy the authorization code and paste it below\n\n")
	fmt.Printf("Enter authorization code: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}

	token, err := a.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save token for future use
	if err := a.saveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\nAuthentication successful! Token saved.")
	return token, nil
}

// RefreshToken refreshes an expired token
func (a *Authenticator) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := a.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save refreshed token
	if err := a.saveToken(newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return newToken, nil
}

// loadToken loads a token from file
func (a *Authenticator) loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(a.tokenPath)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("invalid token file: %w", err)
	}

	return token.toOAuth2Token(), nil
}

// saveToken saves a token to file atomically using temp file + rename
func (a *Authenticator) saveToken(token *oauth2.Token) error {
	// Ensure directory exists with restricted permissions
	dir := filepath.Dir(a.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	t := fromOAuth2Token(token)
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first for atomic operation
	tempPath := a.tokenPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp token file: %w", err)
	}

	// Atomic rename to final path
	if err := os.Rename(tempPath, a.tokenPath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename token file: %w", err)
	}

	return nil
}

// TokenPath returns the path where the token is stored
func (a *Authenticator) TokenPath() string {
	return a.tokenPath
}

// Config returns the OAuth2 config
func (a *Authenticator) Config() *oauth2.Config {
	return a.config
}
