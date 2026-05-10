package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/qidelong3-bot/door-dump/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		deployCmd()
	case "capture":
		captureCmd()
	case "list":
		listCmd()
	case "download":
		downloadCmd()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: doorctl <command> [options]

Commands:
  deploy    Deploy agent binary to router
  capture   Start packet capture on router
  list      List captures on server
  download  Download capture from server`)
}

func deployCmd() {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	host := fs.String("host", "", "router host")
	port := fs.String("port", "22", "SSH port")
	user := fs.String("user", "root", "SSH user")
	password := fs.String("password", "", "SSH password")
	binary := fs.String("binary", "./door-agent", "agent binary path")
	fs.Parse(os.Args[2:])

	if *host == "" {
		log.Fatal("--host required")
	}

	client, err := cli.Connect(cli.SSHConfig{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
	})
	if err != nil {
		log.Fatalf("ssh connect: %v", err)
	}
	defer client.Close()

	if err := cli.Deploy(client, *binary); err != nil {
		log.Fatalf("deploy: %v", err)
	}
	fmt.Println("agent deployed successfully")
}

func captureCmd() {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	host := fs.String("host", "", "router host")
	port := fs.String("port", "22", "SSH port")
	user := fs.String("user", "root", "SSH user")
	password := fs.String("password", "", "SSH password")
	iface := fs.String("iface", "br-lan", "network interface")
	filter := fs.String("filter", "", "BPF filter")
	duration := fs.String("duration", "60s", "capture duration")
	server := fs.String("server", "", "server URL")
	device := fs.String("device", "", "device identifier")
	fs.Parse(os.Args[2:])

	if *host == "" || *server == "" {
		log.Fatal("--host and --server required")
	}

	client, err := cli.Connect(cli.SSHConfig{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
	})
	if err != nil {
		log.Fatalf("ssh connect: %v", err)
	}
	defer client.Close()

	if err := cli.RunCapture(client, cli.CaptureParams{
		Iface:    *iface,
		Filter:   *filter,
		Duration: *duration,
		Server:   *server,
		Device:   *device,
	}); err != nil {
		log.Fatalf("capture: %v", err)
	}
	fmt.Println("capture completed")
}

func listCmd() {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	server := fs.String("server", "", "server URL")
	fs.Parse(os.Args[2:])

	if *server == "" {
		log.Fatal("--server required")
	}

	resp, err := http.Get(*server + "/api/v1/captures")
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()

	var captures []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&captures); err != nil {
		log.Fatalf("decode: %v", err)
	}

	for _, c := range captures {
		fmt.Printf("%-30s  %-10s  %-10s  %v\n",
			c["id"],
			c["interface"],
			c["device"],
			c["start_time"],
		)
	}
}

func downloadCmd() {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	server := fs.String("server", "", "server URL")
	output := fs.String("o", "", "output file")
	fs.Parse(os.Args[2:])

	if *server == "" || fs.NArg() < 1 {
		log.Fatal("--server and capture ID required")
	}

	id := fs.Arg(0)
	url := fmt.Sprintf("%s/api/v1/captures/get?id=%s", *server, id)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("download failed: %s", resp.Status)
	}

	if *output == "" {
		*output = id + ".pcap"
	}

	data := make([]byte, 0)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}
	fmt.Printf("downloaded %d bytes to %s\n", len(data), *output)
}
