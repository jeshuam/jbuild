package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jeshuam/jbuild/config/interfaces"
)

var (
	// A cache from file spec strings --> targets. This will allow the same target
	// to be loaded multiple times with no performance degradations.
	TargetCache = make(map[string]interfaces.Target, 0)

	// A cache of file spec --> file spec object.
	SpecCache = make(map[string]interfaces.Spec, 0)
)

func checkForDependencyCyclesRecurse(
	spec interfaces.TargetSpec, visited []string, seq int) error {
	// If this node has already been visited in the current recursive stack, then
	// there must be a cycle in the graph.
	for i := 0; i < seq; i++ {
		if visited[i] == spec.String() {
			cycle := strings.Join(visited[i:seq], " --> ")
			return errors.New(fmt.Sprintf("Cycle detected: %s --> %s", cycle, spec))
		}
	}

	// Add this item to sequence number `seq`. If the array is already big enough,
	// then just replace what is in there. Otherwise, add a new element.
	if len(visited) > seq {
		visited[seq] = spec.String()
	} else {
		visited = append(visited, spec.String())
	}

	// Look through each child of the current target and recurse to it, first
	// adding this node to the visited list.
	for _, dep := range spec.Dependencies(false) {
		// We can stop if the dep doesn't need processing.
		if dep.Target().TotalOps() == 0 {
			continue
		}

		err := checkForDependencyCyclesRecurse(dep, visited, seq+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func CheckForDependencyCycles(spec interfaces.TargetSpec) error {
	visited := make([]string, 0, 1)
	visited = append(visited, spec.String())
	for _, dep := range spec.Dependencies(false) {
		err := checkForDependencyCyclesRecurse(dep, visited, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func OSPathToWSPath(path string) string {
	return "//" + strings.Trim(
		strings.Replace(path, string(os.PathSeparator), "/", -1), "/")
}

func MakeUnique(args []string) []string {
	found := make(map[string]bool, len(args))
	output := make([]string, 0, len(args))
	for _, arg := range args {
		_, ok := found[arg]
		if !ok {
			found[arg] = true
			output = append(output, arg)
		}
	}

	return output
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}

	return dstFile.Close()
}
