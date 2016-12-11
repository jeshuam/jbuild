package doxygen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
)

// A Filegroup is just a list of files. Filegroups can be nested, so you can
// have a filegroup which is a collection of filegroups.
type Target struct {
	Type       string
	Args       *args.Args
	_processed bool
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *Target) String() string {
	return fmt.Sprintf("doxygen: doxyfile=%s", this.DoxyfilePath())
}

func (this *Target) GetType() string {
	return this.Type
}

func (this *Target) Processed() bool {
	return this._processed
}

func (this *Target) TotalOps() int {
	return 1
}

func (this *Target) OutputFiles() []string {
	return []string{}
}

func (this *Target) Validate() error {
	return nil
}

func (this *Target) Process(args *args.Args, progressBar *progress.ProgressBar, taskQueue chan common.CmdSpec) error {
	// Make the doxygen command.
	cmd := exec.Command("doxygen", this.DoxyfilePath())

	// Doxygen environment.
	workspaceDir := args.WorkspaceDir
	outputDir := filepath.Join(args.OutputDir, "doc")

	// Make the output dir.
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	// Build up the environment.
	env := os.Environ()
	env = append(env, "WORKSPACE_DIR="+workspaceDir)
	env = append(env, "OUTPUT_DIR="+outputDir)
	cmd.Env = env

	// Run in the workspace dir.
	cmd.Dir = args.WorkspaceDir

	// Run doxygen.
	results := make(chan error)
	taskQueue <- common.CmdSpec{cmd, nil, results, nil}

	// Wait for it to finish.
	err = <-results
	if err != nil {
		return err
	}

	progressBar.Finish()
	this._processed = true
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//                              Utility Functions                             //
////////////////////////////////////////////////////////////////////////////////
func (this *Target) DoxyfilePath() string {
	return strings.Replace(
		filepath.Join(this.Args.WorkspaceDir, "Doxyfile"), "\\", "\\\\", -1)
}
