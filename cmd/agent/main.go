package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qidelong3-bot/door-dump/internal/agent"
)

func main() {
	iface := flag.String("i", "br-lan", "network interface")
	filter := flag.String("f", "", "BPF filter expression")
	duration := flag.String("d", "60s", "capture duration (e.g. 30s, 5m)")
	server := flag.String("s", "", "server URL for upload (e.g. https://capture.example.com)")
	device := flag.String("device", "", "device identifier")
	listIfaces := flag.Bool("list", false, "list available network interfaces and exit")
	flag.Parse()

	if *listIfaces {
		ifaces, err := agent.ListInterfaces()
		if err != nil {
			log.Fatalf("list interfaces: %v", err)
		}
		fmt.Printf("%-16s %15s %15s\n", "INTERFACE", "RX_BYTES", "TX_BYTES")
		for _, iface := range ifaces {
			fmt.Printf("%-16s %15s %15s\n", iface.Name, iface.RxBytes, iface.TxBytes)
		}
		return
	}

	if *server == "" {
		log.Fatal("server URL is required (-s)")
	}

	dur, err := time.ParseDuration(*duration)
	if err != nil {
		log.Fatalf("invalid duration %q: %v", *duration, err)
	}

	log.Printf("starting capture on %s filter=%q duration=%s server=%s", *iface, *filter, *duration, *server)

	capture, err := agent.StartCapture(agent.CaptureConfig{
		Iface:   *iface,
		Filter:  *filter,
		Duration: *duration,
	})
	if err != nil {
		log.Fatalf("start capture: %v", err)
	}
	defer capture.Close()

	done := make(chan error, 1)
	go func() {
		done <- agent.StreamUpload(agent.UploadConfig{
			ServerURL: *server,
			Device:    *device,
			Iface:     *iface,
			Filter:    *filter,
		}, capture)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-timer.C:
		log.Println("duration reached, stopping capture")
		capture.Close()
	case s := <-sig:
		log.Printf("received signal %v, stopping capture", s)
		capture.Close()
	case err := <-done:
		if err != nil {
			log.Fatalf("upload failed: %v", err)
		}
		fmt.Println("upload completed")
		return
	}

	if err := <-done; err != nil {
		log.Fatalf("upload failed: %v", err)
	}
	fmt.Println("capture and upload completed")
}
