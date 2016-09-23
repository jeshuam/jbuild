// This package is a set of "functional" tests for the build system. Each
// directory here is a workspace which tests some part (or combination of parts)
// of the system. The main() function here is called as though it was being run
// from the command-line. These are NOT unit tests.
package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/jeshuam/jbuild/config/cc"
	"github.com/jeshuam/jbuild/jbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cwd, _ = os.Getwd()
)

func init() {
	flag.Parse()
}

func runBinary(binary string) (string, error) {
	var out bytes.Buffer

	cmd := exec.Command(binary)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func setupTest(testDir string) args.Args {
	args, _ := args.Load(filepath.Join(cwd, "test", testDir))
	args.ShowLog = true
	args.NoCache = true
	args.Threads = 1
	return args
}

func listOutputFiles(t *testing.T, args *args.Args, binaryName string) ([]string, string) {
	files, err := config.Glob(filepath.Join(args.OutputDir, "**", "*"))
	require.NoError(t, err)

	fileNames := make([]string, 0, len(files))
	binary := ""
	for _, filePath := range files {
		filePathRel, _ := filepath.Rel(args.OutputDir, filePath)
		fileNames = append(fileNames, filePathRel)
		if filePathRel == cc.BinaryName(binaryName) {
			binary = filePath
		}
	}

	return fileNames, binary
}

func jbuildClean(t *testing.T, args args.Args) {
	require.NoError(t, jbuild.JBuildRun(args, []string{"clean"}))
	assert.True(t, !common.FileExists(args.OutputDir))
}

func Test01SimpleCppBinary(t *testing.T) {
	// Set the current directory.
	args := setupTest("01_simple_cpp_binary")

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 2)
	assert.Contains(t, fileNames, "main.cc.o")
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test02SimpleCppLibrary(t *testing.T) {
	// Set the current directory.
	args := setupTest("02_simple_cpp_library")

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 4)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, cc.LibraryName("lib"))
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test03CppMultilibrary(t *testing.T) {
	// Set the current directory.
	args := setupTest("03_cpp_multilibrary")

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 6)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, "lib2.cc.o")
	assert.Contains(t, fileNames, cc.LibraryName("lib"))
	assert.Contains(t, fileNames, cc.LibraryName("lib2"))
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}
