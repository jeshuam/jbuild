package config

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Cache targets to avoid re-loading them multiple times.
	targetCache = make(map[string]*Target)
)

// A target object
type TargetSpec struct {
	Path, Name string
}

func (this *TargetSpec) String() string {
	return "//" + this.Path + ":" + this.Name
}

func splitTargetSpec(targetSpec string) (string, string) {
	// Split the target into path and target.
	parts := strings.Split(targetSpec, ":")
	targetPath := parts[0]
	_, targetName := path.Split(targetPath)
	if len(parts) > 1 {
		targetName = parts[1]
	}

	return targetPath, targetName
}

// Convert the given target into it's canonical form. There are 2 ways to
// specify a target:
//
//     1. Absolute (//path/to/target:target_name)
//     2. Relative (path/to/target:target_name).
//
// target_name may be optionally excluded if the target name is the same as the
// directory in which the target lives. The directory can be disregarded
// completely, but in this case the target must be provided.
func CanonicalTargetSpec(workspaceDir, cwd, target string) (*TargetSpec, error) {
	// Split the target into path and target.
	targetPath, targetName := splitTargetSpec(target)

	// Make sure the target conforms to a regex.
	match, err := regexp.MatchString("^(//)?[a-z]+([/a-z]+)?(:[a-z]+)?$", target)
	if err != nil {
		log.Fatalf("Target regex matching failed: %v", err)
	}

	if !match {
		// Check for special cases: the target path is either empty or just the
		// root path.
		if targetPath != "" && targetPath != "//" {
			log.Info(targetPath)
			return nil, errors.New(fmt.Sprintf("Target didn't match regex"))
		}
	}

	// Convert the target into it's absolute form.
	if !strings.HasPrefix(targetPath, "//") {
		// Special case: workspaceDir and cwd are the same, so the path is actually
		// absolute.
		if workspaceDir == cwd {
			targetPath = targetPath
		} else {
			// Find the relative location of this directory to the workspace directory.
			workspacePath, err := filepath.Rel(workspaceDir, cwd)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("Couldn't make %s relative", target))
			}

			internalPath := strings.Replace(workspacePath, pathSeparator, "/", -1)
			targetPath = internalPath + "/" + targetPath
		}
	} else {
		targetPath = strings.TrimPrefix(targetPath, "//")
	}

	// Build the final target.
	targetSpec := new(TargetSpec)
	targetSpec.Path = targetPath
	targetSpec.Name = targetName
	return targetSpec, nil
}

// Representation of a single Target.
type Target struct {
	Spec       *TargetSpec
	Deps       []*Target
	Processed  bool
	Processing bool
}

func (this *Target) String() string {
	return this.Spec.String()
}

func (this *Target) LoadDependencies(depSpecs []*TargetSpec, workspaceDir string) error {
	for _, depSpec := range depSpecs {
		target, err := LoadTarget(depSpec, workspaceDir)
		if err != nil {
			return err
		}

		this.Deps = append(this.Deps, target)
	}

	return nil
}

func (this *Target) checkForDependencyCyclesRecurse(visited []string, seq int) {
	// If this node has already been visited in the current recursive stack, then
	// there must be a cycle in the graph.
	for i := 0; i < seq; i++ {
		if visited[i] == this.Spec.String() {
			cycle := strings.Join(visited[i:seq], " --> ")
			log.Fatalf("Cycle detected: %s --> %s", cycle, this)
		}
	}

	// Add this item to sequence number `seq`. If the array is already big enough,
	// then just replace what is in there. Otherwise, add a new element.
	if len(visited) > seq {
		visited[seq] = this.Spec.String()
	} else {
		visited = append(visited, this.Spec.String())
	}

	// Look through each child of the current target and recurse to it, first
	// adding this node to the visited list.
	for _, dep := range this.Deps {
		dep.checkForDependencyCyclesRecurse(visited, seq+1)
	}
}

func (this *Target) CheckForDependencyCycles() {
	visited := make([]string, 0, 1)
	visited = append(visited, this.Spec.String())
	for _, dep := range this.Deps {
		dep.checkForDependencyCyclesRecurse(visited, 1)
	}
}

// Check whether this targer is ready to process (i.e. all dependencies are
// processed).
func (this *Target) ReadyToProcess() bool {
	for _, dep := range this.Deps {
		if !dep.Processed {
			return false
		}
	}

	return true
}

func makeTarget(json map[string]interface{}, targetSpec *TargetSpec) (*Target, []*TargetSpec) {
	target := new(Target)
	target.Spec = targetSpec

	// Retuen a list of dependencies which have to be processed.
	depSpecs := make([]*TargetSpec, 0)

	// Load the dependency names.
	depNames, ok := json["deps"]
	if ok {
		for _, depNameInterface := range depNames.([]interface{}) {
			depName := depNameInterface.(string)
			if strings.HasPrefix(depName, "//") {
				depName = target.Spec.Path + depName
			} else {
				depName = strings.TrimPrefix(depName, "//")
			}

			depPath, depName := splitTargetSpec(depName)

			depSpec := new(TargetSpec)
			depSpec.Path = depPath
			depSpec.Name = depName
			depSpecs = append(depSpecs, depSpec)
		}
	}

	return target, depSpecs
}

// Load the given target spec and all related dependencies into Target objects.
func LoadTarget(targetSpec *TargetSpec, workspaceDir string) (*Target, error) {
	// If we have already loaded this target, then return it.
	target, inCache := targetCache[targetSpec.String()]

	if !inCache {
		// Load all of the targets in the given BUILD file.
		buildFilepath := path.Join(workspaceDir, targetSpec.Path, *buildFilename)
		targetsJSON, err := LoadBuildFile(buildFilepath)
		if err != nil {
			return nil, err
		}

		// Load the target from the build file.
		targetJSON, exists := targetsJSON[targetSpec.Name]
		if !exists {
			return nil, errors.New(fmt.Sprintf("Unknown target %s", targetSpec))
		}

		// Load the target from it's JSON.
		var depSpecs []*TargetSpec
		target, depSpecs = makeTarget(targetJSON.(map[string]interface{}), targetSpec)
		targetCache[targetSpec.String()] = target

		// Process dependencies for this target.
		err = target.LoadDependencies(depSpecs, workspaceDir)
		if err != nil {
			return nil, err
		}

		log.Debugf("Loaded target %s", target)
	}

	return target, nil
}
