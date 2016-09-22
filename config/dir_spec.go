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
	path string
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *DirSpecImpl) Dir() string {
	return this.path
}

func (this *DirSpecImpl) Path() string {
	return filepath.Join(
		args.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *DirSpecImpl) String() string {
	return "//" + this.Dir()
}

func (this *DirSpecImpl) Type() string {
	return "dir"
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// MakeDirSpec constructs and returns a valid DirSpec object, or nil if the
// given spec doesn't refer to a valid directory. rawSpec can be absolute or
// relative to `cwd`.
func MakeDirSpec(rawSpec, cwd string) interfaces.DirSpec {
	spec := new(DirSpecImpl)

	// If the spec is absolute, then we can just save the path directly.
	if !strings.HasPrefix(rawSpec, "//") {
		workspacePath, _ := filepath.Rel(args.WorkspaceDir, cwd)
		spec.path = filepath.Join(workspacePath, rawSpec)
	} else {
		spec.path = rawSpec
	}

	// Check to see whether this file exists and is a file. If it doesn't, then
	// we don't have a FileSpec.
	spec.path = strings.Trim(strings.Replace(spec.path, pathSeparator, "/", -1), "/")
	if common.FileExists(spec.Path()) && common.IsDir(spec.Path()) {
		return spec
	}

	return nil
}
