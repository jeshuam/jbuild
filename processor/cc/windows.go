package cc

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// This file contains a bunch of functions which are needed to compile with the
// Visual Studio compiler (cl.exe). This compiler works in some very... obscure
// ways, especially with regards to it's flag structure. As such, it should be
// treated very differently to normal compilers (i.e. g++). cl.exe works best
// when everything is defined as environment variables, so rather do that.

var (
	vcVersion = flag.String("vc_version", "14.0", "The Visual Studio version to use.")

	vcInstallDir, ucrtSdkDir, ucrtSdkVersion string
)

func windowsReadRegistryKey(key, name string) (string, error) {
	// Load the key.
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}

	// Get the value.
	val, _, err := k.GetStringValue(name)
	if err != nil {
		return "", err
	}

	return val, nil
}

func windowsLoadSdkDir() {
	// Load the Windows SDK directory from the registry.
	val, err := windowsReadRegistryKey(`SOFTWARE\Microsoft\VisualStudio\SxS\VC7`, *vcVersion)
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Wow6432Node\Microsoft\VisualStudio\SxS\VC7`, *vcVersion)
		if err != nil {
			log.Fatal("Could not find Visual Studio install directory.")
		}

		vcInstallDir = val
	}

	// Load the UCRT SDK directory from the registry.
	val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\Windows Kits\Installed Roots`, "KitsRoot10")
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Wow6432Node\Microsoft\Windows Kits\Installed Roots`, "KitsRoot10")
		if err != nil {
			log.Fatal("Could not find UCRT SDK directory.")
		}
	}

	// Find any version within this directory.
	versions, err := filepath.Glob(filepath.Join(val, "Include", "*"))
	if err != nil || len(versions) == 0 {
		log.Fatal("Could not find UCRT SDK directory.")
	}

	// Find the version which has all of the required directories.
	ucrtSdkDir = val
	for _, version := range versions {
		files, _ := ioutil.ReadDir(version)
		if len(files) >= 2 {
			ucrtSdkVersion = filepath.Base(version)
			break
		}
	}
}

func windowsPrepareClCommand(cmd *exec.Cmd) {
	env := os.Environ()

	// Set PATH.
	env = append(env, fmt.Sprintf(
		"PATH=%s",
		filepath.Join(vcInstallDir, "bin")))

	// Set INCLUDE.
	env = append(env, fmt.Sprintf(
		"INCLUDE=%s;%s",
		filepath.Join(vcInstallDir, "include"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "ucrt")))

	// Set LIBDIR.
	env = append(env, fmt.Sprintf(
		"LIB=%s;%s;%s;%s",
		filepath.Join(vcInstallDir, "lib"),
		filepath.Join(vcInstallDir, "lib", "amd64"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "ucrt", "x86"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "um", "x86")))

	env = append(env, fmt.Sprintf(
		"LIBDIR=%s;%s;%s;%s",
		filepath.Join(vcInstallDir, "lib"),
		filepath.Join(vcInstallDir, "lib", "amd64"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "ucrt", "x86"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "um", "x86")))

	log.Info(ucrtSdkDir, ucrtSdkVersion)

	cmd.Env = env
}

// Build a command which can be used to compile a windows source file into an
// object. This will not include any additional flags required.
func windowsClCompileCommand(src, obj string) *exec.Cmd {
	command := exec.Command("cl.exe", "/c", "/Fo"+obj, src)

	// Add the required environment variables.
	windowsPrepareClCommand(command)

	return command
}

func windowsClLinkCommand(objs, libs []string, output string) *exec.Cmd {
	// Work out the linker to use. This will change depending on the desired
	// output file.
	var linker string
	if strings.HasSuffix(output, ".lib") {
		linker = "lib.exe"
	} else {
		linker = "link.exe"
	}

	// Make the command.
	cmd := exec.Command(linker, "/OUT:"+output)
	cmd.Args = append(cmd.Args, objs...)
	cmd.Args = append(cmd.Args, libs...)
	windowsPrepareClCommand(cmd)

	return cmd
}

func windowsLibraryName(name string) string {
	if *ccStaticLinking {
		return name + ".lib"
	}

	return name + ".dll"
}
