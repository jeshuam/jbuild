package cc

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/config/util"
)

func compileCommand(args *args.Args, target *Target, src, obj string) *exec.Cmd {
	compiler := args.CCCompiler

	// Add compiler specific options.
	flags := make([]string, 0)
	if compiler == "cl.exe" {
		flags = append(flags, []string{"/c", "/Fo" + obj, src, "/EHsc"}...)
	} else {
		flags = append(flags, []string{
			"-I" + args.WorkspaceDir,
			"-I" + filepath.Join(args.OutputDir, "gen"),
			"-I" + filepath.Join(args.OutputDir, "gen", target.Spec.Dir()),
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
			flags = append(flags, "/I"+filepath.Join(args.OutputDir, "gen", include.WorkspacePath()))
		} else {
			flags = append(flags, "-I"+include.Path())
			flags = append(flags, "-I"+filepath.Join(args.OutputDir, "gen", include.WorkspacePath()))
		}
	}

	// Make the command.
	command := exec.Command(compiler, util.MakeUnique(flags)...)

	// Prepare the command's environment. This will do different things depending
	// on whether this is windows or linux.
	prepareEnvironment(args, target, command)

	// Return the complete command.
	return command
}

func linkCommand(args *args.Args, target *Target, objs []string, output string) *exec.Cmd {
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

		// Add the previous outputs to the commandline.
		for _, output := range target.depOutputs() {
			flags = append(flags, output)
		}
	}

	// Make the command.
	command := exec.Command(linker, util.MakeUnique(flags)...)

	// Prepare the environment.
	prepareEnvironment(args, target, command)

	return command
}
