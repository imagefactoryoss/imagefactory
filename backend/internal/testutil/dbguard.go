package testutil

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

// RequireSafeTestDSN fails the test when DSN appears to target a non-test database.
// Override only when intentional by setting IF_ALLOW_NON_TEST_DB=true.
func RequireSafeTestDSN(t testing.TB, dsn, envVar string) {
	t.Helper()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("IF_ALLOW_NON_TEST_DB")), "true") {
		return
	}

	dbName, schema := extractDSNParts(strings.TrimSpace(dsn))
	if isLikelyTestName(dbName) || isLikelyTestName(schema) {
		return
	}

	t.Fatalf(
		"refusing to run DB test against non-test DSN from %s (db=%q schema=%q). "+
			"Use a *_test database/schema or set IF_ALLOW_NON_TEST_DB=true to override.",
		envVar,
		dbName,
		schema,
	)
}

func isLikelyTestName(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v != "" && strings.Contains(v, "test")
}

func extractDSNParts(dsn string) (dbName string, schema string) {
	if dsn == "" {
		return "", ""
	}

	// URL DSN: postgres://user:pass@host:5432/dbname?search_path=...
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		dbName = strings.TrimPrefix(u.Path, "/")
		q := u.Query()
		schema = q.Get("search_path")
		if schema == "" {
			schema = q.Get("schema")
		}
		return dbName, schema
	}

	// key=value DSN: host=... dbname=... search_path=...
	for _, token := range strings.Fields(dsn) {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		switch key {
		case "dbname", "database":
			dbName = val
		case "search_path", "schema":
			schema = val
		}
	}
	return dbName, schema
}

