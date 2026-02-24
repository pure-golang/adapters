#!/bin/bash

# FCM Test Server - Simple HTTP server for testing FCM web push

cd "$(dirname "$0")"

echo "=== FCM Test Server ==="
echo ""
echo "Starting server on http://localhost:8080"
echo "Press Ctrl+C to stop"
echo ""

if command -v python3 &> /dev/null; then
    python3 -m http.server 8080
elif command -v python &> /dev/null; then
    python -m SimpleHTTPServer 8080
elif command -v php &> /dev/null; then
    php -S localhost:8080
else
    echo "Error: No HTTP server found"
    echo "Install one of: python3, python, or php"
    exit 1
fi
