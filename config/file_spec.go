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
	// The location of the file within the workspace.
	// e.g. //this/is/afile.txt --> this/is
	dir string

	// The name of the file.
	// e.g. //this/is/afile.txt --> afile.txt
	filename string

	// The path to the root of the workspace.
	// e.g. /home/ws/this/is/afile.txt --> /home/ws
	workspacePath string

	// `true` if the file is generated, `false` otherwise. Generated files need
	// not exist when the spec is created, and all output references point them to
	// the generated file directory in `args`.
	isGenerated bool

	args *args.Args
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////

// LEGACY: try not use these if possible.
func (this *FileSpecImpl) Dir() string {
	return this.dir
}

func (this *FileSpecImpl) Path() string {
	return this.FsPath()
}

func (this *FileSpecImpl) String() string {
	if this.Dir() == "" {
		return "//" + this.Filename()
	} else {
		return "//" + this.Dir() + "/" + this.Filename()
	}
}

func (this *FileSpecImpl) Type() string {
	return "file"
}

// NEW: use these if possible.
func (this *FileSpecImpl) Filename() string {
	return this.filename
}

func (this *FileSpecImpl) FsWorkspacePath() string {
	return this.workspacePath
}

func (this *FileSpecImpl) FsOutputDir() string {
	if this.isGenerated {
		return filepath.Join(this.args.GenOutputDir, this.Dir())
	} else {
		return filepath.Join(this.args.OutputDir, this.Dir())
	}
}

func (this *FileSpecImpl) FsOutputPath() string {
	return filepath.Join(this.FsOutputDir(), this.Filename())
}

func (this *FileSpecImpl) FsPath() string {
	if this.IsGenerated() {
		return this.FsOutputPath()
	} else {
		return filepath.Join(this.args.WorkspaceDir, this.Dir(), this.Filename())
	}
}

func (this *FileSpecImpl) IsGenerated() bool {
	return this.isGenerated
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
func MakeFileSpec(args *args.Args, rawSpec, cwd, buildBase string, isGenerated bool) interfaces.FileSpec {
	spec := new(FileSpecImpl)
	spec.args = args

	// If the spec is absolute, we already have the correct path.
	spec.workspacePath = buildBase
	if strings.HasPrefix(rawSpec, "//") {
		spec.dir, spec.filename = splitPathAndFile(strings.Trim(rawSpec, "/"), "/")
	} else {
		spec.dir, spec.filename = splitPathAndFile(filepath.Clean(filepath.Join(cwd, rawSpec)), "/")
	}

	// If the file is generated, save that and stop.
	if isGenerated {
		spec.isGenerated = isGenerated
		return spec
	}

	// Check if the file exists.
	if common.FileExists(spec.FsPath()) && !common.IsDir(spec.FsPath()) {
		return spec
	}

	return nil
}

// MakeFileSpecGlob constructs and returns a list of FileSpec objects, or nil if
// the given spec doesn't refer to any valid files. rawSpec can be absolute or
// relative to `cwd`.
func MakeFileSpecGlob(args *args.Args, rawSpecGlob, cwd, buildBase string) []interfaces.Spec {
	// If the spec is absolute, then we can just save the path directly.
	if !strings.HasPrefix(rawSpecGlob, "//") {
		workspacePath, _ := filepath.Rel(buildBase, cwd)
		rawSpecGlob = filepath.Join(workspacePath, rawSpecGlob)
	}

	// Expand the globs.
	globToSearch := filepath.Join(buildBase, cwd, strings.Trim(rawSpecGlob, "/"))
	globs, _ := Glob(globToSearch)
	specs := make([]interfaces.Spec, 0, len(globs))
	for _, glob := range globs {
		globRel, _ := filepath.Rel(buildBase, glob)
		spec := MakeFileSpec(args, util.OSPathToWSPath(globRel), "", buildBase, false)
		if spec != nil {
			specs = append(specs, spec)
		}
	}

	return specs
}
