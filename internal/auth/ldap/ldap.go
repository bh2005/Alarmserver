// Package ldap implements LDAP/Active Directory authentication for the UI login.
// When configured, it is tried as a fallback if local DB auth fails.
package ldap

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

// Config holds the LDAP connection and search parameters.
type Config struct {
	Host         string
	Port         int
	TLS          bool
	BindDN       string
	BindPassword string
	UserBase     string
	UserFilter   string // e.g. (&(objectClass=user)(sAMAccountName={username}))
	AdminGroup   string // DN of an AD group whose members receive role=admin
	DefaultRole  string // "viewer" or "admin" for all other LDAP users
}

// Authenticator performs a two-step LDAP bind: first with the service account
// to locate the user DN, then with the user's own credentials to verify.
type Authenticator struct {
	cfg Config
}

func New(cfg Config) *Authenticator {
	if cfg.Port == 0 {
		if cfg.TLS {
			cfg.Port = 636
		} else {
			cfg.Port = 389
		}
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "viewer"
	}
	return &Authenticator{cfg: cfg}
}

// Authenticate returns the configured default role on success or an error.
func (a *Authenticator) Authenticate(_ context.Context, username, password string) (string, error) {
	addr := fmt.Sprintf("%s:%d", a.cfg.Host, a.cfg.Port)

	var (
		conn *ldap.Conn
		err  error
	)
	if a.cfg.TLS {
		conn, err = ldap.DialURL("ldaps://" + addr)
	} else {
		conn, err = ldap.DialURL("ldap://" + addr)
	}
	if err != nil {
		return "", fmt.Errorf("ldap connect %s: %w", addr, err)
	}
	defer conn.Close()

	// Bind with the service account to search the directory.
	if err := conn.Bind(a.cfg.BindDN, a.cfg.BindPassword); err != nil {
		return "", fmt.Errorf("ldap service bind: %w", err)
	}

	filter := strings.ReplaceAll(a.cfg.UserFilter, "{username}", ldap.EscapeFilter(username))

	// Fetch memberOf so we can check group membership without a second request.
	attrs := []string{"dn"}
	if a.cfg.AdminGroup != "" {
		attrs = []string{"dn", "memberOf"}
	}
	req := ldap.NewSearchRequest(
		a.cfg.UserBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		2,    // sizeLimit — more than 1 = ambiguous
		0,    // timeLimit
		false,
		filter,
		attrs,
		nil,
	)
	result, err := conn.Search(req)
	if err != nil {
		return "", fmt.Errorf("ldap search: %w", err)
	}
	if len(result.Entries) != 1 {
		return "", fmt.Errorf("ldap: user not found")
	}
	entry := result.Entries[0]

	// Verify the user's password by re-binding as them.
	if err := conn.Bind(entry.DN, password); err != nil {
		return "", fmt.Errorf("ldap: invalid credentials")
	}

	// Determine role: admin if user is member of AdminGroup, else DefaultRole.
	role := a.cfg.DefaultRole
	if a.cfg.AdminGroup != "" {
		for _, val := range entry.GetAttributeValues("memberOf") {
			if strings.EqualFold(val, a.cfg.AdminGroup) {
				role = "admin"
				break
			}
		}
	}
	return role, nil
}
