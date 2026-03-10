package ldap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

// Config represents LDAP configuration
type Config struct {
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
	BaseDN       string `json:"base_dn" yaml:"base_dn"`
	BindDN       string `json:"bind_dn" yaml:"bind_dn"`
	BindPassword string `json:"bind_password" yaml:"bind_password"`
	UserFilter   string `json:"user_filter" yaml:"user_filter"`
	GroupFilter  string `json:"group_filter" yaml:"group_filter"`
	UseTLS       bool   `json:"use_tls" yaml:"use_tls"`
	StartTLS     bool   `json:"start_tls" yaml:"start_tls"`
}

// UserInfo represents user information from LDAP
type UserInfo struct {
	DN          string
	Username    string
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
	Groups      []string
}

// Client represents LDAP client
type Client struct {
	config *Config
	logger *zap.Logger
}

// NewClient creates a new LDAP client
func NewClient(config *Config, logger *zap.Logger) *Client {
	return &Client{
		config: config,
		logger: logger,
	}
}

// Authenticate authenticates a user against LDAP
func (c *Client) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	c.logger.Info("Starting LDAP authentication", zap.String("username", username))

	if username == "" || password == "" {
		c.logger.Warn("LDAP authentication failed: missing credentials")
		return nil, errors.New("username and password are required")
	}

	conn, err := c.connect()
	if err != nil {
		c.logger.Error("LDAP connection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	c.logger.Debug("LDAP connection established successfully")

	// Try direct bind first (e.g., for GLauth where search is restricted)
	// For GLauth, try multiple DN patterns since it's flexible:
	// 1. uid=<username>,ou=people,<baseDN>
	// 2. uid=<username>,<baseDN>
	// 3. cn=<username>,<baseDN>
	dnPatterns := []string{
		fmt.Sprintf("uid=%s,ou=people,%s", username, c.config.BaseDN),
		fmt.Sprintf("uid=%s,%s", username, c.config.BaseDN),
		fmt.Sprintf("cn=%s,%s", username, c.config.BaseDN),
	}

	c.logger.Debug("Trying direct bind with DN patterns", zap.Strings("patterns", dnPatterns))

	for _, userDN := range dnPatterns {
		c.logger.Debug("Attempting bind with DN", zap.String("dn", userDN))
		if err := conn.Bind(userDN, password); err == nil {
			// Direct bind succeeded - now fetch user details using service account
			c.logger.Debug("Direct bind successful, fetching user details", zap.String("user_dn", userDN))
			
			var userInfo *UserInfo
			
			// Try to fetch full user details with service account if available
			if c.config.BindDN != "" && c.config.BindPassword != "" && c.config.UserFilter != "" {
				// Rebind with service account to search for user details
				if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
					c.logger.Warn("Failed to bind with service account for attribute lookup",
						zap.String("bind_dn", c.config.BindDN),
						zap.Error(err))
				} else {
					// Search for user to get full details
					_, fetchedInfo, err := c.searchUser(conn, username)
					if err == nil && fetchedInfo != nil {
						userInfo = fetchedInfo
						c.logger.Debug("User details fetched from LDAP",
							zap.String("email", userInfo.Email),
							zap.String("first_name", userInfo.FirstName),
							zap.String("last_name", userInfo.LastName))
					} else {
						c.logger.Warn("Failed to fetch user details via search",
							zap.String("username", username),
							zap.Error(err))
					}
				}
			}
			
			// If we couldn't fetch from LDAP, use minimal info
			if userInfo == nil {
				userInfo = &UserInfo{
					DN:       userDN,
					Username: username,
				}
				c.logger.Debug("Using minimal user info (no LDAP search available)",
					zap.String("username", username))
			}
			
			c.logger.Info("LDAP direct authentication successful",
				zap.String("username", username),
				zap.String("user_dn", userDN))
			return userInfo, nil
		}
		c.logger.Debug("Failed to bind with DN pattern",
			zap.String("pattern", userDN),
			zap.String("username", username),
			zap.Error(err))
	}

	// If direct bind fails and we have search capability, try searching
	if c.config.UserFilter != "" {
		c.logger.Debug("Direct bind failed, trying search-based authentication")

		// Bind with service account for search
		c.logger.Debug("Binding with service account", zap.String("bind_dn", c.config.BindDN))
		if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
			c.logger.Error("Failed to bind with service account",
				zap.String("bind_dn", c.config.BindDN),
				zap.Error(err))
			return nil, errors.New("invalid credentials")
		}
		c.logger.Debug("Service account bind successful")

		// Search for user
		userDN, userInfo, err := c.searchUser(conn, username)
		if err != nil {
			c.logger.Warn("User search failed",
				zap.String("username", username),
				zap.Error(err))
			return nil, errors.New("invalid credentials")
		}

		c.logger.Debug("User found via search", zap.String("user_dn", userDN))

		// Try to bind with user's DN and password
		if err := conn.Bind(userDN, password); err != nil {
			c.logger.Warn("User bind failed",
				zap.String("username", username),
				zap.String("user_dn", userDN),
				zap.Error(err))
			return nil, errors.New("invalid credentials")
		}

		c.logger.Debug("User password verification successful")

		// Get user's groups
		groups, err := c.getUserGroups(conn, userDN)
		if err != nil {
			c.logger.Warn("Failed to get user groups",
				zap.String("username", username),
				zap.Error(err))
			// Don't fail authentication if group lookup fails
			groups = []string{}
		}

		userInfo.Groups = groups

		c.logger.Info("LDAP search-based authentication successful",
			zap.String("username", username),
			zap.String("user_dn", userDN))

		return userInfo, nil
	}

	// Both methods failed
	c.logger.Warn("LDAP authentication failed: all methods exhausted", zap.String("username", username))
	return nil, errors.New("invalid credentials")
}

