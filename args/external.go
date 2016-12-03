package args

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// An ExternalRepo structure, which contains all information required to build
// and checkout an external repo.
type ExternalRepo struct {
	// The path this external repo should be presented as. Must be unique.
	Path string

	// The location of the external repo on the filesystem. Only populated after
	// LoadExternalRepo is called.
	FsDir string

	// The URL to the git repository that contains the code. Should be https://
	// for maximum compatability.
	Url string

	// The branch to checkout. This is anything that can be passed to the -b flag
	// when checking out the code (e.g. a tag, branch). If blank, uses master.
	Branch string

	// The build instructions needed to build this external repo. Can either be
	// the raw BUILD contents or a filepath (relative to the workspace root).
	Build     map[string]interface{}
	BuildFile string
}

// MakeExternalRepo from a JSON map.
func MakeExternalRepo(path string, repoJson map[string]interface{}) (*ExternalRepo, error) {
	var url, branch, buildFile string
	var build map[string]interface{}
	var err error

	// Get the objects from the JSON.
	urlInt, urlOk := repoJson["url"]
	branchInt, branchOk := repoJson["branch"]
	buildInt, buildOk := repoJson["build"]

	if urlOk {
		url = urlInt.(string)
	} else {
		return nil, errors.New("A URL must be specified for all external repos.")
	}

	if branchOk {
		branch = branchInt.(string)
	} else {
		branch = "master"
	}

	if buildOk {
		switch buildInt.(type) {
		case string:
			build, err = LoadConfigFile(buildInt.(string))
			buildFile = buildInt.(string)
			if err != nil {
				return nil, err
			}

		case map[string]interface{}:
			build = buildInt.(map[string]interface{})
			buildFile = ""
		}
	} else {
		// Assume it must be present in the external repo.
		build = nil
		buildFile = ""
	}

	// Make and return the external repo.
	externalRepo := new(ExternalRepo)
	externalRepo.Path = path
	externalRepo.Url = url
	externalRepo.Branch = branch
	externalRepo.Build = build
	externalRepo.BuildFile = buildFile
	return externalRepo, nil
}

func fetchGit(args *Args, repo *ExternalRepo) error {
	// If the directory doesn't exist, then clone.
	gitDir := filepath.Join(args.ExternalRepoDir, strings.Trim(repo.Path, "/"))
	repo.FsDir = gitDir
	if _, err := os.Stat(gitDir); err != nil {
		// Build the git command.
		cmd := exec.Command("git", "clone", "--recurse-submodules", "-b", repo.Branch, repo.Url, gitDir)

		// Save the command output.
		cmd.Stdout = os.Stdout
		// cmd.Stderr = os.Stderr

		// Clone the repository.
		fmt.Printf("Cloning into %s...\n", repo.Url)
		err := cmd.Run()
		if err != nil {
			return err
		}
	} else if args.UpdateExternals {
		// Otherwise, update the git repo and checkout the branch.
		gitPull := exec.Command("git", "pull", "origin", repo.Branch)
		gitPull.Dir = gitDir

		// Save the command output.
		gitPull.Stdout = os.Stdout
		gitPull.Stderr = os.Stderr

		// Clone the repository.
		fmt.Printf("Updating %s...\n", repo.Url)
		err := gitPull.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadExternalRepo will load the external repository specified by `repo`,
// download it and load the corresponding BUILD file.
func LoadExternalRepo(args *Args, repo *ExternalRepo) error {
	// Fetch the repo.
	err := fetchGit(args, repo)
	if err != nil {
		return err
	}

	return nil
}
