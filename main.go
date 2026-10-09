package main

import (
	"context"
	"os"

	"github.com/alexjoedt/forge/internal/commands"
	"github.com/alexjoedt/forge/internal/log"
)

// Set via ldflags at build time.
var (
	// version is set via ldflags at build time.
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	root := commands.Root(commands.BuildInfo{Version: version, Commit: commit, Date: date, BuiltBy: builtBy})
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.DefaultLogger.Errorf("command failed: %v", err)
		os.Exit(1)
	}
}
