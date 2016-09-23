package cc

import (
	"os/exec"
	"strings"
)

func prepareEnvironment(_ *Target, cmd *exec.Cmd) {

}

func LibraryName(name string) string {
	return name + ".a"
}

func isSharedLib(path string) bool {
	return strings.HasSuffix(path, ".so")
}

func BinaryName(name string) string {
	return name
}
