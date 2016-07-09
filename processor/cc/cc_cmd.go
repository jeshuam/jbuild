package cc

import (
	"os/exec"
	"strings"

	"github.com/jeshuam/jbuild/config"
)

func compileCommand(target *config.Target, src, obj string) *exec.Cmd {
	compiler := *ccCompiler

	// Build up the command line. This varies depending on the compiler type
	// (mainly because cl.exe is really weird).
	var flags []string
	if compiler == "cl.exe" {
		flags = append(flags, []string{"/c", "/Fo" + obj, src}...)
	} else {
		flags = append(flags, []string{
			"-I" + target.Spec.Workspace,
			"-fPIC",
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
		if *ccStaticLinking && target.Type == "c++/library" {
			linker = "ar"
		} else {
			linker = *ccCompiler
		}
	}

	// Get a list of libraries to include from the target.
	libs := []string{}
	for _, dep := range target.Deps {
		libs = append(libs, dep.Output...)
	}

	// Make the flags.
	flags := []string{}
	if linker == "lib.exe" || linker == "link.exe" {
		flags = []string{"/OUT:" + output}
	} else if linker == "ar" {
		flags = []string{"cr", output}
	} else {
		if target.Type == "c++/library" {
			flags = append(flags, "-shared")
		}

		flags = append(flags, []string{"-o", output}...)
	}

	// Add the objects to the commandline.
	flags = append(flags, objs...)
	flags = append(flags, libs...)

	// Make the command.
	command := exec.Command(linker, flags...)

	// Prepare the environment.
	prepareEnvironment(target, command)

	return command
}
