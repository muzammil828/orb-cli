// Package executor runs commands on the local Linux host or over SSH.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Credentials struct {
	Username string
	Password string
}

type Runner struct{}

func (Runner) Run(ctx context.Context, host string, credentials Credentials, command string, args ...string) error {
	return (Runner{}).RunWithInput(ctx, host, credentials, nil, command, args...)
}

func (Runner) RunWithInput(ctx context.Context, host string, credentials Credentials, input []byte, command string, args ...string) error {
	if IsLocal(host) {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if input != nil {
			cmd.Stdin = bytes.NewReader(input)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run local command %q: %w", command, err)
		}
		return nil
	}
	var stdin io.Reader
	if input != nil {
		stdin = bytes.NewReader(input)
	}
	return runSSH(ctx, host, credentials, shellJoin(command, args...), stdin)
}

// RunShell is for fixed application commands that require shell features such
// as changing into the remote user's home directory before executing Compose.
func (Runner) RunShell(ctx context.Context, host string, credentials Credentials, command string) error {
	if IsLocal(host) {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run local shell command: %w", err)
		}
		return nil
	}
	return runSSH(ctx, host, credentials, command, nil)
}

func (Runner) WriteFile(ctx context.Context, host string, credentials Credentials, remotePath string, data []byte) error {
	if IsLocal(host) {
		localPath := remotePath
		if strings.HasPrefix(localPath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("find local home directory: %w", err)
			}
			localPath = filepath.Join(home, filepath.FromSlash(strings.TrimPrefix(localPath, "~/")))
		}
		if err := os.MkdirAll(filepath.Dir(localPath), 0o700); err != nil {
			return fmt.Errorf("create local deployment directory: %w", err)
		}
		if err := os.Chmod(filepath.Dir(localPath), 0o700); err != nil {
			return fmt.Errorf("restrict local deployment directory permissions: %w", err)
		}
		if err := os.WriteFile(localPath, data, 0o600); err != nil {
			return fmt.Errorf("write local deployment file: %w", err)
		}
		if err := os.Chmod(localPath, 0o600); err != nil {
			return fmt.Errorf("restrict local deployment file permissions: %w", err)
		}
		return nil
	}
	// remotePath is supplied by the application, not by a user-entered value.
	filename := path.Base(filepath.ToSlash(remotePath))
	command := "mkdir -p \"$HOME/.orb/infra\" && chmod 700 \"$HOME/.orb/infra\" && umask 077 && cat > \"$HOME/.orb/infra/" + filename + "\" && chmod 600 \"$HOME/.orb/infra/" + filename + "\""
	return runSSH(ctx, host, credentials, command, bytes.NewReader(data))
}

func IsLocal(host string) bool {
	host = strings.TrimSpace(host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if strings.EqualFold(host, "local") || strings.EqualFold(host, "localhost") {
		return true
	}
	if parsed := net.ParseIP(host); parsed != nil && parsed.IsLoopback() {
		return true
	}
	if localHostname, err := os.Hostname(); err == nil && strings.EqualFold(host, localHostname) {
		return true
	}
	return false
}

func runSSH(ctx context.Context, address string, credentials Credentials, command string, stdin io.Reader) error {
	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	if home, err := os.UserHomeDir(); err == nil {
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
	}
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return fmt.Errorf("load SSH known_hosts file %q (connect to each target with OpenSSH first to verify and record its host key): %w", knownHostsPath, err)
	}

	host, port := address, "22"
	if parsedHost, parsedPort, err := net.SplitHostPort(address); err == nil {
		host, port = parsedHost, parsedPort
	} else if strings.Count(address, ":") == 1 {
		parts := strings.SplitN(address, ":", 2)
		host, port = parts[0], parts[1]
	}
	endpoint := net.JoinHostPort(host, port)
	clientConfig := &ssh.ClientConfig{
		User:            credentials.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(credentials.Password)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	dialer := net.Dialer{Timeout: 15 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return fmt.Errorf("connect to SSH host %q: %w", address, err)
	}
	sshConnection, channels, requests, err := ssh.NewClientConn(connection, endpoint, clientConfig)
	if err != nil {
		connection.Close()
		return fmt.Errorf("establish SSH session with %q: %w", address, err)
	}
	client := ssh.NewClient(sshConnection, channels, requests)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH command session on %q: %w", address, err)
	}
	defer session.Close()
	session.Stdout, session.Stderr, session.Stdin = os.Stdout, os.Stderr, stdin

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("run command on SSH host %q: %w", address, err)
		}
		return nil
	case <-ctx.Done():
		client.Close()
		return ctx.Err()
	}
}

func shellJoin(command string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(command))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
