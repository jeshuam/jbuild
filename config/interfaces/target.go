package interfaces

import (
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
)

type Target interface {
	// String returns a string representation of the target for debugging.
	String() string

	// Type returns a unique string specifying the type of target it is.
	GetType() string

	// Processed returns whether the target has been processed or not.
	Processed() bool

	// TotalOps returns the number of operations required to process the target.
	// If set to 0, no progress bar will be shown.
	TotalOps() int

	// Dependencies returns a list of all direct target dependencies.
	Dependencies() []TargetSpec

	// AllDependencies returns a list of all direct and indirect target
	// dependencies.
	AllDependencies() []TargetSpec

	// Get a list of the objects output by this target. These will change
	// depending on the type of the target, but could be executable binaries or
	// library names or source files. For executable targets, the first output
	// file returned should be the executable binary.
	OutputFiles() []string

	// Validate ensures that the internal structures of the target are valid for
	// processing.
	Validate() error

	// Process will perform the necessary actions required to transition the
	// target from not processed --> processed. The channel provided must be used
	// to asynchronously perform work. This is required to ensure that the number
	// of threads used to perform external operations is constrained.
	Process(progressBar *progress.ProgressBar, workQueue chan common.CmdSpec) error
}
