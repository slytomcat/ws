#!/bin/bash
CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w -X main.version=$(git branch --show-current)-$(git rev-parse --short HEAD)" .
upx -qqq --best ws

cd echo-server/
CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w -X main.version=$(git branch --show-current)-$(git rev-parse --short HEAD)" .
upx -qqq --best echo-server