// SearchUser searches for a user in LDAP
func (c *Client) SearchUser(ctx context.Context, username string) (*UserInfo, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	// Bind with service account
	if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return nil, fmt.Errorf("authentication service unavailable")
	}

	_, userInfo, err := c.searchUser(conn, username)
	return userInfo, err
}

// SearchUsers searches for users in LDAP matching a query
func (c *Client) SearchUsers(ctx context.Context, query string, limit int) ([]*UserInfo, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	// Bind with service account
	if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return nil, fmt.Errorf("authentication service unavailable")
	}

	return c.searchUsers(conn, query, limit)
}

// connect establishes LDAP connection
func (c *Client) connect() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if c.config.UseTLS {
		conn, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
			&tls.Config{InsecureSkipVerify: true}) // TODO: Configure proper TLS
	} else {
		conn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", c.config.Host, c.config.Port))
		if err != nil {
			return nil, err
		}

		if c.config.StartTLS {
			err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true}) // TODO: Configure proper TLS
			if err != nil {
				conn.Close()
				return nil, err
			}
		}
	}

	return conn, err
}

// searchUser searches for a user and returns DN and user info
func (c *Client) searchUser(conn *ldap.Conn, username string) (string, *UserInfo, error) {
	filter := fmt.Sprintf(c.config.UserFilter, username)

	c.logger.Debug("Searching for user",
		zap.String("filter", filter),
		zap.String("base_dn", c.config.BaseDN))

	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,     // SizeLimit
		0,     // TimeLimit
		false, // TypesOnly
		filter,
		[]string{"dn", "cn", "sn", "givenName", "mail", "displayName", "sAMAccountName", "uid"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		c.logger.Error("LDAP search failed", zap.Error(err))
		return "", nil, err
	}

	c.logger.Debug("LDAP search completed", zap.Int("entries_found", len(result.Entries)))

	if len(result.Entries) == 0 {
		c.logger.Warn("User not found in LDAP", zap.String("username", username))
		return "", nil, errors.New("user not found")
	}

	if len(result.Entries) > 1 {
		c.logger.Warn("Multiple users found in LDAP", zap.String("username", username), zap.Int("count", len(result.Entries)))
		return "", nil, errors.New("multiple users found")
	}

	entry := result.Entries[0]

	c.logger.Debug("User entry found",
		zap.String("dn", entry.DN),
		zap.String("username", username))

	userInfo := &UserInfo{
		DN:          entry.DN,
		Username:    c.getAttributeValue(entry, "sAMAccountName", "uid"),
		Email:       c.getAttributeValue(entry, "mail"),
		FirstName:   c.getAttributeValue(entry, "givenName"),
		LastName:    c.getAttributeValue(entry, "sn"),
		DisplayName: c.getAttributeValue(entry, "displayName", "cn"),
	}

	c.logger.Debug("User info extracted",
		zap.String("email", userInfo.Email),
		zap.String("first_name", userInfo.FirstName),
		zap.String("last_name", userInfo.LastName))

	return entry.DN, userInfo, nil
}

