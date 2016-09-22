package config

import (
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
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
		args.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *FileSpecImpl) String() string {
	if this.Dir() == "" {
		return "//" + this.File()
	} else {
		return "//" + this.Dir() + "/" + this.File()
	}
}

func (this *FileSpecImpl) Type() string {
	return "file"
}

func (this *FileSpecImpl) File() string {
	return this.file
}

func (this *FileSpecImpl) FilePath() string {
	return filepath.Join(this.Path(), this.File())
}

func (this *FileSpecImpl) OutputPath() string {
	return filepath.Join(
		args.OutputDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

////////////////////////////////////////////////////////////////////////////////
//                             Utility Functions                              //
////////////////////////////////////////////////////////////////////////////////

// splitPathAndFile is a utility function which splits a given path into the
// (path, file) components while taking the separator to use as an argument. The
// returned path will also use a "/" as the separator.
func splitPathAndFile(path, sep string) (string, string) {
	pathParts := strings.Split(path, sep)
	return strings.Join(pathParts[0:len(pathParts)-1], "/"),
		pathParts[len(pathParts)-1]
}

// MakeFileSpec constructs and returns a valid FileSpec object, or nil if the
// given spec doesn't refer to a valid file. rawSpec can be absolute or relative
// to `cwd`.
func MakeFileSpec(rawSpec, cwd string) interfaces.FileSpec {
	spec := new(FileSpecImpl)

	// If the spec is absolute, we already have the correct path.
	if strings.HasPrefix(rawSpec, "//") {
		spec.path, spec.file = splitPathAndFile(
			strings.Trim(rawSpec, "/"), "/")
	} else {
		workspacePath, _ := filepath.Rel(args.WorkspaceDir, cwd)
		spec.path, spec.file = splitPathAndFile(
			filepath.Join(workspacePath, rawSpec), pathSeparator)
	}

	// Check to see whether this file exists and is a file. If it doesn't, then
	// we don't have a FileSpec.
	spec.path = strings.Trim(strings.Replace(spec.path, pathSeparator, "/", -1), "/")
	if common.FileExists(spec.FilePath()) && !common.IsDir(spec.FilePath()) {
		return spec
	}

	return nil
}

// MakeFileSpecGlob constructs and returns a list of FileSpec objects, or nil if
// the given spec doesn't refer to any valid files. rawSpec can be absolute or
// relative to `cwd`.
func MakeFileSpecGlob(rawSpecGlob, cwd string) []interfaces.Spec {
	// If the spec is absolute, then we can just save the path directly.
	if !strings.HasPrefix(rawSpecGlob, "//") {
		workspacePath, _ := filepath.Rel(args.WorkspaceDir, cwd)
		rawSpecGlob = filepath.Join(workspacePath, rawSpecGlob)
	}

	// Expand the globs.
	globToSearch := filepath.Join(args.WorkspaceDir, strings.Trim(rawSpecGlob, "/"))
	globs, _ := Glob(globToSearch)
	specs := make([]interfaces.Spec, 0, len(globs))
	for _, glob := range globs {
		globRel, _ := filepath.Rel(args.WorkspaceDir, glob)
		spec := MakeFileSpec(util.OSPathToWSPath(globRel), "")
		if spec != nil {
			specs = append(specs, spec)
		}
	}

	return specs
}
