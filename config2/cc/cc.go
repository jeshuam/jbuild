package cc

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")

	ccThreads       = flag.Int("cc_threads2", runtime.NumCPU()+1, "Number of threads to use when performing C++ operations.")
	ccCompiler      = flag.String("cc_compiler2", "", "The C++ compiler to use.")
	ccStaticLinking = flag.Bool("cc_static_linking2", true, "Whether or not to use static linking.")

	defaultCompilers = map[string]string{
		"windows": "cl.exe",
	}
)

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
func compileFiles(target *Target, progressBar *progress.ProgressBar, taskQueue chan common.CmdSpec) ([]string, int, error) {
	objs := make([]string, len(target.AllSrcs()))
	results := make(chan error, len(target.AllSrcs()))
	nCompiled := 0

	for i, srcFile := range target.AllSrcs() {
		// Display the source file we are building.
		progressBar.SetSuffix(srcFile.String())

		// Work out the full path to the source file. This will need to be provided
		// to the compiler.
		srcPath := srcFile.Path()
		objPath := srcFile.OutputPath() + ".o"
		objs[i] = objPath

		// Make the directory of the obj if needed.
		err := os.MkdirAll(filepath.Dir(objPath), 0755)
		if err != nil {
			return nil, 0, err
		}

		// If the object is newer than the source file, don't compile it again.
		srcStat, _ := os.Stat(srcPath)
		objStat, _ := os.Stat(objPath)
		srcChanged := true
		depsChanged := true
		if objStat != nil {
			srcChanged = !objStat.ModTime().After(srcStat.ModTime())
		}

		// Recompile this file if the deps or src has changed.
		if !depsChanged && !srcChanged {
			progressBar.Increment()
			continue
		} else {
			log.Debugf("Compiling file %s: depsChanged=%v, srcChanged=%v", srcFile, depsChanged, srcChanged)
		}

		// Build the compilation command.
		cmd := compileCommand(target, srcPath, objPath)

		// Run the command.
		nCompiled++
		taskQueue <- common.CmdSpec{cmd, results, func(string, bool, time.Duration) {
			progressBar.Increment()
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

func linkObjects(target *Target, progressBar *progress.ProgressBar, taskQueue chan common.CmdSpec, objects []string, nCompiled int) (string, error) {
	// Throw and error if there are no source files and this isn't a library.
	if target.IsBinary() && (len(target.AllSrcs()) == 0 && len(target.Deps) == 0) {
		return "", errors.New(fmt.Sprintf("No source files/deps found for binary %s", target))
	}

	// First, work out what the name of the output is.
	var outputName string
	if target.IsLibrary() {
		outputName = libraryName(target.Spec.Name())
	} else if target.IsExecutable() {
		outputName = binaryName(target.Spec.Name())
	}

	// Work out the output filepath.
	// dependenciesChanged := target.IsExecutable() && target.DependenciesChanged()
	outputPath := filepath.Join(target.Spec.OutputPath(), outputName)
	// if nCompiled == 0 && common.FileExists(outputPath) && !dependenciesChanged {
	// 	progressBar.Increment()
	// 	target.Changed = false
	// 	return outputPath, nil
	// }

	// Make the error channel.
	// target.Changed = true
	result := make(chan error)

	// Now, we need to build up the command to run.
	cmd := linkCommand(target, objects, outputPath)

	// Run the command.
	taskQueue <- common.CmdSpec{cmd, result, func(string, bool, time.Duration) {
		progressBar.Increment()
	}}

	err := <-result
	if err != nil {
		return "", err
	}

	return outputPath, nil
}
