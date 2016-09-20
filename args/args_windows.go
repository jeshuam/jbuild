package args

import (
	"flag"
)

// All flags are defined using flag.xVar(&flag, ...). This generally makes them
// easier to use, because pointers are smelly.
var (
	// Windows specific options.
	VCVersion string
)

func init() {
	// Windows specific options.
	flag.StringVar(&VCVersion, "vc_version", "14.0",
		"(Windows Only) The Visual Studio version to use.")
}
