package cli

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

func Deploy(client *ssh.Client, binaryPath string) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read binary: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, "C0755 %d door-agent\n", len(data))
		stdin.Write(data)
		fmt.Fprint(stdin, "\x00")
	}()

	if err := session.Run("scp -t /tmp/door-agent"); err != nil {
		return fmt.Errorf("scp: %w", err)
	}

	return nil
}
