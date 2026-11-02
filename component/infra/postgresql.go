package infra

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"orb/component"
	"orb/executor"
)

// PostgreSQL owns PostgreSQL-specific input gathering, preparation, and
// lifecycle dispatch. It is registered independently from other infra
// components so their behavior can differ.
type PostgreSQL struct{}

func (PostgreSQL) Name() string { return "postgresql" }

func (PostgreSQL) GatherInputs(ctx context.Context, deployment *component.DeploymentConfig, prompter component.Prompter) error {
	hostsText, err := prompter.Text(ctx, "Target host(s), comma separated; the first host is primary: ")
	if err != nil {
		return err
	}
	for _, host := range strings.Split(hostsText, ",") {
		host = strings.TrimSpace(host)
		if host != "" {
			deployment.Hosts = append(deployment.Hosts, host)
		}
	}
	if len(deployment.Hosts) == 0 {
		return fmt.Errorf("at least one target host is required")
	}

	if deployment.SSHUser, err = prompter.Text(ctx, "SSH username: "); err != nil {
		return err
	}
	if deployment.SSHPassword, err = prompter.Secret(ctx, "SSH password: "); err != nil {
		return err
	}
	if deployment.PostgreSQLUsername, err = prompter.Text(ctx, "PostgreSQL username: "); err != nil {
		return err
	}
	if deployment.PostgreSQLPassword, err = prompter.Secret(ctx, "PostgreSQL password: "); err != nil {
		return err
	}
	deployment.RepmgrPassword, err = randomPassword()
	return err
}

func (PostgreSQL) Setup(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	if err := deployment.Validate(); err != nil {
		return err
	}

	partners := make([]string, len(deployment.Hosts))
	for i, host := range deployment.Hosts {
		partners[i] = host + ":5432"
	}

	sshCredentials := executor.Credentials{Username: deployment.SSHUser, Password: deployment.SSHPassword}
	for index, host := range deployment.Hosts {
		if err := preparePostgreSQLDataDirectory(ctx, runner, host, sshCredentials); err != nil {
			return fmt.Errorf("create PostgreSQL data directory on %s: %w", host, err)
		}
		replacements := map[string]string{
			"PostgreSQLUsername": deployment.PostgreSQLUsername,
			"PostgreSQLPassword": deployment.PostgreSQLPassword,
			"RepmgrPassword":     deployment.RepmgrPassword,
			"PrimaryHost":        deployment.Hosts[0],
			"PartnerNodes":       strings.Join(partners, ","),
			"NodeName":           fmt.Sprintf("orb-postgresql-%d", index+1),
			"NetworkHost":        host,
		}
		composeFile, err := component.CreateCompose("infra.yaml", replacements)
		if err != nil {
			return fmt.Errorf("render Compose file for %s: %w", host, err)
		}
		if err := runner.WriteFile(ctx, host, sshCredentials, "~/.orb/infra/infra.yaml", composeFile); err != nil {
			return fmt.Errorf("write Compose file on %s: %w", host, err)
		}
	}
	return nil
}

func preparePostgreSQLDataDirectory(ctx context.Context, runner executor.Runner, host string, credentials executor.Credentials) error {
	rootAccess := (!executor.IsLocal(host) && credentials.Username == "root") || (executor.IsLocal(host) && os.Geteuid() == 0)
	if rootAccess {
		if err := runner.Run(ctx, host, credentials, "mkdir", "-p", "/var/pgsql"); err != nil {
			return err
		}
		return runner.Run(ctx, host, credentials, "chown", "1001:1001", "/var/pgsql")
	}
	// The image runs as UID 1001 and needs write access to the bind-mounted
	// directory. sudo reads the SSH user's password from stdin, never argv.
	command := "mkdir -p /var/pgsql && chown 1001:1001 /var/pgsql"
	return runner.RunWithInput(ctx, host, credentials, []byte(credentials.Password+"\n"), "sudo", "-S", "-p", "", "sh", "-c", command)
}

func (PostgreSQL) Install(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	return runComposeOnHosts(ctx, deployment, runner, "up", "-d")
}

func (PostgreSQL) Upgrade(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	if err := runComposeOnHosts(ctx, deployment, runner, "pull"); err != nil {
		return err
	}
	return runComposeOnHosts(ctx, deployment, runner, "up", "-d")
}

func (PostgreSQL) Status(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	return runComposeOnHosts(ctx, deployment, runner, "ps")
}

func (PostgreSQL) Start(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	return runComposeOnHosts(ctx, deployment, runner, "start")
}

func (PostgreSQL) Stop(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	return runComposeOnHosts(ctx, deployment, runner, "stop")
}

func (PostgreSQL) Restart(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner) error {
	return runComposeOnHosts(ctx, deployment, runner, "restart")
}

func runComposeOnHosts(ctx context.Context, deployment component.DeploymentConfig, runner executor.Runner, args ...string) error {
	if err := deployment.Validate(); err != nil {
		return fmt.Errorf("load setup configuration first: %w", err)
	}
	sshCredentials := executor.Credentials{Username: deployment.SSHUser, Password: deployment.SSHPassword}
	for _, host := range deployment.Hosts {
		command := "cd \"$HOME/.orb/infra\" && docker compose --project-name orb-infra -f infra.yaml"
		for _, arg := range args {
			command += " " + shellQuote(arg)
		}
		if err := runner.RunShell(ctx, host, sshCredentials, command); err != nil {
			return fmt.Errorf("run PostgreSQL Compose operation on %s: %w", host, err)
		}
	}
	return nil
}

func randomPassword() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate repmgr password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
