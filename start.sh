#!/bin/bash

# Packet Repackage System - Startup Script

set -e

echo "Starting Packet Repackage System..."

# Create data directory if it doesn't exist
mkdir -p data

# Create log directory if it doesn't exist
mkdir -p log
touch log/backend.log || echo "Warning: Cannot write to log/backend.log"

# Check if running as root
if [ "$EUID" -ne 0 ] && [ "$1" != "--no-queue" ]; then 
    echo "Warning: Not running as root. NFQueue will not be available."
    echo "Run with: sudo ./start.sh"
    echo "Or run in API-only mode: ./start.sh --no-queue"
    exit 1
fi

# Parse arguments
NO_QUEUE=false
START_FRONTEND=true

for arg in "$@"; do
    case $arg in
        --no-queue)
        NO_QUEUE=true
        shift
        ;;
        --no-frontend)
        START_FRONTEND=false
        shift
        ;;
    esac
done


# Check and enable br_netfilter module
if [ "$EUID" -eq 0 ]; then
    echo "Loading br_netfilter module..."
    modprobe br_netfilter || echo "Warning: Failed to load br_netfilter module"
    
    if [ -f "/proc/sys/net/bridge/bridge-nf-call-iptables" ]; then
        if [ "$(sysctl -n net.bridge.bridge-nf-call-iptables)" -ne 1 ]; then
            echo "Enabling bridge-nf-call-iptables..."
            sysctl -w net.bridge.bridge-nf-call-iptables=1
        fi
    else
        echo "Warning: /proc/sys/net/bridge/bridge-nf-call-iptables not found. Bridge traffic may not be intercepted."
    fi
fi

# Check and enable IP IP forwarding if running as root
if [ "$EUID" -eq 0 ]; then
    if [ "$(sysctl -n net.ipv4.ip_forward)" -ne 1 ]; then
        echo "Enabling IP forwarding..."
        sysctl -w net.ipv4.ip_forward=1
    fi
fi

# Cleanup existing processes
echo "Checking for existing services..."
# Kill backend on port 8080
if lsof -t -i:8080 >/dev/null 2>&1; then
    echo "Stopping existing backend on port 8080..."
    lsof -t -i:8080 | xargs -r kill -9
fi

# Kill explicitly by name just in case
pkill -f "packet-repackage" 2>/dev/null || true

# Kill frontend on port 3000 if restarting frontend
if [ "$START_FRONTEND" = true ]; then
    if lsof -t -i:3000 >/dev/null 2>&1; then
        echo "Stopping existing frontend on port 3000..."
        lsof -t -i:3000 | xargs -r kill -9
    fi
fi
echo "Building backend..."
cd server
rm -f packet-repackage
go build -o packet-repackage main.go
cd ..

# Start backend
echo "Starting backend server..."
if [ "$NO_QUEUE" = true ]; then
    ./server/packet-repackage -db ./data/packet.db -port 8080 -no-queue -log-path ./log/backend.log -log-level debug &
else
    # Listen on queues 0-3 by default to match possible rules
    ./server/packet-repackage -db ./data/packet.db -port 8080 -queues 0-3 -log-path ./log/backend.log -log-level debug &
fi

BACKEND_PID=$!
echo "Backend started with PID: $BACKEND_PID"

# Wait a bit for backend to start
sleep 2

# Start frontend if requested
if [ "$START_FRONTEND" = true ] && [ -d "./web" ]; then
    echo "Starting frontend development server..."
    cd web
    npm run dev &
    FRONTEND_PID=$!
    echo "Frontend started with PID: $FRONTEND_PID"
    cd ..
fi

echo ""
echo "Packet Repackage System is running!"
echo "Backend API: http://localhost:8080"
if [ "$START_FRONTEND" = true ]; then
    echo "Frontend UI: http://localhost:3000"
fi
echo ""
echo "Press Ctrl+C to stop..."

# Trap Ctrl+C and cleanup
trap "echo 'Stopping services...'; kill $BACKEND_PID 2>/dev/null; [ -n \"$FRONTEND_PID\" ] && kill $FRONTEND_PID 2>/dev/null; exit 0" INT

# Wait for processes
wait
