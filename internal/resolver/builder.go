package resolver

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

// Builder manages the NATS JWT resolver directory structure
type Builder struct {
	fs      afero.Fs
	baseDir string
}

// NewBuilder creates a new resolver builder
func NewBuilder(fs afero.Fs, baseDir string) *Builder {
	return &Builder{
		fs:      fs,
		baseDir: baseDir,
	}
}

// Initialize creates the resolver directory structure
func (b *Builder) Initialize() error {
	// Create base resolver directory
	if err := b.fs.MkdirAll(b.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create resolver directory: %w", err)
	}

	// Create accounts subdirectory
	accountsDir := filepath.Join(b.baseDir, "accounts")
	if err := b.fs.MkdirAll(accountsDir, 0755); err != nil {
		return fmt.Errorf("failed to create accounts directory: %w", err)
	}

	return nil
}

// WriteOperatorJWT writes the operator JWT to the resolver directory
func (b *Builder) WriteOperatorJWT(operatorJWT string) error {
	operatorPath := filepath.Join(b.baseDir, "operator.jwt")
	if err := afero.WriteFile(b.fs, operatorPath, []byte(operatorJWT), 0644); err != nil {
		return fmt.Errorf("failed to write operator JWT: %w", err)
	}
	return nil
}

// WriteAccountJWT writes an account JWT to the resolver directory
func (b *Builder) WriteAccountJWT(accountID, accountJWT string) error {
	accountPath := filepath.Join(b.baseDir, "accounts", fmt.Sprintf("%s.jwt", accountID))
	if err := afero.WriteFile(b.fs, accountPath, []byte(accountJWT), 0644); err != nil {
		return fmt.Errorf("failed to write account JWT: %w", err)
	}
	return nil
}

// DeleteAccountJWT removes an account JWT from the resolver directory
func (b *Builder) DeleteAccountJWT(accountID string) error {
	accountPath := filepath.Join(b.baseDir, "accounts", fmt.Sprintf("%s.jwt", accountID))
	exists, err := afero.Exists(b.fs, accountPath)
	if err != nil {
		return fmt.Errorf("failed to check account JWT existence: %w", err)
	}
	if !exists {
		return nil
	}
	if err := b.fs.Remove(accountPath); err != nil {
		return fmt.Errorf("failed to delete account JWT: %w", err)
	}
	return nil
}

// GetResolverConfig generates the resolver configuration for NATS server
func (b *Builder) GetResolverConfig() string {
	return fmt.Sprintf(`resolver: {
    type: full
    dir: %s
    allow_delete: false
    interval: "2m"
}
`, b.baseDir)
}

// GetOperatorJWTPath returns the path to the operator JWT
func (b *Builder) GetOperatorJWTPath() string {
	return filepath.Join(b.baseDir, "operator.jwt")
}

// GetAccountJWTPath returns the path to an account JWT
func (b *Builder) GetAccountJWTPath(accountID string) string {
	return filepath.Join(b.baseDir, "accounts", fmt.Sprintf("%s.jwt", accountID))
}

// AccountJWTExists checks if an account JWT exists
func (b *Builder) AccountJWTExists(accountID string) (bool, error) {
	accountPath := b.GetAccountJWTPath(accountID)
	return afero.Exists(b.fs, accountPath)
}
