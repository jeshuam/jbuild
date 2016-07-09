package cc

import (
	"flag"
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
	objs := make([]string, len(target.Srcs))
	results := make(chan error, len(target.Srcs))
	nCompiled := 0

	for i, srcFile := range target.Srcs {
		// Work out the full path to the source file. This will need to be provided
		// to the compiler.
		srcPath := filepath.Join(target.Spec.Workspace, target.Spec.PathSystem(), srcFile)
		objPath := filepath.Join(target.Spec.OutputPath(), srcFile+".o")
		objs[i] = objPath

		// If the object is newer than the source file, don't compile it again.
		srcStat, _ := os.Stat(srcPath)
		objStat, _ := os.Stat(objPath)
		if srcStat != nil && objStat != nil && objStat.ModTime().After(srcStat.ModTime()) {
			continue
		}

		// Build the compilation command.
		cmd := compileCommand(target, srcPath, objPath)

		// Run the command.
		nCompiled++
		taskQueue <- common.CmdSpec{cmd, results}
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
	// First, work out what the name of the output is.
	var outputName string
	if target.Type == "c++/library" {
		outputName = libraryName(target.Spec.Name)
	} else if target.Type == "c++/binary" {
		outputName = binaryName(target.Spec.Name)
	}

	// Work out the output filepath.
	outputPath := filepath.Join(target.Spec.OutputPath(), outputName)
	if nCompiled == 0 && common.FileExists(outputPath) {
		return outputPath, nil
	}

	// Make the error channel.
	result := make(chan error)

	// Combine all of this target's dependencies outputs into a single list of
	// paths.
	libs := make([]string, 0)
	for _, dep := range target.Deps {
		libs = append(libs, dep.Output...)
	}

	// Now, we need to build up the command to run.
	cmd := linkCommand(target, objects, outputPath)

	// Run the command.
	taskQueue <- common.CmdSpec{cmd, result}
	err := <-result
	if err != nil {
		return "", nil
	}

	return outputPath, nil
}

func (p CCProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	// Make the output directory for this target.
	err := os.MkdirAll(target.Spec.OutputPath(), 0755)
	if err != nil {
		return err
	}

	// Compile all of the source files.
	objFiles, nCompiled, err := compileFiles(target, taskQueue)
	if err != nil {
		return err
	}

	// Link all object files into a binary. What this binary is depends on the
	// type of the target. We only have to do that if something in the target was
	// compiled (this should avoid expensive and pointless linking steps).
	binary, err := linkObjects(target, taskQueue, objFiles, nCompiled)
	if err != nil {
		return err
	}

	// Save the output of this processing command.
	target.Output = append(target.Output, binary)

	// All finished, with no errors!
	return nil
}
