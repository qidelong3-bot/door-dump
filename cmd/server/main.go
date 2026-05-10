package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/qidelong3-bot/door-dump/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data", "./data/captures", "data directory for pcap storage")
	flag.Parse()

	storage := server.NewStorage(*dataDir)
	handler := server.NewHandler(storage)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/upload", handler.Upload)
	mux.HandleFunc("/api/v1/captures", handler.List)
	mux.HandleFunc("/api/v1/captures/get", handler.Get)
	mux.HandleFunc("/api/v1/captures/delete", handler.Delete)
	mux.HandleFunc("/api/v1/health", handler.Health)

	log.Printf("server listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
