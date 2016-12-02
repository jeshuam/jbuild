package genrule

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
)

type Target struct {
	Type string
	Spec interfaces.TargetSpec // The spec of this target.
	Args *args.Args            // Program arguments.
	In   []interfaces.Spec     `types:"file,filegroup"` // The set of input files.
	Out  []string              // The output files created.

	// The command to run. The command will be run in a directory with the same
	// structure as the workspace, from the root.
	Cmd    []string
	CmdOut string // The file to save the output to. Not used if blank.
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *Target) String() string {
	return fmt.Sprintf("genrule: in=%s, out=%s, cmd=%s", this.In, this.Out, this.Cmd)
}

func (this *Target) GetType() string {
	return "genrule"
}

func (this *Target) Processed() bool {
	return true
}

func (this *Target) TotalOps() int {
	return 0
}

func (this *Target) Dependencies() []interfaces.TargetSpec {
	return util.GetDependencies(this.In)
}

func (this *Target) AllDependencies() []interfaces.TargetSpec {
	return util.GetAllDependencies(this.In)
}

func (this *Target) OutputFiles() []string {
	output := make([]string, 0, len(this.Out))
	for _, file := range this.Out {
		output = append(output, filepath.Join(this.outputDir(), file))
	}

	return output
}

func (this *Target) Validate() error {
	return nil
}

func (this *Target) Process(args *args.Args, progress *progress.ProgressBar, workQueue chan common.CmdSpec) error {
	// Make a temporary directory.
	tempDir, err := ioutil.TempDir("", filepath.Base(args.WorkspaceDir))
	if err != nil {
		return err
	}

	// Delete it when done.
	// defer os.RemoveAll(tempDir)

	// Copy all input files to a temporary directory.
	for _, spec := range this.In {
		switch spec.(type) {
		case interfaces.TargetSpec:
			filegroup := spec.(interfaces.TargetSpec).Target().(*filegroup.Target)
			for _, fileSpec := range filegroup.AllFiles() {
				dest := filepath.Join(tempDir, fileSpec.WorkspacePath(), fileSpec.File())
				os.MkdirAll(filepath.Dir(dest), 0755)
				os.Link(fileSpec.FilePath(), dest)
			}

		case interfaces.FileSpec:
			fileSpec := spec.(interfaces.FileSpec)
			dest := filepath.Join(tempDir, fileSpec.WorkspacePath(), fileSpec.File())
			os.MkdirAll(filepath.Dir(dest), 0755)
			os.Link(fileSpec.FilePath(), dest)
		}
	}

	// Run the command in the given directory.
	result := make(chan error)
	cmd := exec.Command(this.Cmd[0], this.Cmd[1:]...)
	cmd.Dir = tempDir

	// Prepare the output file.
	var outFile *os.File
	if this.CmdOut != "" {
		os.MkdirAll(filepath.Join(tempDir, filepath.Dir(this.CmdOut)), 0755)
		outFile, err = os.Create(filepath.Join(tempDir, this.CmdOut))
		if err != nil {
			return err
		}

		cmd.Stdout = outFile
		cmd.Stderr = outFile
	}

	// Run the command.
	workQueue <- common.CmdSpec{cmd, nil, result, nil}

	// Wait for the result.
	err = <-result
	if err != nil {
		return err
	}

	outFile.Close()

	// Make sure the output files were created and then copy them to the generated
	// folder.
	for _, outputFile := range this.Out {
		finalOutputFilePath := filepath.Join(this.outputDir(), outputFile)
		createdOutputFile := filepath.Join(tempDir, outputFile)
		if exists, _ := os.Stat(createdOutputFile); exists != nil {
			os.MkdirAll(filepath.Dir(finalOutputFilePath), 0755)
			os.Link(createdOutputFile, finalOutputFilePath)
		} else {
			return errors.New(fmt.Sprintf("Required file %s was not created", outputFile))
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// outputDir returns the generated output directory for this genrule. This is a
// function of the genrule's location and the workspace root.
func (this *Target) outputDir() string {
	return filepath.Join(this.Args.OutputDir, "gen", this.Spec.Dir())
}
