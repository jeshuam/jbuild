package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/interfaces"
)

// Implementation of the FileSpec interface.
type DirSpecImpl struct {
	path string

	args *args.Args
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *DirSpecImpl) Dir() string {
	return this.path
}

func (this *DirSpecImpl) Path() string {
	return filepath.Join(strings.Replace(this.path, "/", pathSeparator, -1))
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
func MakeDirSpec(args *args.Args, rawSpec, cwd, buildBase string) interfaces.DirSpec {
	spec := new(DirSpecImpl)
	spec.args = args

	// If the spec is absolute, then we can just save the path directly.
	if !strings.HasPrefix(rawSpec, "//") {
		spec.path = filepath.Join(buildBase, cwd, rawSpec)
	} else {
		spec.path = filepath.Join(buildBase, rawSpec)
	}

	// Check to see whether this file exists and is a file. If it doesn't, then
	// we don't have a FileSpec.
	spec.path = strings.Replace(spec.path, pathSeparator, "/", -1)
	if common.FileExists(spec.Path()) && common.IsDir(spec.Path()) {
		return spec
	}

	fmt.Printf("FAILED\n")
	return nil
}
