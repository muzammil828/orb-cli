package component

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DeploymentConfig contains the operator-provided settings collected by
// `orb setup`. Its file is protected with owner-only permissions where
// supported by the operating system.
type DeploymentConfig struct {
	Hosts              []string
	SSHUser            string
	SSHPassword        string
	PostgreSQLUsername string
	PostgreSQLPassword string
	RepmgrPassword     string
}

func DefaultConfigPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(root, "orb", "deployment.cfg"), nil
}

func (d DeploymentConfig) Validate() error {
	if len(d.Hosts) == 0 {
		return fmt.Errorf("at least one target host is required")
	}
	for i, host := range d.Hosts {
		if strings.TrimSpace(host) == "" {
			return fmt.Errorf("target host %d is empty", i+1)
		}
	}
	if strings.TrimSpace(d.SSHUser) == "" || d.SSHPassword == "" {
		return fmt.Errorf("SSH username and password are required")
	}
	if strings.TrimSpace(d.PostgreSQLUsername) == "" || d.PostgreSQLPassword == "" {
		return fmt.Errorf("PostgreSQL username and password are required")
	}
	if d.RepmgrPassword == "" {
		return fmt.Errorf("internal repmgr password is missing; run setup again")
	}
	return nil
}

func SaveConfig(path string, d DeploymentConfig) error {
	if err := d.Validate(); err != nil {
		return fmt.Errorf("validate deployment config: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("restrict config directory permissions: %w", err)
	}
	contents := "[deployment]\n" +
		"hosts = " + strconv.Quote(strings.Join(d.Hosts, ",")) + "\n\n" +
		"[ssh]\n" +
		"user = " + strconv.Quote(d.SSHUser) + "\n" +
		"password = " + strconv.Quote(d.SSHPassword) + "\n\n" +
		"[infra.postgresql]\n" +
		"username = " + strconv.Quote(d.PostgreSQLUsername) + "\n" +
		"password = " + strconv.Quote(d.PostgreSQLPassword) + "\n" +
		"repmgr_password = " + strconv.Quote(d.RepmgrPassword) + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write deployment config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict deployment config permissions: %w", err)
	}
	return nil
}

func LoadConfig(path string) (DeploymentConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return DeploymentConfig{}, fmt.Errorf("open deployment config: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	section := ""
	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section == "" {
			return DeploymentConfig{}, fmt.Errorf("invalid config syntax on line %d", lineNo)
		}
		decoded, err := strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			return DeploymentConfig{}, fmt.Errorf("invalid value on config line %d: %w", lineNo, err)
		}
		values[section+"."+strings.TrimSpace(key)] = decoded
	}
	if err := scanner.Err(); err != nil {
		return DeploymentConfig{}, fmt.Errorf("read deployment config: %w", err)
	}

	deployment := DeploymentConfig{
		Hosts:              splitHosts(values["deployment.hosts"]),
		SSHUser:            values["ssh.user"],
		SSHPassword:        values["ssh.password"],
		PostgreSQLUsername: values["infra.postgresql.username"],
		PostgreSQLPassword: values["infra.postgresql.password"],
		RepmgrPassword:     values["infra.postgresql.repmgr_password"],
	}
	if err := deployment.Validate(); err != nil {
		return DeploymentConfig{}, fmt.Errorf("invalid deployment config: %w", err)
	}
	return deployment, nil
}

func splitHosts(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		host := strings.TrimSpace(part)
		if host != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}
