package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/config/cc"
	"github.com/jeshuam/jbuild/config/filegroup"
	"github.com/jeshuam/jbuild/config/interfaces"
	"github.com/jeshuam/jbuild/config/util"
)

var (
	// Some useful global shortcut variables.
	pathSeparator = string(os.PathSeparator)
)

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
		args.WorkspaceDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

func (this *TargetSpecImpl) String() string {
	return "//" + this.Dir() + ":" + this.Name()
}

func (this *TargetSpecImpl) Type() string {
	return this._type
}

func (this *TargetSpecImpl) Name() string {
	return this.name
}

func (this *TargetSpecImpl) Target() interfaces.Target {
	return util.TargetCache[this.String()]
}

func (this *TargetSpecImpl) OutputPath() string {
	return filepath.Join(
		args.OutputDir, strings.Replace(this.path, "/", pathSeparator, -1))
}

////////////////////////////////////////////////////////////////////////////////
//                            TargetSpec Methods                              //
////////////////////////////////////////////////////////////////////////////////

func (this *TargetSpecImpl) init(json map[string]interface{}) error {
	// If the target has already been loaded, then just return it.
	cachedTarget, ok := util.TargetCache[this.String()]
	if ok {
		this.target = cachedTarget
		return nil
	}

	// Extract the type from the target.
	targetTypeInterface, ok := json["type"]
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
	return LoadTargetFromJson(this, this.Target(), json)
}

////////////////////////////////////////////////////////////////////////////////
//                       TargetSpec Utility Functions                         //
////////////////////////////////////////////////////////////////////////////////

func expandAllTargetsInTree(workspaceDir, path string) ([]interfaces.TargetSpec, error) {
	buildFiles, err := Glob(filepath.Join(workspaceDir, path, "**", args.BuildFilename))
	if err != nil {
		return nil, err
	}

	finalSpecs := make([]interfaces.TargetSpec, 0)
	for _, buildFile := range buildFiles {
		buildRelPath, _ := filepath.Rel(workspaceDir, buildFile)
		buildFilePath, _ := filepath.Split(buildRelPath)
		targets, err := expandAllTargetsInDir(workspaceDir, buildFilePath)
		if err != nil {
			return nil, err
		}

		finalSpecs = append(finalSpecs, targets...)
	}

	return finalSpecs, nil
}

func expandAllTargetsInDir(workspaceDir, path string) ([]interfaces.TargetSpec, error) {
	log.Debugf("Scanning for all targets in '%s'", util.OSPathToWSPath(path))
	buildFilepath := filepath.Join(workspaceDir, path, args.BuildFilename)
	targetsJSON, err := LoadBuildFile(buildFilepath)
	if err != nil {
		return nil, err
	}

	targets := make([]interfaces.TargetSpec, 0, len(targetsJSON))
	for targetName := range targetsJSON {
		targetPath := util.OSPathToWSPath(path) + ":" + targetName
		log.Debugf("Found target '%s'", targetPath)
		specs, err := MakeTargetSpec(targetPath, "")
		if err != nil {
			return nil, err
		}

		targets = append(targets, specs...)
	}

	return targets, nil
}

// MakeTargetSpec constructs and returns a valid TargetSpec object, or nil if
// the given spec doesn't refer to a valid target. rawSpec can be absolute or
// relative to `cwd`.
func MakeTargetSpec(rawSpec string, cwd string) ([]interfaces.TargetSpec, error) {
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
		spec.path, _ = filepath.Rel(args.WorkspaceDir, filepath.Join(cwd, rawPath))
	}

	// Replace any last OS-level separators. Now we can see if it has been cached!
	spec.path = strings.Replace(spec.path, pathSeparator, "/", -1)
	cachedSpec, ok := util.SpecCache[spec.String()]
	if ok {
		return []interfaces.TargetSpec{cachedSpec.(interfaces.TargetSpec)}, nil
	}

	// Final special case: if the target name is a special value, then expand the
	// target into multiple targets.
	if spec.name == "all" {
		log.Infof("Expanding target '%s'", spec)
		return expandAllTargetsInDir(args.WorkspaceDir, spec.Dir())
	} else if strings.HasSuffix(spec.path, "...") {
		log.Infof("Expanding target '//%s'", spec.Dir())
		targetPathWithoutDots, _ := filepath.Split(spec.Dir())
		return expandAllTargetsInTree(args.WorkspaceDir, targetPathWithoutDots)
	}

	// Check to see whether the target exists. This requires that the BUILD file
	// for this directory is parsed.
	buildFile, err := LoadBuildFile(filepath.Join(spec.Path(), args.BuildFilename))
	if err != nil {
		return nil, err
	}

	targetJson, ok := buildFile[spec.Name()]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unknown target spec %s", rawSpec))
	}

	// Otherwise, initialize the target spec and return it.
	err = spec.init(targetJson.(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	util.SpecCache[spec.String()] = spec
	return []interfaces.TargetSpec{spec}, nil
}
