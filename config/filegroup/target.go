package filegroup

import (
	"fmt"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
)

// A Filegroup is just a list of files. Filegroups can be nested, so you can
// have a filegroup which is a collection of filegroups.
type Target struct {
	Type  string
	Files []interfaces.Spec `types:"file,filegroup"`
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *Target) String() string {
	return fmt.Sprintf("filegroup: files=%s", this.Files)
}

func (this *Target) GetType() string {
	return this.Type
}

func (this *Target) Processed() bool {
	return true
}

func (this *Target) TotalOps() int {
	return 0
}

func (this *Target) Dependencies() []interfaces.TargetSpec {
	return util.GetDependencies(this.Files)
}

func (this *Target) AllDependencies() []interfaces.TargetSpec {
	return util.GetAllDependencies(this.Files)
}

func (this *Target) OutputFiles() []string {
	output := make([]string, 0, len(this.files()))
	for _, file := range this.files() {
		output = append(output, file.FilePath())
	}

	return output
}

func (this *Target) Validate() error {
	// TODO(jeshua): validate once JSON validation tags are implemented.
	return nil
}

func (this *Target) Process(*progress.ProgressBar, chan common.CmdSpec) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// Files returns a list of FileSpec objects contained within this filegroup.
func (this *Target) files() []interfaces.FileSpec {
	files := make([]interfaces.FileSpec, 0, len(this.Files))
	for _, fileSpec := range this.Files {
		switch fileSpec.(type) {
		case interfaces.FileSpec:
			files = append(files, fileSpec.(interfaces.FileSpec))
		}
	}

	return files
}

// AllFiles returns a list of the FileSpec objects contained within this
// filegroup and all contained filegroups.
func (this *Target) AllFiles() []interfaces.FileSpec {
	files := this.files()
	for _, fileSpec := range this.Files {
		switch fileSpec.(type) {
		case interfaces.TargetSpec:
			filegroup := fileSpec.(interfaces.TargetSpec).Target().(*Target)
			files = append(files, filegroup.AllFiles()...)
		}
	}

	return files
}
