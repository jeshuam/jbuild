package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/progress"
)

var (
	// Cache targets to avoid re-loading them multiple times.
	targetCache = make(map[string]*Target)
)

// A target object
type TargetSpec struct {
	Path, Name, Workspace string
}

func (this *TargetSpec) String() string {
	return "//" + this.Path + ":" + this.Name
}

func (this *TargetSpec) PathSystem() string {
	return strings.Replace(this.Path, "/", pathSeparator, -1)
}

func (this *TargetSpec) OutputPath() string {
	return filepath.Join(common.OutputDirectory, this.PathSystem())
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
	match, err := regexp.MatchString("^(//)?[0-9A-Za-z_]+([/0-9A-Za-z_]+)?(:[0-9A-Za-z_]+)?$", target)
	if err != nil {
		log.Fatalf("Target regex matching failed: %v", err)
	}

	if !match {
		// Check for special cases: the target path is either empty or just the
		// root path.
		if targetPath != "" && targetPath != "//" {
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
	targetSpec.Workspace = workspaceDir
	return targetSpec, nil
}

type TargetOptions struct {
	Srcs         []string // A list of source files to build. Relative to the target dir.
	Hdrs         []string // A list of header files included with this target.
	CompileFlags []string // A list of compilation flags to pass to the compiler
	LinkFlags    []string
	Includes     []string // Extra directories to include.
	Libs         []string // A list of pre-compiled libraries to include.
}

// Representation of a single Target.
type Target struct {
	Spec      *TargetSpec // The path/name of this target. Useful for printing.
	Type      string      // The type of the target (e.g. c++/library)
	Deps      []*Target   // The targets which this target depends on.
	Processed bool        // Whether or not this target has been processed.
	Changed   bool        // Set to true if the target has changed or not.

	// Options which are required for processors. Note that not all of these may be
	// used depending on the type of target.
	Options         TargetOptions // Common options for all platforms.
	PlatformOptions TargetOptions // Platform specific options.

	// Output options.
	Output []string // A list of output files produced by this target. Should be populated by a processor.

	// Progress bar for updating the display.
	ProgressBar *progress.ProgressBar
}

// Getters for various options, which combine platform and non-platform options.
func (this *Target) Srcs() []string {
	srcs := make([]string, 0)
	srcs = append(srcs, this.Options.Srcs...)
	srcs = append(srcs, this.PlatformOptions.Srcs...)
	return srcs
}

func (this *Target) Hdrs() []string {
	hdrs := make([]string, 0)
	hdrs = append(hdrs, this.Options.Hdrs...)
	hdrs = append(hdrs, this.PlatformOptions.Hdrs...)
	return hdrs
}

func (this *Target) CompileFlags() []string {
	compileFlags := make([]string, 0)
	compileFlags = append(compileFlags, this.Options.CompileFlags...)
	compileFlags = append(compileFlags, this.PlatformOptions.CompileFlags...)
	return compileFlags
}

func (this *Target) LinkFlags() []string {
	linkFlags := make([]string, 0)
	linkFlags = append(linkFlags, this.Options.LinkFlags...)
	linkFlags = append(linkFlags, this.PlatformOptions.LinkFlags...)
	return linkFlags
}

func (this *Target) Includes() []string {
	includes := make([]string, 0)
	includes = append(includes, this.Options.Includes...)
	includes = append(includes, this.PlatformOptions.Includes...)
	return includes
}

func (this *Target) Libs() []string {
	libs := make([]string, 0)
	libs = append(libs, this.Options.Libs...)
	libs = append(libs, this.PlatformOptions.Libs...)
	return libs
}

func (this *Target) LibsOrdered() []string {
	libs := make([]string, 0)
	for _, lib := range this.Libs() {
		libs = append(libs, filepath.Join(this.Spec.Workspace, this.Spec.PathSystem(), lib))
	}

	for _, dep := range this.Deps {
		libs = append(libs, dep.LibsOrdered()...)
	}

	return libs
}

func (this *Target) OutputOrdered() []string {
	outputs := make([]string, 0)
	for _, output := range this.Output {
		outputs = append(outputs, output)
	}

	for _, dep := range this.Deps {
		outputs = append(outputs, dep.OutputOrdered()...)
	}

	return outputs
}

func (this *Target) String() string {
	return this.Spec.String()
}

func (this *Target) IsBinary() bool {
	return strings.HasSuffix(this.Type, "binary")
}

func (this *Target) IsTest() bool {
	return strings.HasSuffix(this.Type, "test")
}

func (this *Target) IsLibrary() bool {
	return strings.HasSuffix(this.Type, "library")
}

func (this *Target) IsExecutable() bool {
	return this.IsBinary() || this.IsTest()
}

func (this *Target) LoadDependencies(depSpecs []*TargetSpec) error {
	for _, depSpec := range depSpecs {
		target, err := LoadTarget(depSpec)
		if err != nil {
			return err
		}

		this.Deps = append(this.Deps, target)
	}

	return nil
}

// Return true if any of the header files within the target or it's dependencies
// have changed.
func (this *Target) HeaderFilesChangedAfter(file os.FileInfo) bool {
	for _, hdr := range this.Hdrs() {
		hdrPath := filepath.Join(this.Spec.Workspace, this.Spec.PathSystem(), hdr)
		hdrStat, _ := os.Stat(hdrPath)
		if hdrStat != nil && hdrStat.ModTime().After(file.ModTime()) {
			fmt.Printf("header %s changed after %s\n", hdr, file.Name())
			return true
		}
	}

	for _, dep := range this.AllDependencies() {
		if dep.HeaderFilesChangedAfter(file) {
			return true
		}
	}

	return false
}

// Get a list of all dependencies of target.
func (this *Target) AllDependencies() []*Target {
	deps := []*Target{}
	deps = append(deps, this.Deps...)
	for _, dep := range this.Deps {
		deps = append(deps, dep.AllDependencies()...)
	}

	return deps
}

func (this *Target) DependenciesChanged() bool {
	for _, dep := range this.AllDependencies() {
		if dep.Changed {
			return true
		}
	}

	return false
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

func loadArrayFromJson(json map[string]interface{}, key string) []string {
	out := make([]string, 0)
	arr, ok := json[key]
	if ok {
		for _, item := range arr.([]interface{}) {
			out = append(out, item.(string))
		}
	}

	return out
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
			if !strings.HasPrefix(depName, "//") {
				depName = target.Spec.Path + depName
			} else {
				depName = strings.TrimPrefix(depName, "//")
			}

			depPath, depName := splitTargetSpec(depName)

			depSpec := new(TargetSpec)
			depSpec.Path = depPath
			depSpec.Name = depName
			depSpec.Workspace = targetSpec.Workspace
			depSpecs = append(depSpecs, depSpec)
		}
	}

	// Function to load a list of globs from the config.
	loadGlobs := func(root map[string]interface{}, key string) []string {
		globs := loadArrayFromJson(root, key)
		finalFiles := make([]string, 0)
		for _, glob := range globs {
			glob = path.Join(targetSpec.Workspace, targetSpec.PathSystem(), glob)
			files, err := filepath.Glob(glob)

			// If there was an error, then just return the glob by itself.
			if err != nil {
				finalFiles = append(finalFiles, glob)
			} else {
				// Otherwise, convert globs into actual paths.
				rel, _ := filepath.Rel(filepath.Join(target.Spec.Workspace, target.Spec.Path), filepath.Dir(glob))
				for _, file := range files {
					finalFiles = append(finalFiles, filepath.Join(rel, filepath.Base(file)))
				}
			}
		}

		return finalFiles
	}

	// Load the target type.
	targetType, ok := json["type"]
	if ok {
		target.Type = targetType.(string)
	}

	// Load the common options.
	target.Options.Srcs = loadGlobs(json, "srcs")
	target.Options.Hdrs = loadGlobs(json, "hdrs")
	target.Options.Libs = loadGlobs(json, "libs")
	target.Options.CompileFlags = loadArrayFromJson(json, "compile_flags")
	target.Options.LinkFlags = loadArrayFromJson(json, "link_flags")
	target.Options.Includes = loadArrayFromJson(json, "includes")

	// Load the platform specific options.
	ops, ok := json[runtime.GOOS]
	if ok {
		platformOptions := ops.(map[string]interface{})
		target.PlatformOptions.Srcs = loadGlobs(platformOptions, "srcs")
		target.PlatformOptions.Hdrs = loadGlobs(platformOptions, "hdrs")
		target.PlatformOptions.Libs = loadGlobs(platformOptions, "libs")
		target.PlatformOptions.CompileFlags = loadArrayFromJson(platformOptions, "compile_flags")
		target.PlatformOptions.LinkFlags = loadArrayFromJson(platformOptions, "link_flags")
		target.PlatformOptions.Includes = loadArrayFromJson(platformOptions, "includes")
	}

	return target, depSpecs
}

// Load the given target spec and all related dependencies into Target objects.
func LoadTarget(targetSpec *TargetSpec) (*Target, error) {
	// If we have already loaded this target, then return it.
	target, inCache := targetCache[targetSpec.String()]

	if !inCache {
		// Load all of the targets in the given BUILD file.
		buildFilepath := path.Join(targetSpec.Workspace, targetSpec.Path, *buildFilename)
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

		// Validate the target.
		if target.Type == "" {
			return nil, errors.New("Missing required field 'type'")
		}

		if len(target.Srcs()) == 0 && len(target.Hdrs()) == 0 {
			return nil, errors.New(fmt.Sprintf("No src/hdr files found for target %s!", target))
		}

		// Save the target to the cache.
		targetCache[targetSpec.String()] = target

		// Process dependencies for this target.
		err = target.LoadDependencies(depSpecs)
		if err != nil {
			return nil, err
		}

		log.Debugf("Loaded target %s", target)
	}

	return target, nil
}
