package config

import (
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/common"
)

// A FileSpec is a single unit within the build system; this could be a target
// (e.g. a C++ library) which is referenced inside a BUILD file, or it could
// just be an ordinary file.
type FileSpec struct {
	// The relative path to the file from the workspace root. Should always use
	// the "/" character as the path separator. This path should not include a
	// trailing "/" character.
	Path string

	// The name of the file (or target) at the given workspace path.
	Name string

	// The target this spec refers to. It may be nil for files. Will not be set
	// until Load() has been called.
	target *Target
}

// Make a file spec object from the given path. The FileSpec returned will not
// be valid if an error has been returned.
func MakeFileSpec(path string) (FileSpec, error) {
	spec := FileSpec{}

	// If a file exists with the given path, then it must be a file.
	if fileExists(path) {

	} else {
		parts := strings.SplitN(path, ":", 1)
		spec.Path = strings.Trim(parts[0], "/")
		if len(parts) == 2 {
			spec.Name = parts[1]
		} else {
			spec.Name = filepath.Base(spec.Path)
		}
	}

	// Check to see if this is a target. This just makes sure there is a ":"
	// somewhere in the string. If there isn't, we will assume it's a file.
	isTarget := strings.Contains(path, ":") || strings.Contains(path, "//")
	if isTarget {

	}

	return spec, nil
}

func (this FileSpec) String() string {
	return "//" + this.Path + ":" + this.Name
}

func (this FileSpec) pathSystem() string {
	return strings.Replace(this.Path, "/", pathSeparator, -1)
}

func (this FileSpec) WorkspacePath() string {
	return filepath.Join(common.WorkspaceDir, this.pathSystem())
}

func loadTargetFromJson(targetJson map[string]interface{}) (*Target, error) {
	target := new(Target)

	return target, nil
}

func (this FileSpec) Load() error {
	// Try to load the BUILD file corresponding to this spec. If there is no BUILD
	// file, then there is nothing else to do.
	buildFileName := filepath.Join(this.WorkspacePath(), *buildFilename)
	if common.FileExists(buildFileName) {
		buildFile, err := LoadBuildFile(buildFileName)
		if err != nil {
			return err
		}

		// If this file is a target, then load it. Otherwise, we have nothing else
		// to do.
		targetJson, ok := buildFile[this.Name]
		if ok {
			this.target, err = loadTargetFromJson(targetJson.(map[string]interface{}))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this FileSpec) IsTarget() bool {
	return this.target != nil
}

func (this FileSpec) IsFile() bool {
	return !this.IsTarget()
}
