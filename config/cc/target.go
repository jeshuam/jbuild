package cc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/genrule"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/progress"
)

type TargetType int

const (
	Unknown TargetType = iota
	Binary
	Test
	Library
)

type Target struct {
	Spec         interfaces.TargetSpec
	Args         *args.Args
	Type         TargetType
	Srcs         []interfaces.Spec       `types:"file,filegroup,genrule"`
	Hdrs         []interfaces.Spec       `types:"file,filegroup,genrule"`
	Deps         []interfaces.TargetSpec `types:"c++/library,genrule"`
	Data         []interfaces.Spec       `types:"file,filegroup"`
	CompileFlags []string
	LinkFlags    []string
	Includes     []interfaces.DirSpec
	Libs         []interfaces.Spec `types:"file,filegroup"`
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
	default:
		return "c++/unknown"
	}
}

func (this *Target) Processed() bool {
	// If there are no source files for this target, then we have to be finished.
	// This is because there must be no output. It's probably just a bag of
	// headers or something.
	if len(this.srcs()) == 0 {
		return true
	}

	// If the output file doesn't exist, then this target isn't processed.
	if !common.FileExists(this.OutputPath()) {
		return false
	}

	// If the BUILD or WORKSPACE files that this file was build in have changed,
	// then we haven't processed.
	outputStat, _ := os.Stat(this.OutputPath())
	buildStat, _ := os.Stat(filepath.Join(this.Spec.Path(), this.Args.BuildFilename))

	// If we couldn't find the BUILD file, maybe it was an external?
	if buildStat == nil {
		externalRepo, ok := this.Args.ExternalRepos["//"+this.Spec.Dir()]
		if !ok {
			panic("Build file not found for non-external repo??")
		}

		if externalRepo.BuildFile != "" {
			buildStat, _ = os.Stat(externalRepo.BuildFile)
		}
	}

	if buildStat != nil && buildStat.ModTime().After(outputStat.ModTime()) {
		return false
	}

	workspaceStat, _ := os.Stat(filepath.Join(this.Args.WorkspaceDir, this.Args.WorkspaceFilename))
	if workspaceStat.ModTime().After(outputStat.ModTime()) {
		return false
	}

	// Validate each file.
	for _, fileSpec := range this.files() {
		fileStat, _ := os.Stat(fileSpec.FsPath())
		if fileStat.ModTime().After(outputStat.ModTime()) {
			return false
		}
	}

	// We haven't been processed if our dependencies haven't been.
	for _, depSpec := range this.Spec.Dependencies(true) {
		if !depSpec.Target().Processed() {
			return false
		}
	}

	// If one of our deps has updated, we should too.
	if this.depsUpdated() {
		return false
	}

	return true
}

func (this *Target) TotalOps() int {
	return len(this.srcs()) + len(this.data()) + 1
}

func (this *Target) OutputFiles() []string {
	if len(this.srcs()) == 0 {
		return []string{}
	}

	return []string{this.OutputPath()}
}

func (this *Target) Validate() error {
	// TODO(jeshua): Decide what to put here once JSON validation is done.
	return nil
}

