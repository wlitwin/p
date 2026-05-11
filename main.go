package main

import "github.com/walter/p/cmd"

// These are set via ldflags at build time:
//
//	go build -ldflags "-X github.com/walter/p/cmd.Version=v1.0.0
//	  -X github.com/walter/p/cmd.GitCommit=$(git rev-parse --short HEAD)
//	  -X github.com/walter/p/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

func main() {
	cmd.Execute()
}
