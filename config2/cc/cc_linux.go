package cc

import (
	"os/exec"
	"strings"
)

func prepareEnvironment(_ *Target, cmd *exec.Cmd) {

}

func libraryName(name string) string {
	if *ccStaticLinking {
		return name + ".a"
	}

	return "lib" + name + ".so"
}

func isSharedLib(path string) bool {
	return strings.HasSuffix(path, ".so")
}

func binaryName(name string) string {
	return name
}
