package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CaptureMeta struct {
	ID        string `json:"id"`
	Device    string `json:"device"`
	Interface string `json:"interface"`
	Filter    string `json:"filter"`
	Size      int64  `json:"size"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	SourceIP  string `json:"source_ip"`
}

type Storage struct {
	baseDir string
}

func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

func (s *Storage) Save(id string, meta CaptureMeta, data []byte) error {
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(s.baseDir, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	pcapPath := filepath.Join(dir, id+".pcap")
	if err := os.WriteFile(pcapPath, data, 0644); err != nil {
		return fmt.Errorf("write pcap: %w", err)
	}

	meta.Size = int64(len(data))
	meta.ID = id
	meta.EndTime = time.Now().Format(time.RFC3339)

	metaPath := filepath.Join(dir, id+".meta.json")
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	return nil
}

func (s *Storage) List() ([]CaptureMeta, error) {
	var captures []CaptureMeta

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return captures, nil
		}
		return nil, fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dateDir := filepath.Join(s.baseDir, entry.Name())
		files, err := os.ReadDir(dateDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if filepath.Ext(f.Name()) != ".json" {
				continue
			}
			metaPath := filepath.Join(dateDir, f.Name())
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var meta CaptureMeta
			if err := json.Unmarshal(data, &meta); err != nil {
				continue
			}
			captures = append(captures, meta)
		}
	}
	return captures, nil
}

func (s *Storage) Get(id string) (string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return "", fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pcapPath := filepath.Join(s.baseDir, entry.Name(), id+".pcap")
		if _, err := os.Stat(pcapPath); err == nil {
			return pcapPath, nil
		}
	}
	return "", fmt.Errorf("capture %s not found", id)
}

func (s *Storage) Delete(id string) error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(s.baseDir, entry.Name())
		pcapPath := filepath.Join(dir, id+".pcap")
		metaPath := filepath.Join(dir, id+".meta.json")

		found := false
		if _, err := os.Stat(pcapPath); err == nil {
			os.Remove(pcapPath)
			found = true
		}
		if _, err := os.Stat(metaPath); err == nil {
			os.Remove(metaPath)
			found = true
		}
		if found {
			return nil
		}
	}
	return fmt.Errorf("capture %s not found", id)
}
