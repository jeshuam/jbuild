package config2

import (
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")
)

// Implementation of the FileSpec interface.
type FileSpecImpl struct {
	path string
	file string
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////
func (this *FileSpecImpl) Dir() string {
	return this.path
}

func (this *FileSpecImpl) Path() string {
	return filepath.Join(
		common.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *FileSpecImpl) String() string {
	return "//" + this.Dir() + "/" + this.File()
}

func (this *FileSpecImpl) File() string {
	return this.file
}

func (this *FileSpecImpl) FilePath() string {
	return filepath.Join(this.Path(), this.File())
}

func (this *FileSpecImpl) OutputPath() string {
	return filepath.Join(
		common.OutputDirectory, strings.Replace(this.path, "/", pathSeparator, -1))
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// MakeFileSpec constructs and returns a valid FileSpec object, or nil if the
// given spec doesn't refer to a valid file. rawSpec can be absolute or relative
// to `cwd`.
func MakeFileSpec(rawSpec, cwd string) interfaces.FileSpec {
	spec := new(FileSpecImpl)

	// If the spec isn't absolute, we need to make it relative.
	if !strings.HasPrefix(rawSpec, "//") {
		rawSpec, _ = filepath.Rel(common.WorkspaceDir, filepath.Join(cwd, spec.path))
	}

	// Split the string into it's file and dir parts.
	rawSpec = strings.Trim(rawSpec, "/")
	spec.path, spec.file = filepath.Split(strings.Replace(rawSpec, "/", pathSeparator, -1))
	spec.path = strings.Trim(strings.Replace(spec.path, pathSeparator, "/", -1), "/")

	// Check to see whether this file exists and is a file. If it doesn't, then
	// we don't have a FileSpec.
	if common.FileExists(spec.FilePath()) && !common.IsDir(spec.FilePath()) {
		return spec
	}

	return nil
}

// MakeFileSpecGlob constructs and returns a list of FileSpec objects, or nil if
// the given spec doesn't refer to any valid files. rawSpec can be absolute or
// relative to `cwd`.
func MakeFileSpecGlob(rawSpecGlob, cwd string) []interfaces.Spec {
	specs := make([]interfaces.Spec, 0)

	// If the spec is absolute, then we can just save the path directly.
	if !strings.HasPrefix(rawSpecGlob, "//") {
		rawSpecGlob, _ = filepath.Rel(
			common.WorkspaceDir, filepath.Join(cwd, strings.Trim(rawSpecGlob, "/")))
	}

	// Expand the globs.
	globs, _ := Glob(rawSpecGlob)
	for _, glob := range globs {
		specs = append(specs, MakeFileSpec("//"+strings.Replace(glob, pathSeparator, "/", -1), ""))
	}

	return specs
}
