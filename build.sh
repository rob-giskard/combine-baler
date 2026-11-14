#!/bin/sh
# Cross-compiles build/CombineBaler.exe from Linux or macOS. Needs Go 1.24+.
set -e
cd "$(dirname "$0")"
go vet .
GOOS=windows GOARCH=amd64 go vet .
[ "$1" = "notest" ] || go test -count=1 .
mkdir -p build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w" -o build/CombineBaler.exe .
echo "Built build/CombineBaler.exe"
