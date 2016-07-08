package cc

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")

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
			compiler = "clang"
		}

		*ccCompiler = compiler
	}

	// If we are on windows and the compiler is cl.exe, then we need to load some
	// stuff from the registry.
	if runtime.GOOS == "windows" && *ccCompiler == "cl.exe" {
		windowsLoadSdkDir()
	}
}

func runCommand(cmd *exec.Cmd) error {
	// Print the command.
	if common.DryRun {
		log.Infof("DRY_RUN: %s", cmd.Args)
		return nil
	} else {
		log.Debug(cmd.Args)
	}

	// Save the command output.
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command.
	err := cmd.Run()
	if err != nil {
		return errors.New(out.String())
	}

	return nil
}

// Compile the source files within the given target.
func compileFiles(target *config.Target) ([]string, error) {
	objs := make([]string, len(target.Srcs))

	// Use a different command maker based on the OS and compiler.
	var compileCommand func(*config.Target, string, string) *exec.Cmd
	if *ccCompiler == "cl.exe" {
		compileCommand = windowsClCompileCommand
	} else {
		return nil, errors.New(fmt.Sprintf("Unsupported compiler %s", *ccCompiler))
	}

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
		cmd.Args = append(cmd.Args, target.CompileFlags...)

		// Run the command.
		err := runCommand(cmd)
		if err != nil {
			return nil, err
		}
	}

	return objs, nil
}

func linkObjects(target *config.Target, objects []string) (string, error) {
	// First, work out what the name of the output is.
	outputName := target.Spec.Name
	if target.Type == "c++/library" {
		if runtime.GOOS == "windows" {
			outputName = windowsLibraryName(outputName)
		} else {
			return "", errors.New("System not yet implemented.")
		}
	} else if target.Type == "c++/binary" {
		if runtime.GOOS == "windows" {
			outputName = outputName + ".exe"
		}
	}

	// Work out the output filepath.
	outputPath := filepath.Join(target.Spec.OutputPath(), outputName)

	// Combine all of this target's dependencies outputs into a single list of
	// paths.
	libs := make([]string, 0)
	for _, dep := range target.Deps {
		libs = append(libs, dep.Output...)
	}

	// Now, we need to build up the command to run.
	var cmd *exec.Cmd
	if *ccCompiler == "cl.exe" {
		cmd = windowsClLinkCommand(target, objects, libs, outputPath)
	} else {
		return "", errors.New("System not yet implemented.")
	}

	// Run the command.
	err := runCommand(cmd)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func (p CCProcessor) Process(target *config.Target) error {
	// Make the output directory for this target.
	err := os.MkdirAll(target.Spec.OutputPath(), os.ModeDir)
	if err != nil {
		return err
	}

	// Compile all of the source files.
	objFiles, err := compileFiles(target)
	if err != nil {
		return err
	}

	// Link all object files into a binary. What this binary is depends on the
	// type of the target.
	binary, err := linkObjects(target, objFiles)
	if err != nil {
		return err
	}

	// Save the output of this processing command.
	target.Output = append(target.Output, binary)

	// All finished, with no errors!
	return nil
}
