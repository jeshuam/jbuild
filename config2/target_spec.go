package config2

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	// "reflect"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/cc"
	"github.com/jeshuam/jbuild/config2/filegroup"
	"github.com/jeshuam/jbuild/config2/interfaces"
	"github.com/jeshuam/jbuild/config2/util"
)

var (
	BuildFileName string

	// Some useful global shortcut variables.
	pathSeparator = string(os.PathSeparator)
)

func init() {
	flag.StringVar(&BuildFileName, "build_filename_v2", "BUILD",
		"Name of the file containing the target spec definitions.")
}

// A TargetSpec is a single unit within the build system; this could be a target
// (e.g. a C++ library) which is referenced inside a BUILD file, or it could
// just be an ordinary file.
type TargetSpec struct {
	// The relative path to the file from the workspace root. Should always use
	// the "/" character as the path separator. This path should not include a
	// trailing "/" character.
	path  string
	_type string

	// The name of the file (or target) at the given workspace path.
	name string

	// The target this spec refers to. It may be nil for files. Will not be set
	// until Load() has been called.
	isTarget bool
}

////////////////////////////////////////////////////////////////////////////////
//                              TargetSpec Getters                            //
////////////////////////////////////////////////////////////////////////////////

func (this *TargetSpec) Dir() string {
	return this.path
}

func (this *TargetSpec) Name() string {
	return this.name
}

func (this *TargetSpec) Type() string {
	return this._type
}

func (this *TargetSpec) Path() string {
	if !this.IsTarget() {
		return filepath.Join(
			common.WorkspaceDir,
			strings.Replace(this.path, "/", pathSeparator, -1),
			this.name)
	} else {
		return filepath.Join(
			common.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
	}
}

func (this *TargetSpec) OutputPath() string {
	if !this.IsTarget() {
		return filepath.Join(
			common.OutputDirectory,
			strings.Replace(this.path, "/", pathSeparator, -1),
			this.name)
	} else {
		return filepath.Join(
			common.OutputDirectory, strings.Replace(this.path, "/", pathSeparator, -1))
	}
}

func (this *TargetSpec) IsTarget() bool {
	return this.isTarget
}

func (this *TargetSpec) Target() interfaces.Target {
	return util.TargetCache[this.String()]
}

func (this *TargetSpec) String() string {
	if this.IsTarget() {
		return "//" + this.path + ":" + this.name
	} else {
		return "//" + this.path + "/" + this.name
	}
}

func (this *TargetSpec) Validate() error {
	return util.TargetCache[this.String()].Validate()
}

////////////////////////////////////////////////////////////////////////////////
//                            TargetSpec Methods                              //
////////////////////////////////////////////////////////////////////////////////

func (this *TargetSpec) Init() error {
	// If this isn't a target, then stop.
	if !this.IsTarget() {
		this._type = "file"
		return nil
	}

	// See if the target is in the cache. If it is, just use that.
	_, ok := util.TargetCache[this.String()]
	if ok {
		return nil
	}

	// Try to load the BUILD file corresponding to this spec. If there is no BUILD
	// file, then there is nothing else to do.
	buildFile, err := LoadBuildFile(filepath.Join(this.Path(), BuildFileName))
	if err != nil {
		return err
	}

	// If this file is a target, then load it. Otherwise, we have nothing else
	// to do.
	targetJsonInterface, ok := buildFile[this.name]
	if !ok {
		return errors.New(fmt.Sprintf("Unknown target %s", this))
	}

	// Extract the type from the target.
	targetJson := targetJsonInterface.(map[string]interface{})
	targetTypeInterface, ok := targetJson["type"]
	if !ok {
		return errors.New(fmt.Sprintf("Target %s missing required 'type' field."))
	}

	// Based on the type, create a new target.
	this._type = targetTypeInterface.(string)
	var target interfaces.Target
	if strings.HasPrefix(this._type, "c++") {
		target = new(cc.Target)
	} else if strings.HasPrefix(this._type, "filegroup") {
		target = new(filegroup.Target)
	} else {
		return errors.New(fmt.Sprintf("Target %s has unknown type %s", this, this._type))
	}

	// Cache the target.
	util.TargetCache[this.String()] = target

	// Load the target.
	return LoadTargetFromJson(this, target, targetJson)
}

////////////////////////////////////////////////////////////////////////////////
//                       TargetSpec Utility Functions                         //
////////////////////////////////////////////////////////////////////////////////

// Make a file spec object from the given path. The TargetSpec returned will not
// be valid if an error has been returned.
func MakeTargetSpec(path string) *TargetSpec {
	spec := new(TargetSpec)

	// If the path is not absolute (there is no // at the start), then we need
	// to figure out where it is relative to the current directory.
	if !strings.HasPrefix(path, "//") {
		path, _ = filepath.Rel(
			common.WorkspaceDir, filepath.Join(common.CurrentDir, path))
	}

	// If a file exists with the given path, then it must be a file. Unless it's
	// a directory, in which case it's meant to be a target.
	fullPath := filepath.Join(common.WorkspaceDir, path)
	if common.FileExists(fullPath) && !common.IsDir(fullPath) {
		spec.path = filepath.Dir(path)
		spec.name = filepath.Base(path)
		spec.isTarget = false
	} else {
		parts := strings.SplitN(path, ":", 2)
		spec.path = parts[0]
		if len(parts) == 2 {
			spec.name = parts[1]
		} else {
			pathParts := strings.Split(spec.path, "/")
			spec.name = pathParts[len(pathParts)-1]
		}

		spec.isTarget = true
	}

	// Trim any trailing or proceeding slashes from the path name.
	spec.path = strings.Trim(strings.Trim(spec.path, "/"), pathSeparator)
	spec.path = strings.Replace(spec.path, pathSeparator, "/", -1)
	return spec
}
