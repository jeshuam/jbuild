package command

import (
	"os"

	"github.com/fatih/color"
	"github.com/jeshuam/jbuild/args"
)

func Clean(workspaceDir string) {
	color.New(color.FgHiBlue, color.Bold).Printf("$ rm -rf %s\n", args.OutputDir)
	err := os.RemoveAll(args.OutputDir)
	if err != nil {
		log.Fatalf("Failed to clean: %s", err)
	}
}
