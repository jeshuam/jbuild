package genrule

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

type Target struct {
	Type string
	Spec interfaces.TargetSpec // The spec of this target.
	Args *args.Args            // Program arguments.
	In   []interfaces.Spec     `types:"file,filegroup"` // The set of input files.
	Out  []interfaces.FileSpec `generated:"true"`       // The output files created.

	// The command to run. The command will be run in a directory with the same
	// structure as the workspace, from the root.
	Cmds []string
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *Target) String() string {
	return fmt.Sprintf("genrule: in=%s, out=%s, cmd=%s", this.In, this.Out, this.Cmds)
}

func (this *Target) GetType() string {
	return "genrule"
}

func (this *Target) Processed() bool {
	// Find the newest input file.
	var newestInputFile os.FileInfo
	for _, inFile := range this.in() {
		inStat, _ := os.Stat(inFile.FsPath())
		if newestInputFile == nil || inStat.ModTime().After(newestInputFile.ModTime()) {
			newestInputFile = inStat
		}
	}

	// Check the BUILD/WORKSPACE file has been updated.
	buildStat, _ := os.Stat(filepath.Join(this.Spec.Path(), this.Args.BuildFilename))
	workspaceStat, _ := os.Stat(filepath.Join(this.Args.WorkspaceDir, this.Args.WorkspaceFilename))

	// If the output files exist, then it has been processed.
	for _, outFile := range this.OutputFiles() {
		outFileStat, _ := os.Stat(outFile)
		if outFileStat == nil {
			return false
		}

		if newestInputFile.ModTime().After(outFileStat.ModTime()) {
			return false
		}

		if buildStat != nil && buildStat.ModTime().After(outFileStat.ModTime()) {
			return false
		}

		if workspaceStat != nil && workspaceStat.ModTime().After(outFileStat.ModTime()) {
			return false
		}
	}

	return true
}

func (this *Target) TotalOps() int {
	return len(this.Cmds)
}

func (this *Target) OutputFiles() []string {
	output := make([]string, 0, len(this.Out))
	for _, file := range this.Out {
		output = append(output, file.FsOutputPath())
	}

	return output
}

func (this *Target) Validate() error {
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}

	return dstFile.Close()
}

func (this *Target) Process(args *args.Args, progress *progress.ProgressBar, workQueue chan common.CmdSpec) error {
	log := logging.MustGetLogger("jbuild")

	// If processed, skip.
	if this.Processed() {
		return nil
	}

	// Make a temporary directory.
	tempDir, err := ioutil.TempDir("", filepath.Base(args.WorkspaceDir))
	if err != nil {
		return err
	}

	// Delete it when done.
	defer os.RemoveAll(tempDir)

	// Copy all input files to a temporary directory.
	for _, spec := range this.In {
		switch spec.(type) {
		case interfaces.TargetSpec:
			filegroup := spec.(interfaces.TargetSpec).Target().(*filegroup.Target)
			for _, fileSpec := range filegroup.AllFiles() {
				dest := filepath.Join(tempDir, fileSpec.Dir(), fileSpec.Filename())
				os.MkdirAll(filepath.Dir(dest), 0755)
				copyFile(fileSpec.FsPath(), dest)
			}

		case interfaces.FileSpec:
			fileSpec := spec.(interfaces.FileSpec)
			dest := filepath.Join(tempDir, fileSpec.Dir(), fileSpec.Filename())
			os.MkdirAll(filepath.Dir(dest), 0755)
			copyFile(fileSpec.FsPath(), dest)
		}
	}

	// Run each command.
	result := make(chan error)
	for _, cmdString := range this.Cmds {
		// First, see if we are redirecting.
		cmdParts := strings.Split(cmdString, ">")
		outputFile := ""
		if len(cmdParts) > 1 {
			outputFile = filepath.Join(this.Spec.Dir(), strings.TrimSpace(cmdParts[len(cmdParts)-1]))
		}

		// Split the string into parts, ignoring the redirect.
		cmdTokens, err := shlex.Split(cmdParts[0])
		if err != nil {
			return err
		}

		cmd := exec.Command(cmdTokens[0], cmdTokens[1:]...)
		cmd.Dir = tempDir
		complete := func(out string, _ bool, _ time.Duration) {
			if outputFile != "" {
				os.MkdirAll(filepath.Join(tempDir, filepath.Dir(outputFile)), 0755)
				ioutil.WriteFile(filepath.Join(tempDir, outputFile), []byte(out), 0755)
			}
		}

		// Run the command.
		log.Debugf("... run %s", cmd.Args)
		workQueue <- common.CmdSpec{cmd, nil, result, complete}

		// Wait for the result.
		err = <-result
		if err != nil {
			return err
		}
	}

	// Make sure the output files were created and then copy them to the generated
	// folder.
	for _, outputFile := range this.Out {
		finalOutputFilePath := outputFile.FsOutputPath()
		createdOutputFile := filepath.Join(tempDir, outputFile.Dir(), outputFile.Filename())
		if exists, _ := os.Stat(createdOutputFile); exists != nil {
			os.MkdirAll(filepath.Dir(finalOutputFilePath), 0755)
			copyFile(createdOutputFile, finalOutputFilePath)
		} else {
			return errors.New(fmt.Sprintf("Required file %s was not created", outputFile))
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// Get a full list of input files.
func (this *Target) in() []interfaces.FileSpec {
	fileSpecs := make([]interfaces.FileSpec, 0, len(this.In))
	for _, spec := range this.In {
		switch spec.(type) {
		case interfaces.TargetSpec:
			filegroup := spec.(interfaces.TargetSpec).Target().(*filegroup.Target)
			fileSpecs = append(fileSpecs, filegroup.AllFiles()...)

		case interfaces.FileSpec:
			fileSpecs = append(fileSpecs, spec.(interfaces.FileSpec))
		}
	}

	return fileSpecs
}
