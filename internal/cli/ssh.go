package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

func Connect(cfg SSHConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", "id_ed25519")
	if key, err := os.ReadFile(keyPath); err == nil {
		if signer, err := ssh.ParsePrivateKey(key); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	var hostKeyCallback ssh.HostKeyCallback
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	if kh, err := knownhosts.New(knownHostsPath); err == nil {
		hostKeyCallback = kh
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	})
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	return client, nil
}

func Exec(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("exec %q: %w", cmd, err)
	}
	return string(output), nil
}
