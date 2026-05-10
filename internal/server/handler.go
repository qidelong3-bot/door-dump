package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	storage *Storage
}

func NewHandler(storage *Storage) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := fmt.Sprintf("%s_%s_%d",
		q.Get("device"),
		q.Get("interface"),
		time.Now().UnixNano(),
	)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("read body: %v", err)
		http.Error(w, "read body failed", http.StatusInternalServerError)
		return
	}

	meta := CaptureMeta{
		Device:    q.Get("device"),
		Interface: q.Get("interface"),
		Filter:    q.Get("filter"),
		StartTime: time.Now().Format(time.RFC3339),
		SourceIP:  r.RemoteAddr,
	}

	if err := h.storage.Save(id, meta, data); err != nil {
		log.Printf("save: %v", err)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	log.Printf("capture %s saved: %d bytes", id, len(data))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id":"%s","size":%d}`, id, len(data))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	captures, err := h.storage.List()
	if err != nil {
		log.Printf("list: %v", err)
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(captures)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	path, err := h.storage.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pcap"`, id))
	http.ServeFile(w, r, path)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	if err := h.storage.Delete(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"deleted":"%s"}`, id)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}