// searchUsers searches for multiple users matching a query
func (c *Client) searchUsers(conn *ldap.Conn, query string, limit int) ([]*UserInfo, error) {
	// Try a simpler filter first - search by uid, mail (primary identifier in GLAuth)
	// Use a more flexible objectClass filter to support different LDAP schemas (posixAccount, inetOrgPerson, person, etc.)
	filter := fmt.Sprintf("(|(uid=*%s*)(mail=*%s*))", query, query)

	c.logger.Debug("Searching for users with simple filter",
		zap.String("filter", filter),
		zap.String("base_dn", c.config.BaseDN),
		zap.Int("limit", limit))

	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		limit,
		0,
		false,
		filter,
		[]string{"dn", "cn", "sn", "givenName", "mail", "displayName", "sAMAccountName", "uid", "objectClass"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		c.logger.Error("LDAP user search failed", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("Simple search completed", zap.Int("entries_found", len(result.Entries)))

	// If no results with simple filter, try broader search
	if len(result.Entries) == 0 {
		filter = fmt.Sprintf("(|(uid=*%s*)(mail=*%s*)(givenName=*%s*)(sn=*%s*)(cn=*%s*))",
			query, query, query, query, query)

		c.logger.Debug("Trying broader search",
			zap.String("filter", filter))

		searchRequest.Filter = filter
		result, err = conn.Search(searchRequest)
		if err != nil {
			c.logger.Error("LDAP broader search failed", zap.Error(err))
			return nil, err
		}

		c.logger.Debug("Broader search completed", zap.Int("entries_found", len(result.Entries)))
	}

	users := make([]*UserInfo, 0, len(result.Entries))
	for _, entry := range result.Entries {
		userInfo := &UserInfo{
			DN:          entry.DN,
			Username:    c.getAttributeValue(entry, "sAMAccountName", "uid"),
			Email:       c.getAttributeValue(entry, "mail"),
			FirstName:   c.getAttributeValue(entry, "givenName"),
			LastName:    c.getAttributeValue(entry, "sn"),
			DisplayName: c.getAttributeValue(entry, "displayName", "cn"),
		}

		c.logger.Debug("Found user in search",
			zap.String("dn", entry.DN),
			zap.String("cn", c.getAttributeValue(entry, "cn")),
			zap.String("uid", c.getAttributeValue(entry, "uid")),
			zap.String("mail", c.getAttributeValue(entry, "mail")),
			zap.String("givenName", c.getAttributeValue(entry, "givenName")),
			zap.String("sn", c.getAttributeValue(entry, "sn")),
			zap.String("username", userInfo.Username),
			zap.String("email", userInfo.Email))

		users = append(users, userInfo)
	}

	return users, nil
}

// getUserGroups retrieves user's group memberships
func (c *Client) getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
	filter := fmt.Sprintf("(&(objectClass=group)(member=%s))", userDN)

	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,     // SizeLimit
		0,     // TimeLimit
		false, // TypesOnly
		filter,
		[]string{"cn", "dn"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	groups := make([]string, len(result.Entries))
	for i, entry := range result.Entries {
		groups[i] = c.getAttributeValue(entry, "cn")
	}

	return groups, nil
}

// getAttributeValue gets the first value of an attribute
func (c *Client) getAttributeValue(entry *ldap.Entry, attrs ...string) string {
	for _, attr := range attrs {
		if values := entry.GetAttributeValues(attr); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// Close closes the LDAP client (no-op for now)
func (c *Client) Close() error {
	return nil
}
