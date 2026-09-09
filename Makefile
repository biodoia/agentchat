.PHONY: build test vet clean install lint

# Build all binaries
build:
	go build -o agentchat-relay ./cmd/relay/
	go build -o agentchat-tui ./cmd/tui/
	go build -o agentchat-mcp ./cmd/mcp/
	go build -o agentchat-tray ./cmd/tray/

# Run all tests
test:
	go test -v -count=1 ./...

# Run tests with coverage
cover:
	go test -cover ./...

# Run go vet
vet:
	go vet ./...

# Clean binaries
clean:
	rm -f agentchat-relay agentchat-tui agentchat-mcp agentchat-tray

# Install binaries to ~/.local/bin
install: build
	cp agentchat-relay ~/.local/bin/
	cp agentchat-tui ~/.local/bin/
	cp agentchat-mcp ~/.local/bin/
	cp agentchat-tray ~/.local/bin/

# Install systemd services
install-services:
	cp deploy/*.service ~/.config/systemd/user/
	systemctl --user daemon-reload

# Tidy go modules
tidy:
	go mod tidy

# Full check: vet + test
check: vet test

# Start relay for development
dev-relay:
	go run ./cmd/relay/ -data ./data

# Start TUI for development
dev-tui:
	go run ./cmd/tui/

# Start tray for development
dev-tray:
	go run ./cmd/tray/
