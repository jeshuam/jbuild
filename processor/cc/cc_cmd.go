package cc

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
)

func compileCommand(target *config.Target, src, obj string) *exec.Cmd {
	compiler := *ccCompiler

	// Build up the command line. This varies depending on the compiler type
	// (mainly because cl.exe is really weird).
	flags := target.CompileFlags()
	for _, include := range target.Includes() {
		includePath := filepath.Join(target.Spec.WorkspacePath(), include)
		flags = append(flags, "-I"+includePath)
	}

	// Include flags from dependencies.
	for _, dep := range target.AllDependencies() {
		if strings.HasPrefix(dep.Type, "c++") {
			flags = append(flags, dep.CompileFlags()...)

			for _, include := range dep.Includes() {
				includePath := filepath.Join(dep.Spec.WorkspacePath(), include)
				flags = append(flags, "-I"+includePath)
			}
		} else if strings.HasPrefix(dep.Type, "genrule") {

		} else {
			log.Fatalf("Invalid dependency from C++ --> %s", dep.Type)
		}
	}

	// Add compiler specific options.
	if compiler == "cl.exe" {
		flags = append(flags, []string{"/c", "/Fo" + obj, src, "/EHsc"}...)
	} else {
		flags = append(flags, []string{
			"-I" + common.WorkspaceDir,
			"-I" + filepath.Join(common.OutputDirectory, "gen"),
			"-I/usr/include",
			"-fPIC",
			"-fcolor-diagnostics",
			"-c", "-o", obj, src}...)
	}

	// Make the command.
	command := exec.Command(compiler, flags...)

	// Prepare the command's environment. This will do different things depending
	// on whether this is windows or linux.
	prepareEnvironment(target, command)

	// Return the complete command.
	return command
}

func linkCommand(target *config.Target, objs []string, output string) *exec.Cmd {
	// Work out which linker to use.
	var linker string
	if *ccCompiler == "cl.exe" {
		if strings.HasSuffix(output, ".lib") {
			linker = "lib.exe"
		} else {
			linker = "link.exe"
		}
	} else {
		if *ccStaticLinking && target.IsLibrary() {
			linker = "ar"
		} else {
			linker = *ccCompiler
		}
	}

	// Make the flags.
	flags := []string{}
	if linker == "lib.exe" || linker == "link.exe" {
		flags = []string{"/OUT:" + output}
	} else if linker == "ar" {
		flags = []string{"cr", output}
	} else {
		if target.IsLibrary() {
			flags = append(flags, "-shared")
		}

		flags = append(flags, []string{"-o", output}...)
	}

	// Add the objects to the commandline.
	flags = append(flags, objs...)
	if target.IsExecutable() {
		for _, lib := range target.Libs() {
			flags = append(flags, lib)
		}
	}

	// Link in libraries for binaries.
	if target.IsExecutable() {
		extraFlags := make(map[string]bool, 0)
		for _, dep := range target.AllDependencies() {
			for _, flag := range dep.LinkFlags() {
				extraFlags[flag] = true
			}
		}

		for _, flag := range target.LinkFlags() {
			extraFlags[flag] = true
		}

		// Add the extra flags.
		for flag := range extraFlags {
			flags = append(flags, flag)
		}

		// Add libs. This has to be done in order.
		for _, lib := range target.LibsOrdered() {
			flags = append(flags, lib)
		}

		// We have to go through the outputs in reverse order to make sure that we
		// put core dependencies last in the list.
		outputUsed := make(map[string]bool, 0)
		outputOrderedDuplicates := target.OutputOrdered()
		outputOrderedFiltered := make([]string, 0)
		for i := len(outputOrderedDuplicates) - 1; i >= 0; i-- {
			output := outputOrderedDuplicates[i]
			_, ok := outputUsed[output]
			if !ok {
				outputOrderedFiltered = append([]string{output}, outputOrderedFiltered...)
				outputUsed[output] = true
			}
		}

		flags = append(flags, outputOrderedFiltered...)
	}

	// Make the command.
	command := exec.Command(linker, flags...)

	// Prepare the environment.
	prepareEnvironment(target, command)

	return command
}
