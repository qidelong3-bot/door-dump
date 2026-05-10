package cli

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type CaptureParams struct {
	Iface    string
	Filter   string
	Duration string
	Server   string
	Device   string
}

func RunCapture(client *ssh.Client, params CaptureParams) error {
	cmd := fmt.Sprintf("/tmp/door-agent -i %s -f %q -d %s -s %s",
		params.Iface,
		params.Filter,
		params.Duration,
		params.Server,
	)
	if params.Device != "" {
		cmd += fmt.Sprintf(" -device %s", params.Device)
	}

	dur, err := time.ParseDuration(params.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				fmt.Print(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	timer := time.NewTimer(dur + 10*time.Second)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("agent finished with error: %w", err)
		}
	case <-timer.C:
		session.Close()
		return fmt.Errorf("capture timed out")
	}

	return nil
}
