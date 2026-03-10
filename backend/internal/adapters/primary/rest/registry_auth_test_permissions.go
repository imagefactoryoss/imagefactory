package rest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type TestRegistryAuthPermissionsRequest struct {
	RegistryRepo string `json:"registry_repo"`
}

type TestRegistryAuthPermissionsResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	RegistryHost string `json:"registry_host,omitempty"`
	RegistryRepo string `json:"registry_repo,omitempty"`
}

func (h *RegistryAuthHandler) TestPermissions(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var req TestRegistryAuthPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.RegistryRepo) == "" {
		http.Error(w, "registry_repo is required", http.StatusBadRequest)
		return
	}

	auth, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == registryauth.ErrRegistryAuthNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load registry authentication", http.StatusInternalServerError)
		return
	}
	if auth.TenantID != authCtx.TenantID {
		http.Error(w, "Registry authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	registryHost, repoPath, err := parseRegistryRepo(req.RegistryRepo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !hostsCompatible(auth.RegistryHost, registryHost) {
		writeRegistryPermissionResponse(w, http.StatusOK, TestRegistryAuthPermissionsResponse{
			Success:      false,
			Message:      fmt.Sprintf("Registry host mismatch: auth host %q does not match repo host %q", auth.RegistryHost, registryHost),
			RegistryHost: registryHost,
			RegistryRepo: repoPath,
		})
		return
	}

	credentials, err := h.service.DecryptCredentials(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to decrypt registry credentials", http.StatusInternalServerError)
		return
	}

	username, password, bearerToken, err := credentialsForHost(auth, credentials, registryHost)
	if err != nil {
		writeRegistryPermissionResponse(w, http.StatusOK, TestRegistryAuthPermissionsResponse{
			Success:      false,
			Message:      err.Error(),
			RegistryHost: registryHost,
			RegistryRepo: repoPath,
		})
		return
	}

	ok, message := testRegistryPushPermission(r.Context(), registryHost, repoPath, username, password, bearerToken)
	writeRegistryPermissionResponse(w, http.StatusOK, TestRegistryAuthPermissionsResponse{
		Success:      ok,
		Message:      message,
		RegistryHost: registryHost,
		RegistryRepo: repoPath,
	})
}

func writeRegistryPermissionResponse(w http.ResponseWriter, code int, payload TestRegistryAuthPermissionsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseRegistryRepo(input string) (string, string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", "", fmt.Errorf("registry_repo is required")
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", "", fmt.Errorf("invalid registry_repo: %w", err)
		}
		host := strings.TrimSpace(parsed.Host)
		repo := strings.Trim(strings.TrimSpace(parsed.Path), "/")
		if host == "" || repo == "" {
			return "", "", fmt.Errorf("registry_repo must include host and repository path")
		}
		return host, repo, nil
	}

	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("registry_repo must be in format host/repository")
	}
	return strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "/"), nil
}

func hostsCompatible(authHost, repoHost string) bool {
	return normalizeRegistryHost(authHost) == normalizeRegistryHost(repoHost)
}

func normalizeRegistryHost(host string) string {
	value := strings.TrimSpace(strings.ToLower(host))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimSuffix(value, "/")
	return value
}

func credentialsForHost(auth *registryauth.RegistryAuth, credentials map[string]interface{}, registryHost string) (string, string, string, error) {
	switch auth.AuthType {
	case registryauth.AuthTypeBasicAuth:
		username := asString(credentials["username"])
		password := asString(credentials["password"])
		if username == "" || password == "" {
			return "", "", "", fmt.Errorf("registry auth is missing username/password")
		}
		return username, password, "", nil
	case registryauth.AuthTypeToken:
		token := asString(credentials["token"])
		if token == "" {
			return "", "", "", fmt.Errorf("registry auth is missing token")
		}
		username := asString(credentials["username"])
		if username == "" {
			username = "token"
		}
		return username, token, "", nil
	case registryauth.AuthTypeDockerConfigJSON:
		raw := asString(credentials["dockerconfigjson"])
		if strings.TrimSpace(raw) == "" {
			return "", "", "", fmt.Errorf("registry auth is missing dockerconfigjson credential")
		}
		return parseDockerConfigJSONCredentials(raw, registryHost, auth.RegistryHost)
	default:
		return "", "", "", fmt.Errorf("unsupported registry auth type %q", auth.AuthType)
	}
}

