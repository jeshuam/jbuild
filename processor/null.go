package processor

import (
	"os/exec"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
)

// A null processor doesn't actually do anything when it processes targets
// except mark them as processed.
type NullProcessor struct {
}

func (p NullProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	result := make(chan error)
	taskQueue <- common.CmdSpec{exec.Command("sleep", "5"), result, func() {}}
	return <-result
}
