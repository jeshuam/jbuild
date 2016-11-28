package args

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ExternalRepo struct {
	Name   string
	Type   string
	Url    string
	Branch string
	Build  string
	Dir    string

	BuildFile map[string]interface{}
	RepoDir   string
}

func makeExternalRepoStruct(name string, repo map[string]interface{}) ExternalRepo {
	r := ExternalRepo{}
	r.Name = name
	r.Type = repo["type"].(string)
	r.Url = repo["url"].(string)
	r.Branch = repo["branch"].(string)
	r.Build = repo["build"].(string)
	r.Dir = repo["dir"].(string)
	return r
}

func fetchGit(args Args, repo ExternalRepo) error {
	// If the directory doesn't exist, then clone.
	gitDir := filepath.Join(repo.RepoDir, strings.Trim(repo.Dir, "/"))
	if _, err := os.Stat(gitDir); err != nil {
		// Build the git command.
		cmd := exec.Command("git", "clone", "-b", repo.Branch, repo.Url, gitDir)

		// Save the command output.
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Clone the repository.
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
func LoadExternalRepo(args Args, repoName string, repoJson map[string]interface{}) (ExternalRepo, error) {
	repo := makeExternalRepoStruct(repoName, repoJson)

	// Checkout the repository.
	var err error
	repo.RepoDir = args.ExternalRepoDir
	if repo.Type == "git" {
		err = fetchGit(args, repo)
	} else {
		return ExternalRepo{}, errors.New(fmt.Sprintf("Unknown repo type %s", repo.Type))
	}

	// Check for errors.
	if err != nil {
		return ExternalRepo{}, err
	}

	// Load the BUILD file.
	repo.BuildFile, err = LoadConfigFile(filepath.Join(args.WorkspaceDir, repo.Build))
	if err != nil {
		return ExternalRepo{}, err
	}

	return repo, nil
}
