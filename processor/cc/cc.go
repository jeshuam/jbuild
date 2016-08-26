package cc

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")

	ccThreads       = flag.Int("cc_threads", runtime.NumCPU()+1, "Number of threads to use when performing C++ operations.")
	ccCompiler      = flag.String("cc_compiler", "", "The C++ compiler to use.")
	ccStaticLinking = flag.Bool("cc_static_linking", true, "Whether or not to use static linking.")

	defaultCompilers = map[string]string{
		"windows": "cl.exe",
	}
)

type CCProcessor struct {
}

func init() {
	// Set the default compiler.
	if *ccCompiler == "" {
		compiler, ok := defaultCompilers[runtime.GOOS]
		if !ok {
			compiler = "clang++"
		}

		*ccCompiler = compiler
	}
}

// Compile the source files within the given target.
func compileFiles(target *config.Target, taskQueue chan common.CmdSpec) ([]string, int, error) {
	objs := make([]string, len(target.Srcs()))
	results := make(chan error, len(target.Srcs()))
	nCompiled := 0

	for i, srcFile := range target.Srcs() {
		// Display the source file we are building.
		target.ProgressBar.SetSuffix(srcFile)

		// Work out the full path to the source file. This will need to be provided
		// to the compiler.
		srcPath := filepath.Join(target.Spec.Workspace, target.Spec.PathSystem(), srcFile)
		objPath := filepath.Join(target.Spec.OutputPath(), srcFile+".o")
		objs[i] = objPath

		// Make the directory of the obj if needed.
		err := os.MkdirAll(filepath.Dir(objPath), 0755)
		if err != nil {
			return nil, 0, err
		}

		// Check if any dependent header files have been updated.
		srcStat, _ := os.Stat(srcPath)
		depsChanged := target.HeaderFilesChangedAfter(srcStat)

		// If the object is newer than the source file, don't compile it again.
		objStat, _ := os.Stat(objPath)
		srcChanged := true
		if objStat != nil {
			srcChanged = !objStat.ModTime().After(srcStat.ModTime())
		}

		// Recompile this file if the deps or src has changed.
		if !depsChanged && !srcChanged {
			target.ProgressBar.Increment()
			continue
		} else {
			log.Debugf("Compiling file %s: depsChanged=%v, srcChanged=%v", srcFile, depsChanged, srcChanged)
		}

		// Build the compilation command.
		cmd := compileCommand(target, srcPath, objPath)

		// Run the command.
		nCompiled++
		taskQueue <- common.CmdSpec{cmd, results, func(error) {
			target.ProgressBar.Increment()
		}}
	}

	// Check results.
	for i := 0; i < nCompiled; i++ {
		err := <-results
		if err != nil {
			return nil, 0, err
		}
	}

	return objs, nCompiled, nil
}

func linkObjects(target *config.Target, taskQueue chan common.CmdSpec, objects []string, nCompiled int) (string, error) {
	// Throw and error if there are no source files and this isn't a library.
	if target.IsBinary() && (len(target.Srcs()) == 0 && len(target.Deps) == 0) {
		return "", errors.New(fmt.Sprintf("No source files/deps found for binary %s", target))
	}

	// First, work out what the name of the output is.
	var outputName string
	if target.IsLibrary() {
		outputName = libraryName(target.Spec.Name)
	} else if target.IsExecutable() {
		outputName = binaryName(target.Spec.Name)
	}

	// Work out the output filepath.
	dependenciesChanged := target.IsExecutable() && target.DependenciesChanged()
	outputPath := filepath.Join(target.Spec.OutputPath(), outputName)
	if nCompiled == 0 && common.FileExists(outputPath) && !dependenciesChanged {
		target.ProgressBar.Increment()
		target.Changed = false
		return outputPath, nil
	}

	// Make the error channel.
	target.Changed = true
	result := make(chan error)

	// Now, we need to build up the command to run.
	cmd := linkCommand(target, objects, outputPath)

	// Run the command.
	taskQueue <- common.CmdSpec{cmd, result, func(error) {
		target.ProgressBar.Increment()
	}}

	err := <-result
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func (p CCProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	// Make the output directory for this target.
	err := os.MkdirAll(target.Spec.OutputPath(), 0755)
	if err != nil {
		return err
	}

	// If there are no source files and this is a library, just finish.
	if target.IsLibrary() && len(target.Srcs()) == 0 {
		target.ProgressBar.Finish()
		return nil
	}

	// Compile all of the source files.
	target.ProgressBar.SetOperation("compiling")
	objFiles, nCompiled, err := compileFiles(target, taskQueue)
	if err != nil {
		return err
	}

	// Link all object files into a binary. What this binary is depends on the
	// type of the target. We only have to do that if something in the target was
	// compiled (this should avoid expensive and pointless linking steps).
	target.ProgressBar.SetOperation("linking")
	binary, err := linkObjects(target, taskQueue, objFiles, nCompiled)
	if err != nil {
		return err
	}

	// Save the output of this processing command.
	target.ProgressBar.Finish()
	target.Output = append(target.Output, binary)

	// All finished, with no errors!
	return nil
}
