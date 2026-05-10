.PHONY: build build-agent build-server build-cli deploy test clean

# Build all
build: build-agent build-server build-cli

# Build agent for OpenWrt (mipsle)
build-agent:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-s -w" -trimpath -o door-agent ./cmd/agent

# Build server for current platform
build-server:
	go build -o door-server ./cmd/server

# Build CLI for current platform
build-cli:
	go build -o doorctl ./cmd/cli

# Deploy agent to router
deploy: build-agent
	./doorctl deploy --host $(HOST) --password $(PASSWORD) --binary ./door-agent

# Run server locally
run-server: build-server
	./door-server -addr :8080 -data ./data/captures

# Test capture
test-capture: build-agent
	./doorctl capture --host $(HOST) --password $(PASSWORD) --iface br-lan --duration 10s --server http://localhost:8080

# Clean build artifacts
clean:
	rm -f door-agent door-server doorctl