func parseDockerConfigJSONCredentials(raw, registryHost, configuredHost string) (string, string, string, error) {
	type dockerAuthEntry struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		Auth          string `json:"auth"`
		IdentityToken string `json:"identitytoken"`
	}
	var parsed struct {
		Auths map[string]dockerAuthEntry `json:"auths"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", "", "", fmt.Errorf("invalid dockerconfigjson: %w", err)
	}
	if len(parsed.Auths) == 0 {
		return "", "", "", fmt.Errorf("dockerconfigjson has no auth entries")
	}

	candidates := []string{
		normalizeRegistryHost(registryHost),
		normalizeRegistryHost(configuredHost),
		"https://" + normalizeRegistryHost(registryHost),
		"https://" + normalizeRegistryHost(configuredHost),
		"http://" + normalizeRegistryHost(registryHost),
		"http://" + normalizeRegistryHost(configuredHost),
	}

	var entry dockerAuthEntry
	found := false
	for key, value := range parsed.Auths {
		normalizedKey := normalizeRegistryHost(key)
		for _, candidate := range candidates {
			if normalizeRegistryHost(candidate) == normalizedKey {
				entry = value
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		for _, value := range parsed.Auths {
			entry = value
			found = true
			break
		}
	}
	if !found {
		return "", "", "", fmt.Errorf("no usable credential entry found in dockerconfigjson")
	}

	if entry.Username != "" && entry.Password != "" {
		return entry.Username, entry.Password, "", nil
	}
	if entry.IdentityToken != "" {
		return "", "", entry.IdentityToken, nil
	}
	if entry.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid dockerconfigjson auth field: %w", err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", "", fmt.Errorf("dockerconfigjson auth must decode to username:password")
		}
		return parts[0], parts[1], "", nil
	}
	return "", "", "", fmt.Errorf("dockerconfigjson entry does not include username/password or auth token")
}

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func testRegistryPushPermission(ctx context.Context, registryHost, repoPath, username, password, bearerToken string) (bool, string) {
	uploadURL := fmt.Sprintf("https://%s/v2/%s/blobs/uploads/", registryHost, repoPath)

	status, body, location, challenge, err := doRegistryUploadAttempt(ctx, uploadURL, username, password, bearerToken)
	if err != nil {
		return false, fmt.Sprintf("failed to test registry permissions: %v", err)
	}
	if status == http.StatusAccepted {
		cleanupUploadSession(ctx, registryHost, location, username, password, bearerToken)
		return true, "Registry authentication validated with push permissions."
	}

	if status == http.StatusUnauthorized && strings.HasPrefix(strings.ToLower(challenge), "bearer ") {
		token, tokenErr := exchangeRegistryBearerToken(ctx, challenge, username, password, bearerToken, repoPath)
		if tokenErr != nil {
			return false, fmt.Sprintf("token exchange failed: %v", tokenErr)
		}
		status, body, location, _, err = doRegistryUploadAttempt(ctx, uploadURL, "", "", token)
		if err != nil {
			return false, fmt.Sprintf("failed to retry upload permission test: %v", err)
		}
		if status == http.StatusAccepted {
			cleanupUploadSession(ctx, registryHost, location, "", "", token)
			return true, "Registry authentication validated with push permissions."
		}
	}

	if strings.TrimSpace(body) != "" {
		return false, fmt.Sprintf("registry denied push permission (HTTP %d): %s", status, strings.TrimSpace(body))
	}
	return false, fmt.Sprintf("registry denied push permission (HTTP %d)", status)
}

func doRegistryUploadAttempt(ctx context.Context, uploadURL, username, password, bearerToken string) (int, string, string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, nil)
	if err != nil {
		return 0, "", "", "", err
	}
	applyRegistryAuth(req, username, password, bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	return resp.StatusCode, string(body), resp.Header.Get("Location"), resp.Header.Get("WWW-Authenticate"), nil
}

func applyRegistryAuth(req *http.Request, username, password, bearerToken string) {
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
		return
	}
	if strings.TrimSpace(username) != "" || strings.TrimSpace(password) != "" {
		req.SetBasicAuth(username, password)
	}
}

func exchangeRegistryBearerToken(ctx context.Context, challenge, username, password, bearerToken, repoPath string) (string, error) {
	params := parseAuthChallenge(challenge)
	realm := strings.TrimSpace(params["realm"])
	if realm == "" {
		return "", fmt.Errorf("missing bearer realm in challenge")
	}

	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("invalid bearer realm: %w", err)
	}

	query := tokenURL.Query()
	if service := strings.TrimSpace(params["service"]); service != "" {
		query.Set("service", service)
	}
	query.Set("scope", fmt.Sprintf("repository:%s:pull,push", repoPath))
	tokenURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	applyRegistryAuth(req, username, password, bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("token endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("invalid token response: %w", err)
	}
	token := strings.TrimSpace(payload.Token)
	if token == "" {
		token = strings.TrimSpace(payload.AccessToken)
	}
	if token == "" {
		return "", fmt.Errorf("token response missing token")
	}
	return token, nil
}

func parseAuthChallenge(challenge string) map[string]string {
	out := map[string]string{}
	challenge = strings.TrimSpace(challenge)
	if challenge == "" {
		return out
	}
	if strings.HasPrefix(strings.ToLower(challenge), "bearer ") {
		challenge = strings.TrimSpace(challenge[len("Bearer "):])
	}
	parts := strings.Split(challenge, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		out[key] = val
	}
	return out
}

func cleanupUploadSession(ctx context.Context, registryHost, location, username, password, bearerToken string) {
	if strings.TrimSpace(location) == "" {
		return
	}
	deleteURL := strings.TrimSpace(location)
	if !strings.HasPrefix(deleteURL, "http://") && !strings.HasPrefix(deleteURL, "https://") {
		if !strings.HasPrefix(deleteURL, "/") {
			deleteURL = "/" + deleteURL
		}
		deleteURL = "https://" + registryHost + deleteURL
	}

	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return
	}
	applyRegistryAuth(req, username, password, bearerToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
