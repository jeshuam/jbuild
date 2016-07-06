package common

import (
	"flag"
)

var (
	OutputDirectory string
	DryRun          bool
)

func init() {
	flag.StringVar(&OutputDirectory, "output_dir", "bin", "The output directory in which all processed files will be placed.")
	flag.BoolVar(&DryRun, "dry_run", false, "Don't actually compile anything, just say what is going to happen.")
}
