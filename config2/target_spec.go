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

type TargetSpecImpl struct {
	path   string
	name   string
	_type  string
	target interfaces.Target
}

////////////////////////////////////////////////////////////////////////////////
//                          Interface Implementation                          //
////////////////////////////////////////////////////////////////////////////////

func (this *TargetSpecImpl) Dir() string {
	return this.path
}

func (this *TargetSpecImpl) Path() string {
	return filepath.Join(
		common.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *TargetSpecImpl) String() string {
	return "//" + this.Dir() + ":" + this.Name()
}

func (this *TargetSpecImpl) Name() string {
	return this.name
}

func (this *TargetSpecImpl) Target() interfaces.Target {
	return util.TargetCache[this.String()]
}

func (this *TargetSpecImpl) OutputPath() string {
	return filepath.Join(
		common.OutputDirectory, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *TargetSpecImpl) Type() string {
	return this._type
}

////////////////////////////////////////////////////////////////////////////////
//                            TargetSpec Methods                              //
////////////////////////////////////////////////////////////////////////////////

func (this *TargetSpecImpl) init() error {
	// See if the target is in the cache. If it is, just use that.
	_, ok := util.TargetCache[this.String()]
	if ok {
		this.target = util.TargetCache[this.String()]
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
	if strings.HasPrefix(this._type, "c++") {
		this.target = new(cc.Target)
	} else if strings.HasPrefix(this._type, "filegroup") {
		this.target = new(filegroup.Target)
	} else {
		return errors.New(fmt.Sprintf("Target %s has unknown type %s", this, this._type))
	}

	// Cache the target.
	util.TargetCache[this.String()] = this.target

	// Load the target.
	return LoadTargetFromJson(this, this.Target(), targetJson)
}

////////////////////////////////////////////////////////////////////////////////
//                       TargetSpec Utility Functions                         //
////////////////////////////////////////////////////////////////////////////////

// MakeTargetSpec constructs and returns a valid TargetSpec object, or nil if
// the given spec doesn't refer to a valid target. rawSpec can be absolute or
// relative to `cwd`.
func MakeTargetSpec(rawSpec string, cwd string) (interfaces.TargetSpec, error) {
	spec := new(TargetSpecImpl)

	// Split the string into it's file and dir parts.
	rawSpecParts := strings.Split(rawSpec, ":")
	rawPath := rawSpecParts[0]
	if len(rawSpecParts) == 2 {
		spec.name = rawSpecParts[1]
	} else {
		rawPathParts := strings.Split(rawPath, "/")
		spec.name = rawPathParts[len(rawPathParts)-1]
	}

	// If the spec is absolute, then we can just save the path directly.
	if strings.HasPrefix(rawSpec, "//") {
		spec.path = strings.Trim(rawPath, "/")
	} else {
		// Otherwise, we need to figure out the absolute path.
		spec.path, _ = filepath.Rel(common.WorkspaceDir, filepath.Join(cwd, rawPath))
	}

	// Check to see whether the target exists. This requires that the BUILD file
	// for this directory is parsed.
	buildFile, err := LoadBuildFile(filepath.Join(spec.Path(), BuildFileName))
	if err != nil {
		return nil, err
	}

	_, ok := buildFile[spec.Name()]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unknown target spec %s", rawSpec))
	}

	// Otherwise, initialize the target spec and return it.
	err = spec.init()
	if err != nil {
		return nil, err
	}

	return spec, nil
}
