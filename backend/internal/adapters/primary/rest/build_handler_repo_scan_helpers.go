package rest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type dockerfileScoreCandidate struct {
	path    string
	context string
	score   int
}

func scanRepositoryBuildStructure(ctx context.Context, repoURL, ref string, gitAuthSecret map[string][]byte) ([]BuildContextPathSuggestion, []BuildDockerfilePathSuggestion, error) {
	tmpDir, err := os.MkdirTemp("", "if-repo-structure-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	effectiveURL, env, cleanup, err := prepareGitCloneAuth(repoURL, gitAuthSecret)
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	cloneArgs := []string{"clone", "--depth", "1", "--filter=blob:none", "--no-checkout"}
	if strings.TrimSpace(ref) != "" {
		cloneArgs = append(cloneArgs, "--branch", ref)
	}
	cloneArgs = append(cloneArgs, effectiveURL, repoDir)

	if err := runGitCommand(ctx, env, cloneArgs...); err != nil {
		// Fallback for branch/ref mismatches: retry default branch clone when ref clone fails.
		if strings.TrimSpace(ref) == "" {
			return nil, nil, err
		}
		fallbackArgs := []string{"clone", "--depth", "1", "--filter=blob:none", "--no-checkout", effectiveURL, repoDir}
		if fallbackErr := runGitCommand(ctx, env, fallbackArgs...); fallbackErr != nil {
			return nil, nil, err
		}
	}

	output, err := runGitCommandOutput(ctx, env, "-C", repoDir, "ls-tree", "-r", "--name-only", "HEAD")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list repository files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	contextScores := map[string]int{
		".": 10,
	}
	contextReasons := map[string]map[string]struct{}{
		".": {"Repository root": {}},
	}
	dockerfileCandidates := map[string]dockerfileScoreCandidate{}

	addContextReason := func(path, reason string, score int) {
		normalized := normalizeRepoPath(path)
		if normalized == "" {
			normalized = "."
		}
		contextScores[normalized] += score
		if _, ok := contextReasons[normalized]; !ok {
			contextReasons[normalized] = map[string]struct{}{}
		}
		contextReasons[normalized][reason] = struct{}{}
	}

	contextMarkers := map[string]struct {
		reason string
		score  int
	}{
		"go.mod":           {reason: "Contains go.mod", score: 20},
		"package.json":     {reason: "Contains package.json", score: 20},
		"project.toml":     {reason: "Contains project.toml (Paketo/Buildpacks)", score: 25},
		"pyproject.toml":   {reason: "Contains pyproject.toml", score: 20},
		"requirements.txt": {reason: "Contains requirements.txt", score: 15},
		"cargo.toml":       {reason: "Contains Cargo.toml", score: 20},
		"pom.xml":          {reason: "Contains pom.xml", score: 20},
		"build.gradle":     {reason: "Contains build.gradle", score: 20},
		"makefile":         {reason: "Contains Makefile", score: 10},
	}

	for _, rawPath := range lines {
		path := normalizeRepoPath(rawPath)
		if path == "" {
			continue
		}
		base := strings.ToLower(filepath.Base(path))
		dir := normalizeRepoPath(filepath.Dir(path))
		if dir == "" {
			dir = "."
		}

		if strings.Contains(base, "dockerfile") {
			score := 80
			if base == "dockerfile" {
				score = 100
			}
			dockerfileCandidates[path] = dockerfileScoreCandidate{
				path:    path,
				context: dir,
				score:   score,
			}
			addContextReason(dir, "Contains Dockerfile", 40)
		}

		if marker, ok := contextMarkers[base]; ok {
			addContextReason(dir, marker.reason, marker.score)
		}
	}

	contexts := make([]BuildContextPathSuggestion, 0, len(contextScores))
	for path, score := range contextScores {
		reasons := make([]string, 0, len(contextReasons[path]))
		for reason := range contextReasons[path] {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)
		contexts = append(contexts, BuildContextPathSuggestion{
			Path:   path,
			Reason: strings.Join(reasons, "; "),
			Score:  score,
		})
	}
	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].Score != contexts[j].Score {
			return contexts[i].Score > contexts[j].Score
		}
		return contexts[i].Path < contexts[j].Path
	})
	if len(contexts) > 15 {
		contexts = contexts[:15]
	}

	dockerfiles := make([]BuildDockerfilePathSuggestion, 0, len(dockerfileCandidates))
	for _, candidate := range dockerfileCandidates {
		dockerfiles = append(dockerfiles, BuildDockerfilePathSuggestion{
			Path:    candidate.path,
			Context: candidate.context,
			Score:   candidate.score,
		})
	}
	sort.Slice(dockerfiles, func(i, j int) bool {
		if dockerfiles[i].Score != dockerfiles[j].Score {
			return dockerfiles[i].Score > dockerfiles[j].Score
		}
		return dockerfiles[i].Path < dockerfiles[j].Path
	})
	if len(dockerfiles) > 20 {
		dockerfiles = dockerfiles[:20]
	}

	return contexts, dockerfiles, nil
}

