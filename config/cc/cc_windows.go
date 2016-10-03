package cc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"golang.org/x/sys/windows/registry"
)

var (
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

func windowsLoadSdkDir(args *args.Args) {
	// Load the Windows SDK directory from the registry.
	val, err := windowsReadRegistryKey(`SOFTWARE\Wow6432Node\Microsoft\VisualStudio\SxS\VC7`, args.VCVersion)
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\VisualStudio\SxS\VC7`, args.VCVersion)
		if err != nil {
			log.Fatal("Could not find Visual Studio install directory.")
		}
	}

	vcInstallDir = val

	// Load the UCRT SDK directory from the registry.
	val, err = windowsReadRegistryKey(`SOFTWARE\Wow6432Node\Microsoft\Windows Kits\Installed Roots`, "KitsRoot10")
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\Windows Kits\Installed Roots`, "KitsRoot10")
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
	ucrtSdkVersion = filepath.Base(versions[len(versions)-1])
}

func prepareEnvironment(args *args.Args, target *Target, cmd *exec.Cmd) {
	// If we haven't loaded yet, then load.
	if vcInstallDir == "" {
		windowsLoadSdkDir(args)
	}

	env := os.Environ()

	// Set PATH.
	env = append(env, fmt.Sprintf(
		"PATH=%s",
		filepath.Join(vcInstallDir, "bin")))

	// Set INCLUDE.
	env = append(env, fmt.Sprintf(
		"INCLUDE=%s;%s;%s;%s;%s;%s;%s;%s",
		filepath.Join(vcInstallDir, "include"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "ucrt"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "um"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "shared"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "winrt"),
		filepath.Join(ucrtSdkDir, "Include", "um"),
		args.WorkspaceDir,
		filepath.Join(args.OutputDir, "gen")))

	// Set LIBDIR.
	env = append(env, fmt.Sprintf(
		"LIB=%s;%s;%s;%s;%s",
		filepath.Join(vcInstallDir, "lib"),
		filepath.Join(vcInstallDir, "lib", "amd64"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "ucrt", "x86"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "um", "x86")))

	cmd.Env = env
	cmd.Path = filepath.Join(vcInstallDir, "bin", cmd.Args[0])
}

func LibraryName(name string) string {
	return name + ".lib"
}

func isSharedLib(path string) bool {
	return strings.HasSuffix(path, ".dll")
}

func BinaryName(name string) string {
	return name + ".exe"
}
