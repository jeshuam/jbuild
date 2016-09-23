// Args is where all flags used anywhere are defined. This is useful for many
// flags which have no clear point at which they can be defined, and provides a
// simple way of adding new flags.
package args

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Args struct {
	// General flags.
	DryRun bool

	// Input/Output options.
	OutputDir    string
	WorkspaceDir string

	// Workspace options.
	WorkspaceFilename string
	BuildFilename     string

	// Display options.
	ShowLog           bool
	ShowCommands      bool
	UseSimpleProgress bool

	// Processing options.
	Threads int

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

	flag.StringVar(&args.WorkspaceDir, "workspace_dir", "",
		"The root directory of the workspace. If blank, the directory tree will be "+
			"scanned for a WORKSPACE file. The filename searched for can be configured "+
			"using the workspace_filename flag.")

	// Workspace options.
	flag.StringVar(&args.WorkspaceFilename, "workspace_filename", "WORKSPACE",
		"The name of the WORKSPACE file when looking for the workspace root.")

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

// Load performs additional setup required to ensure the arguments are in a
// consistent format. This involves making the paths absolute, finding the
// workspace directory if necessary etc.
func Load(cwd string) (Args, error) {
	// Make a copy of the default args.
	newArgs := args

	// Load the CurrentDir flag.
	newArgs.CurrentDir = cwd

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
			tmpCwd = strings.Trim(tmpCwd, string(os.PathSeparator))

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

	// Load OutputDir based on WorkspaceDir.
	if !filepath.IsAbs(newArgs.OutputDir) {
		newArgs.OutputDir = filepath.Join(newArgs.WorkspaceDir, newArgs.OutputDir)
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
