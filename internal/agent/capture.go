package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CaptureConfig struct {
	Iface   string
	Filter  string
	Duration string
}

type IfaceInfo struct {
	Name string
	RxBytes string
	TxBytes string
}

func ListInterfaces() ([]IfaceInfo, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("open /proc/net/dev: %w", err)
	}
	defer file.Close()

	var ifaces []IfaceInfo
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(fields) < 9 {
			continue
		}
		ifaces = append(ifaces, IfaceInfo{
			Name:    name,
			RxBytes: fields[0],
			TxBytes: fields[8],
		})
	}
	return ifaces, nil
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
