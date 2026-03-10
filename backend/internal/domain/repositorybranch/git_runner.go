package repositorybranch

import (
    "bufio"
    "context"
    "errors"
    "net/url"
    "os"
    "os/exec"
    "strings"

    "github.com/srikarm/image-factory/internal/domain/repositoryauth"
)

const (
    defaultTokenUsername = "oauth2"
)

type ExecGitRunner struct {}

func NewExecGitRunner() *ExecGitRunner {
    return &ExecGitRunner{}
}

func (r *ExecGitRunner) ListRemoteBranches(ctx context.Context, repoURL string, auth GitAuth) ([]string, error) {
    if strings.TrimSpace(repoURL) == "" {
        return nil, errors.New("repository URL is required")
    }

    effectiveURL := repoURL
    env := os.Environ()
    env = append(env, "GIT_TERMINAL_PROMPT=0")

    switch auth.AuthType {
    case repositoryauth.AuthTypeSSH:
        if auth.SSHKey == "" {
            return nil, errors.New("ssh key is required")
        }
        keyPath, err := writeTempKey(auth.SSHKey)
        if err != nil {
            return nil, err
        }
        defer os.Remove(keyPath)
        sshCommand := "ssh -i " + keyPath + " -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
        env = append(env, "GIT_SSH_COMMAND="+sshCommand)
    case repositoryauth.AuthTypeToken, repositoryauth.AuthTypeBasic:
        urlWithCreds, err := applyHTTPAuth(repoURL, auth)
        if err != nil {
            return nil, err
        }
        effectiveURL = urlWithCreds
    }

    cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", effectiveURL)
    cmd.Env = env

    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    branches := []string{}
    scanner := bufio.NewScanner(strings.NewReader(string(output)))
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.Split(line, "\t")
        if len(parts) != 2 {
            continue
        }
        ref := parts[1]
        if strings.HasPrefix(ref, "refs/heads/") {
            branches = append(branches, strings.TrimPrefix(ref, "refs/heads/"))
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return branches, nil
}

func writeTempKey(key string) (string, error) {
    file, err := os.CreateTemp("", "git-key-*")
    if err != nil {
        return "", err
    }
    path := file.Name()
    if _, err := file.Write([]byte(key)); err != nil {
        file.Close()
        return "", err
    }
    if err := file.Close(); err != nil {
        return "", err
    }
    if err := os.Chmod(path, 0600); err != nil {
        return "", err
    }
    return path, nil
}

func applyHTTPAuth(repoURL string, auth GitAuth) (string, error) {
    parsed, err := url.Parse(repoURL)
    if err != nil {
        return "", err
    }

    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return "", errors.New("repository URL must be http(s) for token/basic auth")
    }

    username := auth.Username
    password := auth.Password

    if auth.AuthType == repositoryauth.AuthTypeToken {
        if auth.Token == "" {
            return "", errors.New("token is required")
        }
        if username == "" {
            username = defaultTokenUsername
        }
        password = auth.Token
    }

    if username == "" || password == "" {
        return "", errors.New("username and password are required")
    }

    parsed.User = url.UserPassword(username, password)
    return parsed.String(), nil
}