func prepareGitCloneAuth(repoURL string, gitAuthSecret map[string][]byte) (effectiveURL string, env []string, cleanup func(), err error) {
	effectiveURL = strings.TrimSpace(repoURL)
	env = os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	cleanup = func() {}

	if len(gitAuthSecret) == 0 {
		return effectiveURL, env, cleanup, nil
	}

	authType := strings.ToLower(strings.TrimSpace(string(gitAuthSecret["auth_type"])))
	switch authType {
	case "token", "oauth":
		parsed, parseErr := url.Parse(effectiveURL)
		if parseErr != nil {
			return "", nil, cleanup, parseErr
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return effectiveURL, env, cleanup, nil
		}
		username := strings.TrimSpace(string(gitAuthSecret["username"]))
		if username == "" {
			username = "oauth2"
		}
		token := strings.TrimSpace(string(gitAuthSecret["token"]))
		if token == "" {
			return effectiveURL, env, cleanup, nil
		}
		parsed.User = url.UserPassword(username, token)
		effectiveURL = parsed.String()
		return effectiveURL, env, cleanup, nil
	case "basic":
		parsed, parseErr := url.Parse(effectiveURL)
		if parseErr != nil {
			return "", nil, cleanup, parseErr
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return effectiveURL, env, cleanup, nil
		}
		username := strings.TrimSpace(string(gitAuthSecret["username"]))
		password := strings.TrimSpace(string(gitAuthSecret["password"]))
		if username == "" || password == "" {
			return effectiveURL, env, cleanup, nil
		}
		parsed.User = url.UserPassword(username, password)
		effectiveURL = parsed.String()
		return effectiveURL, env, cleanup, nil
	case "ssh":
		privateKey := strings.TrimSpace(string(gitAuthSecret["ssh-privatekey"]))
		if privateKey == "" {
			return effectiveURL, env, cleanup, nil
		}
		keyFile, createErr := os.CreateTemp("", "if-git-ssh-key-*")
		if createErr != nil {
			return "", nil, cleanup, createErr
		}
		if _, writeErr := keyFile.Write([]byte(privateKey)); writeErr != nil {
			keyFile.Close()
			return "", nil, cleanup, writeErr
		}
		if closeErr := keyFile.Close(); closeErr != nil {
			return "", nil, cleanup, closeErr
		}
		if chmodErr := os.Chmod(keyFile.Name(), 0o600); chmodErr != nil {
			return "", nil, cleanup, chmodErr
		}
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Name()))
		cleanup = func() {
			_ = os.Remove(keyFile.Name())
		}
		return effectiveURL, env, cleanup, nil
	default:
		return effectiveURL, env, cleanup, nil
	}
}

func runGitCommand(ctx context.Context, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runGitCommandOutput(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func normalizeRepoPath(path string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	trimmed = strings.TrimPrefix(trimmed, "./")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "."
	}
	return trimmed
}
