package token

import (
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "Generate with default length",
			length:  32,
			wantErr: false,
		},
		{
			name:    "Generate with custom length",
			length:  64,
			wantErr: false,
		},
		{
			name:    "Generate with zero length (should use default)",
			length:  0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateToken() returned empty token")
				}
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	password, err := GeneratePassword()
	if err != nil {
		t.Errorf("GeneratePassword() error = %v", err)
	}

	if password == "" {
		t.Error("GeneratePassword() returned empty password")
	}

	// Verify uniqueness by generating multiple passwords
	password2, err := GeneratePassword()
	if err != nil {
		t.Errorf("GeneratePassword() error = %v", err)
	}

	if password == password2 {
		t.Error("GeneratePassword() returned identical passwords")
	}
}

func TestGenerateUsername(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "Generate with custom prefix",
			prefix:     "app",
			wantPrefix: "app-",
			wantErr:    false,
		},
		{
			name:       "Generate with empty prefix (should use default)",
			prefix:     "",
			wantPrefix: "user-",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := GenerateUsername(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if username == "" {
					t.Error("GenerateUsername() returned empty username")
				}

				if !strings.HasPrefix(username, tt.wantPrefix) {
					t.Errorf("GenerateUsername() = %v, want prefix %v", username, tt.wantPrefix)
				}
			}
		})
	}
}
