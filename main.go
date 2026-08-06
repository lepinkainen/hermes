package main

import "github.com/lepinkainen/hermes/cmd"

// Build information, injected at link time via -ldflags "-X main.Version=...".
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var execute = cmd.Execute

func main() {
	cmd.SetVersionInfo(Version, GitCommit, BuildDate)
	execute()
}
