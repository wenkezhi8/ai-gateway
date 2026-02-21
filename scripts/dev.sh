#!/bin/bash

# Development setup script

set -e

echo "Setting up AI Gateway development environment..."

# Create necessary directories
echo "Creating directories..."
mkdir -p data
mkdir -p logs

# Copy example config if config doesn't exist
if [ ! -f "configs/config.json" ]; then
    echo "Copying example configuration..."
    cp configs/config.example.json configs/config.json
    echo "Please edit configs/config.json with your API keys"
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod download

# Install development tools
echo "Installing development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || true

echo ""
echo "Setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit configs/config.json with your API keys"
echo "2. Run 'make run' to start the server"
echo ""
