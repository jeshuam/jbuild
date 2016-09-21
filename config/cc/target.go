package cc

import (
	"fmt"
	"os"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
	"github.com/jeshuam/jbuild/progress"
)

type TargetType int

const (
	Binary TargetType = iota
	Test
	Library
)

type Target struct {
	Spec         interfaces.TargetSpec
	Type         TargetType
	Srcs         []interfaces.Spec
	Hdrs         []interfaces.Spec
	Deps         []interfaces.TargetSpec
	Data         []interfaces.Spec
	CompileFlags []string
	LinkFlags    []string
	Includes     []interfaces.DirSpec
	Libs         []interfaces.Spec

	// Working arguments.
	_outputFile string
	_processed  bool
	_changed    bool
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////

func (this *Target) String() string {
	return fmt.Sprintf(
		"C++ Target: srcs=%s, hdrs=%s, compile_flags=%s, link_flags=%s",
		this.Srcs, this.Hdrs, this.CompileFlags, this.LinkFlags)
}

func (this *Target) GetType() string {
	switch this.Type {
	case Binary:
		return "c++/binary"
	case Test:
		return "c++/test"
	case Library:
		return "c++/library"
	}

	return ""
}

func (this *Target) Processed() bool {
	return this._processed
}

func (this *Target) TotalOps() int {
	return len(this.srcs()) + len(this.data()) + 1
}

func (this *Target) Dependencies() []interfaces.TargetSpec {
	deps := util.GetDependencies(this.Srcs)
	deps = append(deps, util.GetDependencies(this.Hdrs)...)
	deps = append(deps, util.GetDependencies(this.Data)...)
	deps = append(deps, util.GetDependencies(this.Libs)...)
	for _, dep := range this.Deps {
		deps = append(deps, dep)
	}
	return deps
}

func (this *Target) AllDependencies() []interfaces.TargetSpec {
	deps := util.GetAllDependencies(this.Srcs)
	deps = append(deps, util.GetAllDependencies(this.Hdrs)...)
	deps = append(deps, util.GetAllDependencies(this.Data)...)
	deps = append(deps, util.GetAllDependencies(this.Libs)...)
	for _, dep := range this.Deps {
		deps = append(deps, dep)
		deps = append(deps, dep.Target().AllDependencies()...)
	}
	return deps
}

func (this *Target) OutputFiles() []string {
	return []string{this._outputFile}
}

func (this *Target) Validate() error {
	// TODO(jeshua): Decide what to put here once JSON validation is done.
	return nil
}

func (this *Target) Process(progressBar *progress.ProgressBar, workQueue chan common.CmdSpec) error {
	// Make the output directory for this target.
	err := os.MkdirAll(this.Spec.OutputPath(), 0755)
	if err != nil {
		return err
	}

	// If there are no source files and this is a library, just finish.
	if this.IsLibrary() && len(this.srcs()) == 0 {
		progressBar.Finish()
		this._processed = true
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
	progressBar.SetOperation("copying data")
	err = copyData(this, progressBar)
	if err != nil {
		return err
	}

	// Save the output of this processing command.
	progressBar.Finish()
	this._outputFile = binary
	this._processed = true
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// IsLibrary returns true iff this target refers to a library output file.
func (this *Target) IsLibrary() bool {
	return this.Type == Library
}

// IsBinary returns true iff this target refers to a binary output file.
func (this *Target) IsBinary() bool {
	return this.Type == Binary
}

// IsTest returns true iff this target refers to a test output file.
func (this *Target) IsTest() bool {
	return this.Type == Test
}

// IsExecutable returns true iff the output of this target is executable.
func (this *Target) IsExecutable() bool {
	return this.IsTest() || this.IsBinary()
}

// extractFileSpecs goes through a list of generic specs and returns a list of
// file specs. It is assumed that the specs are either filegroup targets or
// FileSpecs.
func extractFileSpecs(specs []interfaces.Spec) []interfaces.FileSpec {
	fileSpecs := make([]interfaces.FileSpec, 0, len(specs))
	for _, spec := range specs {
		switch spec.(type) {
		case interfaces.FileSpec:
			fileSpecs = append(fileSpecs, spec.(interfaces.FileSpec))
		case interfaces.TargetSpec:
			target := spec.(interfaces.TargetSpec).Target().(*filegroup.Target)
			fileSpecs = append(fileSpecs, target.AllFiles()...)
		}
	}

	return fileSpecs
}

// Srcs returns a list of all sources for this current target with all
// filegroups expanded.
func (this *Target) srcs() []interfaces.FileSpec {
	return extractFileSpecs(this.Srcs)
}

// Hdrs returns a list of all headers for this current target with all
// filegroups expanded.
func (this *Target) hdrs() []interfaces.FileSpec {
	return extractFileSpecs(this.Hdrs)
}

// CompileFlags returns a list of all compile flags for this current target and
// all dependent targets with all filegroups expanded.
func (this *Target) compileFlags() []string {
	compileFlags := this.CompileFlags
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			compileFlags = append(compileFlags, dep.Target().(*Target).compileFlags()...)
		}
	}

	return compileFlags
}

// LinkFlags returns a list of all link flags for this current target and all
// dependent targets with all filegroups expanded.
func (this *Target) linkFlags() []string {
	linkFlags := this.LinkFlags
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			linkFlags = append(linkFlags, dep.Target().(*Target).linkFlags()...)
		}
	}

	return linkFlags
}

// Includes returns a list of all include paths for this current target and all
// dependent targets with all filegroups expanded.
func (this *Target) includes() []interfaces.DirSpec {
	includes := this.Includes
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			includes = append(includes, dep.Target().(*Target).includes()...)
		}
	}

	return includes
}

// Libs returns a list of all libs for this current target and all dependent
// targets with all filegroups expanded.
func (this *Target) libs() []interfaces.FileSpec {
	libs := extractFileSpecs(this.Libs)
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			libs = append(libs, dep.Target().(*Target).libs()...)
		}
	}

	return libs
}

// Data returns a list of all data for this current target and all dependent
// targets with all filegroups expanded.
func (this *Target) data() []interfaces.FileSpec {
	data := extractFileSpecs(this.Data)
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			data = append(data, dep.Target().(*Target).data()...)
		}
	}

	return data
}

// DepOutputs returns a list of output files for all dependencies recursively.
func (this *Target) depOutputs() []string {
	outputs := make([]string, 0)
	for _, dep := range this.AllDependencies() {
		switch dep.Target().(type) {
		case *Target:
			outputs = append(outputs, dep.Target().OutputFiles()...)
		}
	}

	return outputs
}

// DepsChangedSince returns true iff at least one of the dependencies has
// changed. This will scan through the header files of the dependencies and
// check whether they have changed relative to the given object.
func (this *Target) depsChangedSince(objStat os.FileInfo) bool {
	for _, depSpec := range this.AllDependencies() {
		switch depSpec.Target().(type) {
		case *Target:
			dep := depSpec.Target().(*Target)
			for _, depHdr := range dep.hdrs() {
				depHdrStat, _ := os.Stat(depHdr.FilePath())
				if depHdrStat.ModTime().After(objStat.ModTime()) {
					return true
				}
			}
		}
	}

	return false
}
