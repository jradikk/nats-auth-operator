package authconf

import (
	"strings"
	"testing"

	natsv1alpha1 "github.com/jradikk/nats-auth-operator/api/v1alpha1"
)

func TestRenderTokenAuthConf(t *testing.T) {
	tests := []struct {
		name  string
		users []TokenUser
		want  []string // Expected strings to be present in output
	}{
		{
			name:  "Empty user list",
			users: []TokenUser{},
			want:  []string{},
		},
		{
			name: "Single user with password",
			users: []TokenUser{
				{
					Username: "testuser",
					Password: "testpass",
				},
			},
			want: []string{"authorization", "users", "testuser", "testpass"},
		},
		{
			name: "User with token",
			users: []TokenUser{
				{
					Username: "tokenuser",
					Token:    "abc123",
				},
			},
			want: []string{"authorization", "users", "tokenuser", "token", "abc123"},
		},
		{
			name: "User with permissions",
			users: []TokenUser{
				{
					Username: "permuser",
					Password: "pass",
					Permissions: &natsv1alpha1.Permissions{
						PublishAllow:   []string{"foo.>", "bar.>"},
						SubscribeAllow: []string{"baz.>"},
					},
				},
			},
			want: []string{"permissions", "publish", "allow", "foo.>", "subscribe", "baz.>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderTokenAuthConf(tt.users)

			if len(tt.users) == 0 {
				if got != "" {
					t.Errorf("RenderTokenAuthConf() with empty users should return empty string, got %v", got)
				}
				return
			}

			for _, expected := range tt.want {
				if !strings.Contains(got, expected) {
					t.Errorf("RenderTokenAuthConf() output missing expected string %q\nGot:\n%s", expected, got)
				}
			}
		})
	}
}

func TestRenderJWTAuthConf(t *testing.T) {
	operatorJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ..."
	resolverDir := "/var/lib/nats-resolver"

	output := RenderJWTAuthConf(operatorJWT, resolverDir)

	expectedStrings := []string{
		"operator:",
		operatorJWT,
		"resolver:",
		"type: full",
		"dir:",
		resolverDir,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("RenderJWTAuthConf() output missing expected string %q\nGot:\n%s", expected, output)
		}
	}
}

func TestRenderMixedAuthConf(t *testing.T) {
	operatorJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ..."
	resolverDir := "/var/lib/nats-resolver"
	tokenUsers := []TokenUser{
		{
			Username: "mixeduser",
			Password: "mixedpass",
		},
	}

	output := RenderMixedAuthConf(operatorJWT, resolverDir, tokenUsers)

	// Should contain both JWT and token auth sections
	expectedStrings := []string{
		"operator:",
		"resolver:",
		"authorization",
		"users",
		"mixeduser",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("RenderMixedAuthConf() output missing expected string %q\nGot:\n%s", expected, output)
		}
	}
}

func TestFormatSubjectList(t *testing.T) {
	tests := []struct {
		name     string
		subjects []string
		want     string
	}{
		{
			name:     "Single subject",
			subjects: []string{"foo.>"},
			want:     `"foo.>"`,
		},
		{
			name:     "Multiple subjects",
			subjects: []string{"foo.>", "bar.>", "baz.>"},
			want:     `["foo.>", "bar.>", "baz.>"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSubjectList(tt.subjects)
			if got != tt.want {
				t.Errorf("formatSubjectList() = %v, want %v", got, tt.want)
			}
		})
	}
}
