package cc

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
)

func compileCommand(target *Target, src, obj string) *exec.Cmd {
	compiler := args.CCCompiler

	// Build up the command line. This varies depending on the compiler type
	// (mainly because cl.exe is really weird).
	flags := target.compileFlags()
	for _, include := range target.includes() {
		flags = append(flags, "-I"+include.Path())
	}

	// Add compiler specific options.
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

	// Make the command.
	command := exec.Command(compiler, flags...)

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
		linker = "ar"
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
		// Save the flags to a map to ensure we don't have duplicates. Order should
		// not matter in this case.
		extraFlags := make(map[string]bool, 0)
		for _, flag := range target.linkFlags() {
			extraFlags[flag] = true
		}

		// Add the extra flags.
		for flag := range extraFlags {
			flags = append(flags, flag)
		}

		// Add static libs.
		for _, lib := range target.libs() {
			flags = append(flags, lib.FilePath())
		}

		// We have to go through the outputs in reverse order to make sure that we
		// put core dependencies last in the list.
		outputUsed := make(map[string]bool, 0)
		outputOrderedDuplicates := target.depOutputs()
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
