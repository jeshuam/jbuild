// Args is where all flags used anywhere are defined. This is useful for many
// flags which have no clear point at which they can be defined, and provides a
// simple way of adding new flags.
package args

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/client9/xson/hjson"
)

type Args struct {
	// General flags.
	DryRun bool

	// Input/Output options.
	OutputDir    string
	GenOutputDir string
	WorkspaceDir string

	// Workspace options.
	BaseWorkspaceFiles string
	WorkspaceFilename  string
	ExternalRepoKey    string
	ExternalRepoDir    string
	UpdateExternals    bool
	CleanExternalRepos bool
	BuildFilename      string

	// Display options.
	ShowLog           bool
	ShowCommands      bool
	ShowCommandEnv    bool
	UseSimpleProgress bool

	// Processing options.
	Threads       int
	Configuration string

	// Testing options.
	ForceRunTests bool
	TestRuns      uint
	TestOutput    string
	TestThreads   int

	// C++ options.
	CCCompiler string

	// Testing options.
	NoCache bool

	// Not actual arguments, but still useful.
	CurrentDir string

	// Windows options.
	VCVersion string

	// External repos that need to be loaded. Once loaded, this map should contain
	// a mapping from the local workspace path --> BUILD file text. The local path
	// must be absolute.
	ExternalRepos map[string]*ExternalRepo

	// The WORKSPACE file loaded.
	WorkspaceOptions     map[string]interface{}
	ConfigurationOptions map[string]interface{}
}

// All flags are defined using flag.xVar(&flag, ...). This generally makes them
// easier to use, because pointers are smelly.
var (
	args Args
)

func init() {
	// General flags.
	flag.BoolVar(&args.DryRun, "dry_run", false,
		"Enable DryRun mode. When DryRun is enabled, no changes will be made to "+
			"filesystem, but all target dependency resolution will be performed. This "+
			"is mostly useful for testing.")

	// Input/Output options.
	flag.StringVar(&args.OutputDir, "output_dir", "bin",
		"The output directory in which all processed files will be placed. Can be "+
			"absolute or relative to the workspace directory.")

	flag.StringVar(&args.GenOutputDir, "gen_output_dir", "gen",
		"The output directory in which all generated files will be placed. Can be "+
			"absolute or relative to the output directory.")

	flag.StringVar(&args.WorkspaceDir, "workspace_dir", "",
		"The root directory of the workspace. If blank, the directory tree will be "+
			"scanned for a WORKSPACE file. The filename searched for can be configured "+
			"using the workspace_filename flag.")

	// Workspace options.
	flag.StringVar(&args.BaseWorkspaceFiles, "base_workspace_files", "",
		"A directory from which to load base WORKSPACE files. This allows "+
			"standard configuration files (like boost) to be shipped with jbuild. "+
			"The values in the project's WORKSPACE file take precedence over there. "+
			"Files should be named XXX.workspace, where XXX signifies what is in "+
			"the file.")

	flag.StringVar(&args.WorkspaceFilename, "workspace_filename", "WORKSPACE",
		"The name of the WORKSPACE file when looking for the workspace root.")

	flag.StringVar(&args.ExternalRepoKey, "external_repo_key", "external",
		"The key in the WORKSPACE file that will be loaded as an external repo "+
			"list.")

	flag.StringVar(&args.ExternalRepoDir, "external_repo_dir", "",
		"The absolute path to the location to store external repos. If blank, defaults "+
			"to a location within the user's home directory.")

	flag.BoolVar(&args.UpdateExternals, "update_externals", false,
		"If set to true, external repositories will be updated.")

	flag.BoolVar(&args.CleanExternalRepos, "clean_external_repos", false,
		"If set to true, remove external repos when cleaning.")

	flag.StringVar(&args.BuildFilename, "build_filename", "BUILD",
		"The name of the BUILD file specifying the targets in each directory.")

	// Display options.
	flag.BoolVar(&args.ShowLog, "show_log", false,
		"If enabled, raw log messages will be shown rather than progress bars.")

	flag.BoolVar(&args.ShowCommands, "show_commands", false,
		"If enabled, the commands run will be printed to the display. These will "+
			"only be visible if show_log is also enabled.")

	flag.BoolVar(&args.UseSimpleProgress, "use_simple_progress", true,
		"If enabled, use the simple (and reliable) progress bar system.")

	// Processing options.
	flag.IntVar(&args.Threads, "threads", runtime.NumCPU(),
		"Number of threads to use while processing targets.")

	flag.StringVar(&args.Configuration, "c", "",
		"The configuration to use when building. By default, no configuration is "+
			"used (except for the common stuff).")

	// Test options.
	flag.BoolVar(&args.ForceRunTests, "force_run_tests", false,
		"If set, tests will be run even if cached results are available.")

	flag.UintVar(&args.TestRuns, "test_runs", 1,
		"The number of times to run each test.")

	flag.StringVar(&args.TestOutput, "test_output", "errors",
		"The verbosity of test output to show. 'all' means all output is shown, "+
			"'errors' (default) means only show error output and 'none' means only "+
			"show pass/fail status (no raw output).")

	flag.IntVar(&args.TestThreads, "test_threads", runtime.NumCPU(),
		"The number of threads to use when running tests. For accurate timing "+
			"results, a small number should be used. For pass/fail results, any number "+
			"can be used.")

	// C++ options.
	flag.StringVar(&args.CCCompiler, "cc_compiler", "", "The C++ compiler to use.")

	// Testing options.
	flag.BoolVar(&args.NoCache, "no_cache", false,
		"If set to true, no internal caching of any kind will be used. This is "+
			"useful for testing.")
}

