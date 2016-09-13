package command

import (
	"os"

	"github.com/fatih/color"
	"github.com/jeshuam/jbuild/common"
)

func Clean(workspaceDir string) {
	color.New(color.FgHiBlue, color.Bold).Printf("$ rm -rf %s\n", common.OutputDirectory)
	err := os.RemoveAll(common.OutputDirectory)
	if err != nil {
		log.Fatalf("Failed to clean: %s", err)
	}
}
