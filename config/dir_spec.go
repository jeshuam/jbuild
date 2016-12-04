package config

import (
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/interfaces"
)

// Implementation of the FileSpec interface.
type DirSpecImpl struct {
	// The location of the file within the workspace.
	dir string

	// The path to the root of the workspace.
	workspacePath string

	args *args.Args
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *DirSpecImpl) Dir() string {
	return this.dir
}

func (this *DirSpecImpl) Path() string {
	return this.FsPath()
}

func (this *DirSpecImpl) String() string {
	return "//" + this.Dir()
}

func (this *DirSpecImpl) Type() string {
	return "dir"
}

func (this *DirSpecImpl) FsWorkspacePath() string {
	return this.workspacePath
}

func (this *DirSpecImpl) FsOutputPath() string {
	return filepath.Join(this.args.OutputDir, this.Dir())
}

func (this *DirSpecImpl) FsPath() string {
	return filepath.Join(this.FsWorkspacePath(), this.Dir())
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// MakeDirSpec constructs and returns a valid DirSpec object, or nil if the
// given spec doesn't refer to a valid directory. rawSpec can be absolute or
// relative to `cwd`.
func MakeDirSpec(args *args.Args, rawSpec, cwd, buildBase string) interfaces.DirSpec {
	spec := new(DirSpecImpl)
	spec.args = args

	// If the spec is absolute, then we can just save the path directly.
	spec.workspacePath = buildBase
	if strings.HasPrefix(rawSpec, "//") {
		spec.dir = strings.Trim(rawSpec, "/")
	} else {
		spec.dir = filepath.Clean(filepath.Join(cwd, rawSpec))
	}

	// Check to see whether this file exists and is a file. If it doesn't, then
	// we don't have a FileSpec.
	if common.FileExists(spec.FsPath()) && common.IsDir(spec.FsPath()) {
		return spec
	}

	return nil
}
