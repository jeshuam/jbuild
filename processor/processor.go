package processor

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/processor/cc"
	"github.com/jeshuam/jbuild/processor/cs"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")
)

type ProcessingResult struct {
	Target *config.Target
	Err    error
}

type Processor interface {
	// Process the given target using this processor.
	Process(*config.Target, chan common.CmdSpec) error
}

func (e *ProcessingResult) Error() string {
	return fmt.Sprintf("Processing target %s failed: %s", e.Target, e.Err)
}

func makeProcessingResult(target *config.Target, err error) ProcessingResult {
	return ProcessingResult{target, err}
}

func Process(target *config.Target, ch chan ProcessingResult, taskQueue chan common.CmdSpec) error {
	// Switch on the processor type.
	var p Processor
	if strings.HasPrefix(target.Type, "c++/") {
		p = new(cc.CCProcessor)
		target.ProgressBar = progress.AddBar(len(target.Srcs())+1, target.String())
	} else if strings.HasPrefix(target.Type, "c#/") {
		p = new(cs.CSProcessor)
		target.ProgressBar = progress.AddBar(1, target.String())
	} else {
		return errors.New(fmt.Sprintf("Unknown target type '%s'", target.Type))
	}

	// Make the progress bar.

	// Process the target.
	go func() {
		// Make the output directory for this target.
		err := os.MkdirAll(target.Spec.OutputPath(), 0755)
		if err != nil {
			ch <- makeProcessingResult(target, err)
			return
		}

		err = p.Process(target, taskQueue)
		target.Processed = true
		ch <- makeProcessingResult(target, err)
	}()

	return nil
}
