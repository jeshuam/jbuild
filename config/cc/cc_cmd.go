package cc

import (
	"os/exec"
	"path/filepath"
	"runtime"
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
			"-fcolor-diagnostics",
			"-c", "-o", obj, src}...)
	}

	// Add the OS as a #define, which could be useful.
	flags = append(flags, "-DOS_"+strings.ToUpper(runtime.GOOS))

	// Build up the command line. This varies depending on the compiler type
	// (mainly because cl.exe is really weird).
	flags = append(flags, target.compileFlags()...)
	for _, include := range target.includes() {
		if compiler == "cl.exe" {
			flags = append(flags, "/I"+filepath.Join(args.GenOutputDir, include.Dir()))
			flags = append(flags, "/I"+filepath.Join(include.FsPath()))
		} else {
			flags = append(flags, "-I"+filepath.Join(args.GenOutputDir, include.Dir()))
			flags = append(flags, "-I"+filepath.Join(include.FsPath()))
		}
	}

	if compiler == "cl.exe" {
		flags = append(flags, "/I"+args.WorkspaceDir)
		flags = append(flags, "/I"+args.ExternalRepoDir)
		flags = append(flags, "/I"+args.GenOutputDir)
	} else {
		flags = append(flags, "-I"+args.WorkspaceDir)
		flags = append(flags, "-I"+args.ExternalRepoDir)
		flags = append(flags, "-I"+args.GenOutputDir)
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
	if runtime.GOOS == "windows" {
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
		flags = []string{"/OUT:" + output, "msvcrt.lib"}
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
			flags = append(flags, lib.FsPath())
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
