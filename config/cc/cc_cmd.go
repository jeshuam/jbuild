package cc

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/config/util"
)

func compileCommand(target *Target, src, obj string) *exec.Cmd {
	compiler := args.CCCompiler

	// Add compiler specific options.
	flags := make([]string, 0)
	if compiler == "cl.exe" {
		flags = append(flags, []string{"/c", "/Fo" + obj, src, "/EHsc"}...)
	} else {
		flags = append(flags, []string{
			"-I" + args.WorkspaceDir,
			"-I" + filepath.Join(args.OutputDir, "gen"),
			"-I/usr/include",
			"-fPIC",
			"-fcolor-diagnostics",
			"-c", "-o", obj, src}...)
	}

	// Build up the command line. This varies depending on the compiler type
	// (mainly because cl.exe is really weird).
	flags = append(flags, target.compileFlags()...)
	for _, include := range target.includes() {
		if compiler == "cl.exe" {
			flags = append(flags, "/I"+include.Path())
		} else {
			flags = append(flags, "-I"+include.Path())
		}

	}

	// Make the command.
	command := exec.Command(compiler, util.MakeUnique(flags)...)

	// Prepare the command's environment. This will do different things depending
	// on whether this is windows or linux.
	prepareEnvironment(target, command)

	// Return the complete command.
	return command
}

func linkCommand(target *Target, objs []string, output string) *exec.Cmd {
	// Work out which linker to use.
	var linker string
	if args.CCCompiler == "cl.exe" {
		if strings.HasSuffix(output, ".lib") {
			linker = "lib.exe"
		} else {
			linker = "link.exe"
		}
	} else {
		if target.IsExecutable() {
			linker = "clang++"
		} else {
			linker = "ar"
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

	// Add the objects to the command-line.
	flags = append(flags, objs...)

	// Link in libraries for binaries.
	if target.IsExecutable() {
		// Add the extra flags.
		for _, flag := range target.linkFlags() {
			flags = append(flags, flag)
		}

		// Add static libs.
		for _, lib := range target.libs() {
			flags = append(flags, lib.FilePath())
		}

		// We have to go through the outputs in reverse order to make sure that we
		// put core dependencies last in the list.
		depOutputs := target.depOutputs()
		for i := len(depOutputs) - 1; i >= 0; i-- {
			flags = append(flags, depOutputs[i])
		}
	}

	// Make the command.
	command := exec.Command(linker, util.MakeUnique(flags)...)

	// Prepare the environment.
	prepareEnvironment(target, command)

	return command
}
