package common

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/jeshuam/jbuild/args"
	"github.com/op/go-logging"
)

var (
	FileExists = func(filepath string) bool {
		_, err := os.Stat(filepath)
		if err == nil {
			return true
		}

		return false
	}

	IsDir = func(filepath string) bool {
		stat, _ := os.Stat(filepath)
		return stat != nil && stat.IsDir()
	}
)

type CmdSpec struct {
	Cmd      *exec.Cmd
	Lock     *sync.Mutex
	Result   chan error
	Complete func(string, bool, time.Duration)
}

func RunCommand(args *args.Args, cmd *exec.Cmd, result chan error, complete func(string, bool, time.Duration)) {
	log := logging.MustGetLogger("jbuild")

	// Print the command.
	if args.DryRun {
		log.Infof("DRY_RUN: %s", cmd.Args)
		if complete != nil {
			complete("", true, 0)
		}
		result <- nil
		return
	} else {
		if args.ShowCommands {
			log.Debug(cmd.Args)
		}

		if args.ShowCommandEnv {
			log.Debug(cmd.Env)
		}
	}

	// Save the command output.
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command.
	startTime := time.Now()
	err := cmd.Run()
	elaspedTime := time.Since(startTime)
	if err != nil {
		if complete != nil {
			complete(out.String(), false, elaspedTime)
		}
		if out.String() != "" {
			result <- errors.New(out.String())
		} else {
			result <- err
		}

		return
	}

	if complete != nil {
		complete(out.String(), true, elaspedTime)
	}
	result <- nil
}
