// This package is a set of "functional" tests for the build system. Each
// directory here is a workspace which tests some part (or combination of parts)
// of the system. The main() function here is called as though it was being run
// from the command-line. These are NOT unit tests.
package main

import (
	"bytes"
	"flag"
	"os/exec"
	"path/filepath"
	"runtime"
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
	cwd string
)

func init() {
	flag.Parse()

	_, filename, _, _ := runtime.Caller(1)
	cwd = filepath.Dir(filename)
}

func runBinary(binary string) (string, error) {
	var out bytes.Buffer

	cmd := exec.Command(binary)
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = filepath.Dir(binary)
	err := cmd.Run()
	return out.String(), err
}

func setupTest(t *testing.T, testDir string, baseArgs *args.Args) args.Args {
	args, err := args.Load(filepath.Join(cwd, "test", testDir), baseArgs)
	require.NoError(t, err)

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
		if !common.IsDir(filePath) {
			filePathRel, _ := filepath.Rel(args.OutputDir, filePath)
			fileNames = append(fileNames, filePathRel)
			if filePathRel == cc.BinaryName(binaryName) {
				binary = filePath
			}
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
	args := setupTest(t, "01_simple_cpp_binary", nil)

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
	args := setupTest(t, "02_simple_cpp_library", nil)

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
	args := setupTest(t, "03_cpp_multilibrary", nil)

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

func Test04CppData(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "04_cpp_data", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 5)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, "data.txt")
	assert.Contains(t, fileNames, cc.LibraryName("lib"))
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test05CppAllTargets(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "05_cpp_all_target", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":all"}))

	// Make sure the output is valid.
	fileNames, _ := listOutputFiles(t, &args, "")
	require.Len(t, fileNames, 4)
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, "lib2.cc.o")
	assert.Contains(t, fileNames, cc.LibraryName("lib"))
	assert.Contains(t, fileNames, cc.LibraryName("lib2"))

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test06CppAllTargetsInTree(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "06_cpp_all_targets_in_tree", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", "//..."}))

	// Make sure the output is valid.
	fileNames, _ := listOutputFiles(t, &args, "")
	require.Len(t, fileNames, 4)
	assert.Contains(t, fileNames, filepath.Join("lib", "lib.cc.o"))
	assert.Contains(t, fileNames, filepath.Join("lib2", "lib2.cc.o"))
	assert.Contains(t, fileNames, filepath.Join("lib", cc.LibraryName("lib")))
	assert.Contains(t, fileNames, filepath.Join("lib2", cc.LibraryName("lib2")))

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test07CppFilegroup(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "07_cpp_filegroup", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 3)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test08CppFilegroupOfFilegroups(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "08_cpp_filegroup_of_filegroups", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 3)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, "lib.cc.o")
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test09CppIncludes(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "09_cpp_includes", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 2)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test10CppDepWithHeaders(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "10_cpp_dep_with_headers", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 2)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test11CppGlobs(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, "11_cpp_globs", nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 3)
	assert.Contains(t, fileNames, filepath.Join("dir1", "main.cc.o"))
	assert.Contains(t, fileNames, filepath.Join("dir1", "dir2", "lib.cc.o"))
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test12CppRunFromDifferentDir(t *testing.T) {
	// Set the current directory.
	args := setupTest(t, filepath.Join("12_cpp_run_from_different_dir", "dir1", "dir2"), nil)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", "../..:hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 3)
	assert.Contains(t, fileNames, filepath.Join("dir1", "main.cc.o"))
	assert.Contains(t, fileNames, filepath.Join("dir1", "dir2", "lib.cc.o"))
	assert.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test13BaseWorkspaceFiles(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.BaseWorkspaceFiles = "base_workspace_files"
	args := setupTest(t, filepath.Join("13_base_workspace_files"), &defaultArgs)

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

func Test14MergeWorkspaceFiles(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.BaseWorkspaceFiles = "base_workspace_files"
	args := setupTest(t, filepath.Join("14_merge_workspace_files"), &defaultArgs)

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

func Test15MultipleConfigurations(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.Configuration = "debug"
	args := setupTest(t, filepath.Join("15_multiple_configurations"), &defaultArgs)

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

func Test16ExternalLibrary(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.CleanExternalRepos = true
	args := setupTest(t, filepath.Join("16_external_library"), &defaultArgs)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 10)
	assert.Contains(t, fileNames, "main.cc.o")
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test17ExternalLibraryWithBUILDFile(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.CleanExternalRepos = true
	args := setupTest(t, filepath.Join("17_external_library_with_build_file"), &defaultArgs)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 10)
	assert.Contains(t, fileNames, "main.cc.o")
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test18MultistepGenrules(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	args := setupTest(t, filepath.Join("18_multistep_genrules"), &defaultArgs)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.Len(t, fileNames, 8)
	assert.Contains(t, fileNames, "main.cc.o")
	assert.Contains(t, fileNames, filepath.Join("gen", "pa.cc"))
	assert.Contains(t, fileNames, filepath.Join("gen", "ss.cc"))
	assert.Contains(t, fileNames, filepath.Join("gen", "ed.cc"))
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}

func Test19ExternalLibraryWithIncludedBUILDFile(t *testing.T) {
	// Set the current directory.
	defaultArgs := args.DefaultArgs()
	defaultArgs.CleanExternalRepos = true
	args := setupTest(t, filepath.Join("19_external_library_with_included_workspace"), &defaultArgs)

	// Build up the command-line.
	require.NoError(t, jbuild.JBuildRun(args, []string{"build", ":hello_world"}))

	// Make sure the output is valid.
	fileNames, binary := listOutputFiles(t, &args, "hello_world")
	require.True(t, len(fileNames) >= 16)
	assert.Contains(t, fileNames, "main.cc.o")
	require.Contains(t, fileNames, cc.BinaryName("hello_world"))

	// Run the binary and get the output.
	output, err := runBinary(binary)
	require.NoError(t, err)
	assert.Equal(t, "PASSED", output)

	// Now, cleanup the output directory.
	jbuildClean(t, args)
}
