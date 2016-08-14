package common

import (
	"bytes"
	"errors"
	"os"
	"os/exec"

	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")
)

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true
	}

	return false
}

type CmdSpec struct {
	Cmd      *exec.Cmd
	Result   chan error
	Complete func(error)
}

func RunCommand(cmd *exec.Cmd, result chan error, complete func(error)) {
	// Print the command.
	if DryRun {
		log.Infof("DRY_RUN: %s", cmd.Args)
		result <- nil
	} else {
		log.Debug(cmd.Args)
	}

	// Save the command output.
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command.
	err := cmd.Run()
	if err != nil {
		if out.String() != "" {
			complete(errors.New(out.String()))
			result <- errors.New(out.String())

		} else {
			complete(err)
			result <- err
		}

		return
	}

	complete(nil)
	result <- nil
}
