package cc

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/jeshuam/jbuild/config"
)

var (
	// Windows specific variables.
	vcVersion = flag.String("vc_version", "14.0", "The Visual Studio version to use.")

	vcInstallDir, ucrtSdkDir, ucrtSdkVersion, netFxSdkDir string
)

func init() {
	windowsLoadSdkDir()
}

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
	val, err := windowsReadRegistryKey(`SOFTWARE\Wow6432Node\Microsoft\VisualStudio\SxS\VC7`, *vcVersion)
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\VisualStudio\SxS\VC7`, *vcVersion)
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

	// Load the NETFXSDK directory from the registry.
	val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\Microsoft SDKs\Windows\v7.1A`, "InstallationFolder")
	if err != nil {
		val, err = windowsReadRegistryKey(`SOFTWARE\Microsoft\Microsoft SDKs\Windows\v7.1A`, "InstallationFolder")
		if err != nil {
			log.Fatal("Could not find NetFX SDK directory.")
		}
	}

	netFxSdkDir = val
}

func prepareEnvironment(target *config.Target, cmd *exec.Cmd) {
	env := os.Environ()

	// Set PATH.
	env = append(env, fmt.Sprintf(
		"PATH=%s",
		filepath.Join(vcInstallDir, "bin")))

	// Set INCLUDE.
	env = append(env, fmt.Sprintf(
		"INCLUDE=%s;%s;%s",
		filepath.Join(vcInstallDir, "include"),
		filepath.Join(ucrtSdkDir, "Include", ucrtSdkVersion, "ucrt"),
		filepath.Join(netFxSdkDir, "Include"),
		target.Spec.Workspace))

	// Set LIBDIR.
	env = append(env, fmt.Sprintf(
		"LIB=%s;%s;%s;%s;%s",
		filepath.Join(vcInstallDir, "lib"),
		filepath.Join(vcInstallDir, "lib", "amd64"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "ucrt", "x86"),
		filepath.Join(ucrtSdkDir, "Lib", ucrtSdkVersion, "um", "x86"),
		filepath.Join(netFxSdkDir, "Lib")))

	cmd.Env = env
}

func libraryName(name string) string {
	if *ccStaticLinking {
		return name + ".lib"
	}

	return name + ".dll"
}

func isSharedLib(path string) bool {
	return strings.HasSuffix(path, ".dll")
}

func binaryName(name string) string {
	return name + ".exe"
}
