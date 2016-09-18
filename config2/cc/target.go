package cc

import (
	"fmt"
	"os"
	"strings"

	"github.com/deckarep/golang-set"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/filegroup"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/jeshuam/jbuild/config2/util"
	"github.com/jeshuam/jbuild/progress"
)

type TargetType int

const (
	Binary TargetType = iota
	Test
	Library
)

type Target struct {
	// Input arguments.
	Spec         interfaces.Spec
	OutputType   TargetType
	Srcs         []interfaces.Spec
	Hdrs         []interfaces.Spec
	Deps         []interfaces.Spec
	Data         []interfaces.Spec
	CompileFlags []string
	LinkFlags    []string
	Includes     []string
	Libs         []interfaces.Spec

	// Output arguments.
	Output struct {
		File string
	}

	_processed bool
}

func (this *Target) String() string {
	return fmt.Sprintf("srcs=%s, hdrs=%s, compile_flags=%s, link_flags=%s", this.Srcs, this.Hdrs, this.CompileFlags, this.LinkFlags)
}

func (this *Target) Type() string {
	switch this.OutputType {
	case Binary:
		return "c++/binary"
	case Test:
		return "c++/test"
	case Library:
		return "c++/library"
	}

	return "c++/unknown"
}

func (this *Target) Validate() error {
	err := util.EnsureDependenciesAreOfType(this.Srcs, mapset.NewSet("file", "filegroup"))
	if err != nil {
		return err
	}

	// Ensure dependencies are valid.
	for _, dep := range this.Dependencies() {
		depTarget := util.TargetCache[dep.String()]
		err = depTarget.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *Target) DirectDependencies() []interfaces.Spec {
	deps := make([]interfaces.Spec, 0, len(this.Srcs)+len(this.Hdrs)+len(this.Deps))
	deps = append(deps, util.GetDirectDependencies(this.Srcs)...)
	deps = append(deps, util.GetDirectDependencies(this.Hdrs)...)
	deps = append(deps, util.GetDirectDependencies(this.Deps)...)
	return deps
}

func (this *Target) Dependencies() []interfaces.Spec {
	deps := make([]interfaces.Spec, 0, len(this.Srcs)+len(this.Hdrs)+len(this.Deps))
	deps = append(deps, util.GetDependencies(this.Srcs)...)
	deps = append(deps, util.GetDependencies(this.Hdrs)...)
	deps = append(deps, util.GetDependencies(this.Deps)...)
	return deps
}

func (this *Target) Processed() bool {
	return this._processed
}

// Some C++ specific functions.
func (this *Target) IsLibrary() bool {
	return this.Type() == "c++/library"
}

func (this *Target) IsBinary() bool {
	return this.Type() == "c++/binary"
}

func (this *Target) IsTest() bool {
	return this.Type() == "c++/test"
}

func (this *Target) IsExecutable() bool {
	return this.IsTest() || this.IsBinary()
}

func (this *Target) extractAllSpecs(specs []interfaces.Spec) []interfaces.Spec {
	allSpecs := make([]interfaces.Spec, 0, len(specs))
	for _, spec := range specs {
		if spec.IsTarget() {
			allSpecs = append(allSpecs, spec.Target().(*filegroup.Target).ExtractAllFiles()...)
		} else {
			allSpecs = append(allSpecs, spec)
		}
	}

	return allSpecs
}

func (this *Target) AllSrcs() []interfaces.Spec {
	return this.extractAllSpecs(this.Srcs)
}

func (this *Target) AllLibs() []interfaces.Spec {
	allLibs := this.Libs
	for _, dep := range this.Dependencies() {
		if dep.IsTarget() && strings.HasPrefix(dep.Target().Type(), "c++") {
			for _, lib := range dep.Target().(*Target).Libs {
				allLibs = append(allLibs, lib)
			}
		}
	}

	return allLibs
}

func (this *Target) DepOutputs() []string {
	output := make([]string, 0, len(this.Dependencies()))
	for _, dep := range this.Dependencies() {
		if strings.HasPrefix(dep.Target().Type(), "c++") {
			output = append(output, dep.Target().(*Target).Output.File)
		}
	}

	return output
}

func (this *Target) Process(progressBar *progress.ProgressBar, workQueue chan common.CmdSpec) error {
	// Make the output directory for this target.
	err := os.MkdirAll(this.Spec.OutputPath(), 0755)
	if err != nil {
		return err
	}

	// If there are no source files and this is a library, just finish.
	if this.IsLibrary() && len(this.Srcs) == 0 {
		progressBar.Finish()
		return nil
	}

	// Compile all of the source files.
	progressBar.SetOperation("compiling")
	objFiles, nCompiled, err := compileFiles(this, progressBar, workQueue)
	if err != nil {
		return err
	}

	// Link all object files into a binary. What this binary is depends on the
	// type of the target. We only have to do that if something in the target was
	// compiled (this should avoid expensive and pointless linking steps).
	progressBar.SetOperation("linking")
	binary, err := linkObjects(this, progressBar, workQueue, objFiles, nCompiled)
	if err != nil {
		return err
	}

	// Copy the data to the output directory.
	for _, data := range this.Data {
		// Data files are either plain files or filegroup targets. Make a list of
		// all files that this data item references, and do the copy in one go.
		dataSpecs := make([]interfaces.Spec, 0, 1)
		if !data.IsTarget() {
			dataSpecs = append(dataSpecs, data)
		} else {
			dataSpecs = data.Target().(*filegroup.Target).ExtractAllFiles()
		}

		for _, dataSpec := range dataSpecs {
			dataStat, _ := os.Stat(dataSpec.Path())
			dataOutStat, _ := os.Stat(dataSpec.OutputPath())
			if dataOutStat != nil && dataStat.ModTime().After(dataOutStat.ModTime()) {
				os.Remove(dataSpec.OutputPath())
			}

			if !common.FileExists(dataSpec.OutputPath()) {
				err := os.Link(dataSpec.Path(), dataSpec.OutputPath())
				if err != nil {
					return err
				}
			}
		}
	}

	// Save the output of this processing command.
	progressBar.Finish()
	this.Output.File = binary
	this._processed = true
	return nil
}

func (this *Target) TotalOps() int {
	return len(this.Srcs) + 1
}

// Some useful utility functions for processing this target.