func (this *Target) Process(args *args.Args, progressBar *progress.ProgressBar, workQueue chan common.CmdSpec) error {
	// Make the output directory for this target.
	err := os.MkdirAll(this.Spec.OutputPath(), 0755)
	if err != nil {
		return err
	}

	// Check if we should force compile.
	outputStat, _ := os.Stat(this.OutputPath())
	forceCompile := false
	if outputStat != nil {
		buildStat, _ := os.Stat(filepath.Join(this.Spec.Path(), this.Args.BuildFilename))
		workspaceStat, _ := os.Stat(filepath.Join(this.Args.WorkspaceDir, this.Args.WorkspaceFilename))

		// If we couldn't find the BUILD file, maybe it was an external?
		if buildStat == nil {
			externalRepo, ok := this.Args.ExternalRepos["//"+this.Spec.Dir()]
			if !ok {
				panic("Build file not found for non-external repo??")
			}

			if externalRepo.BuildFile != "" {
				buildStat, _ = os.Stat(externalRepo.BuildFile)
			}
		}

		// Check if header files are newer than the output file.
		for _, hdrFile := range this.hdrs() {
			hdrStat, _ := os.Stat(hdrFile.FsPath())
			if hdrStat.ModTime().After(outputStat.ModTime()) {
				forceCompile = true
				break
			}
		}

		forceCompile = forceCompile ||
			(buildStat != nil && buildStat.ModTime().After(outputStat.ModTime())) ||
			workspaceStat.ModTime().After(outputStat.ModTime())
	}

	// If there are no source files and this is a library, just finish.
	if this.IsLibrary() && len(this.srcs()) == 0 {
		progressBar.Finish()
		return nil
	}

	// Compile all of the source files.
	progressBar.SetOperation("compiling")
	objFiles, nCompiled, err := compileFiles(args, this, progressBar, workQueue, forceCompile)
	if err != nil {
		return err
	}

	// Link all object files into a binary. What this binary is depends on the
	// type of the target. We only have to do that if something in the target was
	// compiled (this should avoid expensive and pointless linking steps).
	progressBar.SetOperation("linking")
	_, err = linkObjects(args, this, progressBar, workQueue, objFiles, nCompiled)
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

// OutputPath returns the fully specified output path for this target.
func (this *Target) OutputPath() string {
	if this.IsLibrary() {
		return filepath.Join(this.Spec.OutputPath(), LibraryName(this.Spec.Name()))
	} else {
		return filepath.Join(this.Spec.OutputPath(), BinaryName(this.Spec.Name()))
	}
}

// extractFileSpecs goes through a list of generic specs and returns a list of
// file specs. It is assumed that the specs are either filegroup targets,
// genrules or FileSpecs. If suffixes is supplied, then make sure each file has
// that suffix.
func extractFileSpecs(specs []interfaces.Spec, suffixes []string) []interfaces.FileSpec {
	fileSpecs := make([]interfaces.FileSpec, 0, len(specs))
	for _, spec := range specs {
		switch spec.(type) {
		case interfaces.FileSpec:
			fileSpecs = append(fileSpecs, spec.(interfaces.FileSpec))
		case interfaces.TargetSpec:
			targetSpec := spec.(interfaces.TargetSpec)
			switch targetSpec.Target().(type) {
			case *filegroup.Target:
				target := targetSpec.Target().(*filegroup.Target)
				fileSpecs = append(fileSpecs, target.AllFiles()...)
			case *genrule.Target:
				target := targetSpec.Target().(*genrule.Target)
				fileSpecs = append(fileSpecs, target.Out...)
			}
		}
	}

	// Filter the file specs if necessary.
	if suffixes != nil {
		filteredFileSpecs := make([]interfaces.FileSpec, 0, len(fileSpecs))
		for _, fileSpec := range fileSpecs {
			for _, suffix := range suffixes {
				if strings.HasSuffix(fileSpec.Filename(), suffix) {
					filteredFileSpecs = append(filteredFileSpecs, fileSpec)
				}
			}
		}

		return filteredFileSpecs
	}

	return fileSpecs
}

// files returns a list of all files for this target (i.e. source files, data
// files and header files).
func (this *Target) files() []interfaces.FileSpec {
	fileSpecs := this.srcs()
	fileSpecs = append(fileSpecs, this.hdrs()...)
	fileSpecs = append(fileSpecs, this.data()...)
	fileSpecs = append(fileSpecs, this.libs()...)
	return fileSpecs
}

// Srcs returns a list of all sources for this current target with all
// filegroups expanded.
func (this *Target) srcs() []interfaces.FileSpec {
	return extractFileSpecs(this.Srcs, []string{".cc", ".cpp", ".c", ".cxx"})
}

// Hdrs returns a list of all headers for this current target with all
// filegroups expanded.
func (this *Target) hdrs() []interfaces.FileSpec {
	return extractFileSpecs(this.Hdrs, []string{".h", ".hpp"})
}

// CompileFlags returns a list of all compile flags for this current target and
// all dependent targets with all filegroups expanded.
func (this *Target) compileFlags() []string {
	compileFlags := this.CompileFlags
	for _, dep := range this.Spec.Dependencies(true) {
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
	for _, dep := range this.Spec.Dependencies(true) {
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
	for _, dep := range this.Spec.Dependencies(true) {
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
	libs := extractFileSpecs(this.Libs, nil)
	for _, dep := range this.Spec.Dependencies(true) {
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
	data := extractFileSpecs(this.Data, nil)
	for _, dep := range this.Spec.Dependencies(true) {
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
	for _, dep := range this.Spec.Dependencies(true) {
		switch dep.Target().(type) {
		case *Target:
			if dep.Target().GetType() == "c++/library" {
				outputs = append(outputs, dep.Target().OutputFiles()...)
			}
		}
	}

	return outputs
}

// DepsChangedSince returns true iff at least one of the dependencies has
// changed. This will scan through the header files of the dependencies and
// check whether they have changed relative to the given object.
func (this *Target) depsChangedSince(objStat os.FileInfo) bool {
	for _, depSpec := range this.Spec.Dependencies(true) {
		switch depSpec.Target().(type) {
		case *Target:
			dep := depSpec.Target().(*Target)
			for _, depFile := range dep.hdrs() {
				depFileStat, _ := os.Stat(depFile.FsPath())
				if depFileStat.ModTime().After(objStat.ModTime()) {
					return true
				}
			}
		}
	}

	return false
}

// depsUpdated returns true iff at least one of the dependencies outputs has
// changed relative to this target's output.
func (this *Target) depsUpdated() bool {
	// Check all of our object files, and compare them.
	for _, outFile := range this.OutputFiles() {
		outFileStat, _ := os.Stat(outFile)
		for _, depOutFile := range this.depOutputs() {
			depOutFileStat, _ := os.Stat(depOutFile)
			if depOutFileStat.ModTime().After(outFileStat.ModTime()) {
				return true
			}
		}
	}

	return false
}
