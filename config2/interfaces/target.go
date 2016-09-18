package interfaces

import (
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
)

type Target interface {
	// Display a string representation of the target, mostly for debugging.
	String() string

	// Ensure that this target is valid. Will only be called after Load.
	Validate() error

	// Return a list of targets on which this target depends.
	Dependencies() []Spec
	DirectDependencies() []Spec

	Processed() bool

	// Process this target.
	Process(progressBar *progress.ProgressBar, workQueue chan common.CmdSpec) error

	TotalOps() int

	Type() string
}
