package main

import (
	"flag"
	"os"

	"github.com/jeshuam/jbuild/args"
	"github.com/jeshuam/jbuild/jbuild"
	"github.com/op/go-logging"
)

func main() {
	log := logging.MustGetLogger("jbuild")

	// Load arguments.
	flag.Parse()

	// Get the current dir.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	// Load flags.
	programArgs, err := args.Load(cwd, nil)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	if err := jbuild.JBuildRun(programArgs, flag.Args()); err != nil {
		log.Fatalf("Error: %s", err)
	}
}
