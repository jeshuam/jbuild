package config

import (
	"path/filepath"
	"testing"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeDirSpecWithFullyQualifiedDirThatExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return true }

	dirSpec := MakeDirSpec("//path/to/dir", "")
	require.NotNil(t, dirSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to", "dir"),
		dirSpec.Path())

	assert.Equal(t, "//path/to/dir", dirSpec.String())
	assert.Equal(t, "path/to/dir", dirSpec.Dir())
	assert.Equal(t, "dir", dirSpec.Type())
}

func TestMakeDirSpecWithFullyQualifiedDirDoesNotExistReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return false }
	common.IsDir = func(string) bool { return true }

	dirSpec := MakeDirSpec("//path/to/dir", "")
	assert.Nil(t, dirSpec)
}

func TestMakeDirSpecWithFullyQualifiedDirExistsButIsNotDirReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }

	dirSpec := MakeDirSpec("//path/to/dir", "")
	assert.Nil(t, dirSpec)
}

func TestMakeDirSpecWithRelativeDirThatsExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return true }

	dirSpec := MakeDirSpec("dir", filepath.Join(args.WorkspaceDir, "path", "to"))
	require.NotNil(t, dirSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "path", "to", "dir"),
		dirSpec.Path())

	assert.Equal(t, "//path/to/dir", dirSpec.String())
	assert.Equal(t, "path/to/dir", dirSpec.Dir())
	assert.Equal(t, "dir", dirSpec.Type())
}

func TestMakeDirSpecWithRelativeDirInRootThatsExistsReturnsSpec(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return true }

	dirSpec := MakeDirSpec("dir", filepath.Join(args.WorkspaceDir))
	require.NotNil(t, dirSpec)

	assert.Equal(t,
		filepath.Join("path", "to", "workspace", "dir"),
		dirSpec.Path())

	assert.Equal(t, "//dir", dirSpec.String())
	assert.Equal(t, "dir", dirSpec.Dir())
	assert.Equal(t, "dir", dirSpec.Type())
}

func TestMakeDirSpecWithRelativeDirThatDoesNotExistReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return false }
	common.IsDir = func(string) bool { return true }

	dirSpec := MakeDirSpec("dir", filepath.Join(args.WorkspaceDir, "path", "to"))
	assert.Nil(t, dirSpec)
}

func TestMakeDirSpecWithRelativeDirThatIsNotDirReturnsNil(t *testing.T) {
	args.WorkspaceDir = filepath.Join("path", "to", "workspace")
	common.FileExists = func(string) bool { return true }
	common.IsDir = func(string) bool { return false }

	dirSpec := MakeDirSpec("dir", filepath.Join(args.WorkspaceDir, "path", "to"))
	assert.Nil(t, dirSpec)
}
