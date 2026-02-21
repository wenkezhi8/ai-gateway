#!/bin/bash

# AI Gateway - Development Start Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 Starting AI Gateway..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check if Node.js is installed (for frontend)
if ! command -v node &> /dev/null; then
    echo "⚠️  Node.js is not installed. Frontend will not be available."
fi

# Create necessary directories
mkdir -p "$PROJECT_ROOT/data"

# Check for config file
if [ ! -f "$PROJECT_ROOT/configs/config.yaml" ]; then
    echo "📝 Creating default configuration..."
    cp "$PROJECT_ROOT/configs/config.example.yaml" "$PROJECT_ROOT/configs/config.yaml"
fi

# Start backend
echo "🔧 Starting backend server..."
cd "$PROJECT_ROOT/gateway"
go run ./cmd/server &
BACKEND_PID=$!

# Wait for backend to start
sleep 2

# Check if backend is running
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo "❌ Backend failed to start"
    exit 1
fi

echo "✅ Backend started (PID: $BACKEND_PID)"

# Start frontend if Node.js is available
if command -v node &> /dev/null; then
    echo "🎨 Starting frontend development server..."
    cd "$PROJECT_ROOT/console"

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing frontend dependencies..."
        npm install
    fi

    npm run dev &
    FRONTEND_PID=$!
    echo "✅ Frontend started (PID: $FRONTEND_PID)"
fi

echo ""
echo "🎉 AI Gateway is running!"
echo "   Backend:  http://localhost:8080"
echo "   Frontend: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop all services"

# Trap exit signals
trap "echo 'Stopping services...'; kill $BACKEND_PID 2>/dev/null; kill $FRONTEND_PID 2>/dev/null; exit 0" SIGINT SIGTERM

# Wait for processes
wait
