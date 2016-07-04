package processor

import (
	"fmt"

	"github.com/jeshuam/jbuild/config"
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
	Process(*config.Target) error
}

func (e *ProcessingResult) Error() string {
	return fmt.Sprintf("Processing target %s failed: %s", e.Target, e.Err)
}

func makeProcessingResult(target *config.Target, err error) ProcessingResult {
	return ProcessingResult{target, err}
}

func Process(target *config.Target, ch chan ProcessingResult) {
	p := NullProcessor{}
	go func() {
		err := p.Process(target)
		ch <- makeProcessingResult(target, err)
	}()
}
