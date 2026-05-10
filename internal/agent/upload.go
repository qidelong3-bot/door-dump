package agent

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type UploadConfig struct {
	ServerURL string
	Device    string
	Iface     string
	Filter    string
}

func StreamUpload(cfg UploadConfig, body io.Reader) error {
	u, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}
	u.Path = "/api/v1/upload"

	q := u.Query()
	if cfg.Device != "" {
		q.Set("device", cfg.Device)
	}
	if cfg.Iface != "" {
		q.Set("interface", cfg.Iface)
	}
	if cfg.Filter != "" {
		q.Set("filter", cfg.Filter)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Transfer-Encoding", "chunked")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}
	return nil
}