// LoadConfigFile loads the BUILD specification file located at `path` and
// returns a generic key-value mapping as the result. The BUILD file is
// actually JSON, but we use hjson to make the config easier to write.
func LoadConfigFile(path string) (map[string]interface{}, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, errors.New(fmt.Sprintf("Config file not found '%s'", path))
	}

	// Load the BUILD file.
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Could not read config file '%s': %s", path, err))
	}

	configJson := make(map[string]interface{})
	err = hjson.Unmarshal(content, &configJson)
	if err != nil {
		return nil, err
	}

	return configJson, nil
}

// Get a reference to the default arguments.
func DefaultArgs() Args {
	return args
}

// Union two dictionaries together recursively. This will modify dst by merging
// in all values within src. If values in dst are multi-valued (i.e. maps or
// slices), everything in dst will be preserved. Single-value overrides will
// still take place.
func Merge(dst, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		switch v.(type) {
		case map[string]interface{}:
			_, ok := dst[k]
			if !ok {
				dst[k] = make(map[string]interface{}, 0)
			}

			dst[k] = Merge(dst[k].(map[string]interface{}), v.(map[string]interface{}))

		case []interface{}:
			_, ok := dst[k]
			if !ok {
				dst[k] = make([]interface{}, 0)
			}

			dst[k] = append(dst[k].([]interface{}), v.([]interface{})...)

		default:
			dst[k] = v
		}
	}

	return dst
}

