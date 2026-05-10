package agent

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type CaptureConfig struct {
	Iface   string
	Filter  string
	Duration string
}

func StartCapture(cfg CaptureConfig) (io.ReadCloser, error) {
	args := []string{"-i", cfg.Iface, "-w", "-"}
	if cfg.Filter != "" {
		args = append(args, strings.Fields(cfg.Filter)...)
	}

	cmd := exec.Command("tcpdump", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("tcpdump stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("tcpdump start: %w", err)
	}

	return &captureReadCloser{
		Reader: stdout,
		cmd:    cmd,
	}, nil
}

type captureReadCloser struct {
	io.Reader
	cmd *exec.Cmd
}

func (c *captureReadCloser) Close() error {
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
}
