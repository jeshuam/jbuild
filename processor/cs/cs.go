package cs

import (
	"flag"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")

	csCompiler = flag.String("cs_compiler", "", "The C# compiler to use.")

	defaultCompilers = map[string]string{
		"windows": "C:\\Windows\\Microsoft.NET\\Framework\\v4.0.30319\\csc.exe",
	}
)

type CSProcessor struct {
}

func init() {
	// Set the default compiler.
	if *csCompiler == "" {
		compiler, ok := defaultCompilers[runtime.GOOS]
		if !ok {
			compiler = "mcs"
		}

		*csCompiler = compiler
	}
}

func buildCommand(target *config.Target) *exec.Cmd {
	// Work out the output name.
	var outputName string
	var flags []string
	if target.IsLibrary() {
		outputName = target.Spec.Name + ".netmodule"
		flags = []string{"/target:module"}
	} else {
		outputName = target.Spec.Name + ".exe"
		flags = []string{"/target:exe"}
	}

	// Load dependencies
	for _, dep := range target.AllDependencies() {
		for _, output := range dep.Output {
			flags = append(flags, "/addmodule:"+output)
		}
	}

	// Build the final command.
	target.Output = append(target.Output, filepath.Join(target.Spec.OutputPath(), outputName))
	flags = append(flags, "/out:"+target.Output[0])
	flags = append(flags, target.Srcs()...)
	return exec.Command(*csCompiler, flags...)
}

func (p CSProcessor) Process(target *config.Target, taskQueue chan common.CmdSpec) error {
	// Compile the source files into a
	target.ProgressBar.SetOperation("building")
	results := make(chan error, 1)
	cmd := buildCommand(target)
	taskQueue <- common.CmdSpec{cmd, results, func(error) {
		target.ProgressBar.Increment()
	}}

	result := <-results
	target.ProgressBar.Finish()
	return result
}
