package filegroup

import (
	// "errors"
	"fmt"

	"github.com/deckarep/golang-set"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/jeshuam/jbuild/config2/util"
	"github.com/jeshuam/jbuild/progress"
)

type Target struct {
	Files []interfaces.Spec
}

func (this *Target) String() string {
	return fmt.Sprintf("files=%s", this.Files)
}

func (this *Target) Type() string {
	return "filegroup"
}

func (this *Target) Validate() error {
	err := util.EnsureDependenciesAreOfType(this.Files, mapset.NewSet("file", "filegroup"))
	if err != nil {
		return err
	}

	return nil
}

func (this *Target) DirectDependencies() []interfaces.Spec {
	deps := make([]interfaces.Spec, 0, len(this.Files))
	deps = append(deps, util.GetDirectDependencies(this.Files)...)
	return deps
}

func (this *Target) Dependencies() []interfaces.Spec {
	deps := make([]interfaces.Spec, 0, len(this.Files))
	deps = append(deps, util.GetDependencies(this.Files)...)
	return deps
}

func (this *Target) Processed() bool {
	return true
}

func (this *Target) Process(*progress.ProgressBar, chan common.CmdSpec) error {
	return nil
}

func (this *Target) TotalOps() int {
	return 0
}

func (this *Target) ExtractAllFiles() []interfaces.Spec {
	files := make([]interfaces.Spec, 0, len(this.Files))
	for _, file := range this.Files {
		if file.IsTarget() {
			files = append(files, file.Target().(*Target).ExtractAllFiles()...)
		} else {
			files = append(files, file)
		}
	}

	return files
}