// Load performs additional setup required to ensure the arguments are in a
// consistent format. This involves making the paths absolute, finding the
// workspace directory if necessary etc.
func Load(cwd string, customArgs *Args) (Args, error) {
	// Make a copy of the default args.
	var newArgs Args
	if customArgs != nil {
		newArgs = *customArgs
	} else {
		newArgs = args
	}

	// Get the user's home directory.
	usr, err := user.Current()
	if err != nil {
		return Args{}, err
	}

	// Load the CurrentDir flag.
	newArgs.CurrentDir = cwd

	// Load any base workspace files.
	newArgs.WorkspaceOptions = make(map[string]interface{})
	if newArgs.BaseWorkspaceFiles == "" {
		newArgs.BaseWorkspaceFiles = filepath.Join(usr.HomeDir, ".jbuild")
	} else {
		if !filepath.IsAbs(newArgs.BaseWorkspaceFiles) {
			newArgs.BaseWorkspaceFiles = filepath.Join(cwd, newArgs.BaseWorkspaceFiles)
		}
	}

	// If the base workspace files dir doesn't exist, make it.
	if exists, _ := os.Stat(newArgs.BaseWorkspaceFiles); exists == nil {
		os.MkdirAll(newArgs.BaseWorkspaceFiles, 0755)
	}

	baseWorkspaceFiles, err := ioutil.ReadDir(newArgs.BaseWorkspaceFiles)
	if err != nil {
		return Args{}, err
	}

	for _, file := range baseWorkspaceFiles {
		filePath := filepath.Join(newArgs.BaseWorkspaceFiles, file.Name())
		if strings.HasSuffix(strings.ToLower(file.Name()), ".workspace") {
			cfg, err := LoadConfigFile(filePath)
			if err != nil {
				return Args{}, err
			}

			newArgs.WorkspaceOptions = Merge(newArgs.WorkspaceOptions, cfg)
		}
	}

	// Load the WorkspaceDir flag.
	if newArgs.WorkspaceDir == "" {
		tmpCwd := newArgs.CurrentDir
		for tmpCwd != "" {
			// Check if the workspace file exists in this directory.
			workspaceFile := filepath.Join(tmpCwd, newArgs.WorkspaceFilename)
			if _, err := os.Stat(workspaceFile); err == nil {
				newArgs.WorkspaceDir = tmpCwd
				break
			}

			// Remove the last part of the path off.
			var file string
			tmpCwd, file = filepath.Split(tmpCwd)
			tmpCwd = strings.TrimRight(tmpCwd, string(os.PathSeparator))

			if file == "" {
				break
			}
		}

		// If WorkspaceDir still isn't set, then we have failed.
		if newArgs.WorkspaceDir == "" {
			return Args{}, errors.New(fmt.Sprintf(
				"Could not find WORKSPACE file '%s' anywhere above the current directory.",
				newArgs.WorkspaceFilename))
		}
	}

	// Load the ExternalRepoDir flag.
	workspaceName := filepath.Base(newArgs.WorkspaceDir)
	if newArgs.ExternalRepoDir == "" {
		newArgs.ExternalRepoDir = filepath.Join(usr.HomeDir, ".jbuild", workspaceName)
	}

	// Load the workspace file.
	workspaceFilePath := filepath.Join(newArgs.WorkspaceDir, newArgs.WorkspaceFilename)
	workspaceFileStat, _ := os.Stat(workspaceFilePath)
	if workspaceFileStat.Size() > 0 {
		loadedOptions, err := LoadConfigFile(workspaceFilePath)
		if err != nil {
			return Args{}, err
		}

		newArgs.WorkspaceOptions = Merge(newArgs.WorkspaceOptions, loadedOptions)
	}

	// Load any additional dependencies (e.g. from github).
	newArgs.ExternalRepos = make(map[string]*ExternalRepo)
	externalRepos, ok := newArgs.WorkspaceOptions[newArgs.ExternalRepoKey]
	if ok {
		for repoPath, repoJson := range externalRepos.(map[string]interface{}) {
			// Load some basic information about the external repo.
			externalRepo, err := MakeExternalRepo(repoPath, repoJson.(map[string]interface{}))
			if err != nil {
				return Args{}, err
			}

			newArgs.ExternalRepos[repoPath] = externalRepo
		}
	}

	// Load OS specific options.
	workspaceOptions, ok := newArgs.WorkspaceOptions[runtime.GOOS]
	if ok {
		newArgs.WorkspaceOptions = Merge(
			newArgs.WorkspaceOptions, workspaceOptions.(map[string]interface{}))
		configurationOptions, ok := newArgs.WorkspaceOptions[newArgs.Configuration]
		if ok {
			newArgs.ConfigurationOptions = configurationOptions.(map[string]interface{})
		}
	}

	// Load OutputDir based on WorkspaceDir.
	if !filepath.IsAbs(newArgs.OutputDir) {
		newArgs.OutputDir = filepath.Join(newArgs.WorkspaceDir, newArgs.OutputDir)
		if newArgs.Configuration != "" {
			newArgs.OutputDir = filepath.Join(newArgs.OutputDir, newArgs.Configuration)
		}
	}

	// Load GenOutputDir based on OutputDir.
	if !filepath.IsAbs(newArgs.GenOutputDir) {
		newArgs.GenOutputDir = filepath.Join(newArgs.OutputDir, newArgs.GenOutputDir)
	}

	// Load the C++ compiler.
	if newArgs.CCCompiler == "" {
		if runtime.GOOS == "windows" {
			newArgs.CCCompiler = "cl.exe"
		} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
			newArgs.CCCompiler = "clang++"
		} else {
			return Args{}, errors.New(
				fmt.Sprintf("Could not set C++ compiler: unknown OS %s", runtime.GOOS))
		}
	}

	return newArgs, nil
}
