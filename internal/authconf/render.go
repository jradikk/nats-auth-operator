package authconf

import (
	"fmt"
	"strings"

	natsv1alpha1 "github.com/jradikk/nats-auth-operator/api/v1alpha1"
)

// TokenUser represents a token-based user for auth.conf
type TokenUser struct {
	Username    string
	Password    string
	Token       string
	Permissions *natsv1alpha1.Permissions
}

// RenderTokenAuthConf generates the authorization section for token-based auth
func RenderTokenAuthConf(users []TokenUser) string {
	if len(users) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("authorization {\n")
	sb.WriteString("  users = [\n")

	for i, user := range users {
		sb.WriteString("    {\n")

		// Add username
		if user.Username != "" {
			sb.WriteString(fmt.Sprintf("      user: %q\n", user.Username))
		}

		// Add password or token
		if user.Token != "" {
			sb.WriteString(fmt.Sprintf("      token: %q\n", user.Token))
		} else if user.Password != "" {
			sb.WriteString(fmt.Sprintf("      password: %q\n", user.Password))
		}

		// Add permissions if specified
		if user.Permissions != nil {
			sb.WriteString("      permissions: {\n")

			// Publish permissions
			if len(user.Permissions.PublishAllow) > 0 || len(user.Permissions.PublishDeny) > 0 {
				sb.WriteString("        publish: {\n")
				if len(user.Permissions.PublishAllow) > 0 {
					sb.WriteString(fmt.Sprintf("          allow: %s\n", formatSubjectList(user.Permissions.PublishAllow)))
				}
				if len(user.Permissions.PublishDeny) > 0 {
					sb.WriteString(fmt.Sprintf("          deny: %s\n", formatSubjectList(user.Permissions.PublishDeny)))
				}
				sb.WriteString("        }\n")
			}

			// Subscribe permissions
			if len(user.Permissions.SubscribeAllow) > 0 || len(user.Permissions.SubscribeDeny) > 0 {
				sb.WriteString("        subscribe: {\n")
				if len(user.Permissions.SubscribeAllow) > 0 {
					sb.WriteString(fmt.Sprintf("          allow: %s\n", formatSubjectList(user.Permissions.SubscribeAllow)))
				}
				if len(user.Permissions.SubscribeDeny) > 0 {
					sb.WriteString(fmt.Sprintf("          deny: %s\n", formatSubjectList(user.Permissions.SubscribeDeny)))
				}
				sb.WriteString("        }\n")
			}

			sb.WriteString("      }\n")
		}

		sb.WriteString("    }")
		if i < len(users)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  ]\n")
	sb.WriteString("}\n")

	return sb.String()
}

// formatSubjectList formats a list of subjects for the NATS config
func formatSubjectList(subjects []string) string {
	if len(subjects) == 1 {
		return fmt.Sprintf("%q", subjects[0])
	}

	quoted := make([]string, len(subjects))
	for i, s := range subjects {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// RenderJWTAuthConf generates the JWT resolver configuration
func RenderJWTAuthConf(operatorJWT, resolverDir string) string {
	var sb strings.Builder

	sb.WriteString("operator: ")
	sb.WriteString(operatorJWT)
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf(`resolver: {
  type: full
  dir: %q
  allow_delete: false
  interval: "2m"
}
`, resolverDir))

	return sb.String()
}

// AccountJWT represents an account JWT for preload
type AccountJWT struct {
	AccountName string // Kubernetes resource name
	AccountID   string // NATS public key (starts with AC...)
	JWT         string // The signed JWT
}

// RenderJWTAuthConfWithPreload generates JWT config with resolver_preload
// This eliminates the need for emptyDir or shared filesystem
func RenderJWTAuthConfWithPreload(operatorJWT string, accounts []AccountJWT) string {
	var sb strings.Builder

	sb.WriteString("operator: ")
	sb.WriteString(operatorJWT)
	sb.WriteString("\n\n")

	// Use resolver_preload instead of directory
	if len(accounts) > 0 {
		sb.WriteString("resolver_preload: {\n")
		for i, acc := range accounts {
			sb.WriteString(fmt.Sprintf("  %q: %q", acc.AccountID, acc.JWT))
			if i < len(accounts)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("}\n")
	}

	return sb.String()
}

// RenderMixedAuthConf generates configuration for mixed mode (both token and JWT)
func RenderMixedAuthConf(operatorJWT, resolverDir string, tokenUsers []TokenUser) string {
	var sb strings.Builder

	// JWT configuration
	sb.WriteString(RenderJWTAuthConf(operatorJWT, resolverDir))
	sb.WriteString("\n")

	// Token configuration
	sb.WriteString(RenderTokenAuthConf(tokenUsers))

	return sb.String()
}
