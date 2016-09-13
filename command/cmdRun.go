package command

import (
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/jeshuam/jbuild/config"
)

func Run(target *config.Target, args []string) {
	cmd := exec.Command(target.Output[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	color.New(color.FgHiBlue, color.Bold).Printf("\n$ %s\n", strings.Join(cmd.Args, " "))
	cmd.Run()
}
