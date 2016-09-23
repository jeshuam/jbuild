package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitPathAndFileWithNormalPathReturnsCorrectParts(t *testing.T) {
	dir, file := splitPathAndFile("a+b+c+d+e", "+")
	assert.Equal(t, "a/b/c/d", dir)
	assert.Equal(t, "e", file)
}

func TestSplitPathAndFileWithPathWithSingleElementsReturnsEmptyDir(t *testing.T) {
	dir, file := splitPathAndFile("a", "/")
	assert.Equal(t, "", dir)
	assert.Equal(t, "a", file)
}

func TestMakeFileSpecWithFullyQualifiedPathThatExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }

	fileSpec := MakeFileSpec("//path/to/file.txt", "")
	require.NotNil(t, fileSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to"),
		fileSpec.Path())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to", "file.txt"),
		fileSpec.FilePath())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "out", "path", "to"),
		fileSpec.OutputPath())

	assert.Equal(t, "//path/to/file.txt", fileSpec.String())
	assert.Equal(t, "path/to", fileSpec.Dir())
	assert.Equal(t, "file.txt", fileSpec.File())
	assert.Equal(t, "file", fileSpec.Type())
}

func TestMakeFileSpecWithFullyQualifiedPathThatDoesNotExistsReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	common.FileExists = func(string) bool { return false }
	common.IsDir = func(string) bool { return false }

	fileSpec := MakeFileSpec("//path/to/file.txt", "")
	assert.Nil(t, fileSpec)
}

func TestMakeFileSpecWithFullyQualifiedPathThatIsDirReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return true }

	fileSpec := MakeFileSpec("//path/to/file.txt", "")
	assert.Nil(t, fileSpec)
}

func TestMakeFileSpecWithRelativePathThatExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	args.CurrentDir = filepath.Join(args.WorkspaceDir, "path")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }

	fileSpec := MakeFileSpec("to/file.txt", args.CurrentDir)
	require.NotNil(t, fileSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to"),
		fileSpec.Path())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to", "file.txt"),
		fileSpec.FilePath())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "out", "path", "to"),
		fileSpec.OutputPath())

	assert.Equal(t, "//path/to/file.txt", fileSpec.String())
	assert.Equal(t, "path/to", fileSpec.Dir())
	assert.Equal(t, "file.txt", fileSpec.File())
	assert.Equal(t, "file", fileSpec.Type())
}

func TestMakeFileSpecWithRelativePathToRootThatExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	args.CurrentDir = filepath.Join(args.WorkspaceDir)
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }

	fileSpec := MakeFileSpec("file.txt", args.CurrentDir)
	require.NotNil(t, fileSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace"),
		fileSpec.Path())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "file.txt"),
		fileSpec.FilePath())

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "out"),
		fileSpec.OutputPath())

	assert.Equal(t, "//file.txt", fileSpec.String())
	assert.Equal(t, "", fileSpec.Dir())
	assert.Equal(t, "file.txt", fileSpec.File())
	assert.Equal(t, "file", fileSpec.Type())
}

func TestMakeFileSpecWithRelativePathThatDoesNotExistsReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	args.CurrentDir = filepath.Join(args.WorkspaceDir, "path", "to")
	common.FileExists = func(string) bool { return false }
	common.IsDir = func(string) bool { return false }

	fileSpec := MakeFileSpec("file.txt", args.CurrentDir)
	assert.Nil(t, fileSpec)
}

func TestMakeFileSpecWithRelativePathThatIsDirReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	args.OutputDir = filepath.Join(args.WorkspaceDir, "out")
	args.CurrentDir = filepath.Join(args.WorkspaceDir, "path", "to")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return true }

	fileSpec := MakeFileSpec("file.txt", args.CurrentDir)
	assert.Nil(t, fileSpec)
}

func TestMakeFileSpecGlobWithManyGlobsReturnsSomething(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }
	Glob = func(string) ([]string, error) {
		return []string{
			filepath.Join(args.WorkspaceDir, "a", "1.txt"),
			filepath.Join(args.WorkspaceDir, "a", "2.txt"),
			filepath.Join(args.WorkspaceDir, "b", "1.txt"),
			filepath.Join(args.WorkspaceDir, "b", "2.txt"),
		}, nil
	}

	specs := MakeFileSpecGlob("**/*.txt", args.WorkspaceDir)
	require.Len(t, specs, 4)
	require.NotNil(t, specs[0])
	require.NotNil(t, specs[1])
	require.NotNil(t, specs[2])
	require.NotNil(t, specs[3])
	assert.Equal(t, "//a/1.txt", specs[0].String())
	assert.Equal(t, "//a/2.txt", specs[1].String())
	assert.Equal(t, "//b/1.txt", specs[2].String())
	assert.Equal(t, "//b/2.txt", specs[3].String())
}

func TestMakeFileSpecGlobWithManyGlobsIgnoresNonFiles(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(path string) bool {
		return strings.HasSuffix(path, "1.txt")
	}

	Glob = func(string) ([]string, error) {
		return []string{
			filepath.Join(args.WorkspaceDir, "a", "1.txt"),
			filepath.Join(args.WorkspaceDir, "a", "2.txt"),
			filepath.Join(args.WorkspaceDir, "b", "1.txt"),
			filepath.Join(args.WorkspaceDir, "b", "2.txt"),
		}, nil
	}

	specs := MakeFileSpecGlob("**/*.txt", args.WorkspaceDir)
	require.Len(t, specs, 2)
	require.NotNil(t, specs[0])
	require.NotNil(t, specs[1])
	assert.Equal(t, "//a/2.txt", specs[0].String())
	assert.Equal(t, "//b/2.txt", specs[1].String())
}
